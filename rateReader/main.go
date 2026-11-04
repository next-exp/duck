package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-sql-driver/mysql"
	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/jmbenlloch/next_duck/pkg/database"
)

const maxPendingSamples = 4096

type sample struct {
	run       int32
	host      string
	timestamp time.Time
	metric    duck.Metrics
	reset     bool
}

// All mutable subscription state is protected; DB writes use immutable snapshots.
type sampleKey struct {
	run  int32
	host string
}

type collector struct {
	mu        sync.Mutex
	period    time.Duration
	latest    map[string]sample
	selected  map[sampleKey]sample
	summaries map[sampleKey]sample
	pending   []sample
}

func newCollector(period time.Duration) *collector {
	return &collector{period: period, latest: make(map[string]sample), selected: make(map[sampleKey]sample), summaries: make(map[sampleKey]sample)}
}

func (c *collector) receive(message *duck.Message) error {
	if message.Type != duck.MessageMetric || !strings.HasPrefix(message.Host, "gdc") {
		return nil
	}
	if message.RunNumber <= 0 || message.Timestamp.IsZero() {
		return fmt.Errorf("GDC metric lacks a valid run or source timestamp")
	}
	if int64(message.RunNumber) > math.MaxInt32 || len(message.Host) > 100 {
		return fmt.Errorf("GDC metric identity exceeds database limits")
	}
	var metric duck.Metrics
	if err := json.Unmarshal([]byte(message.Value), &metric); err != nil {
		return err
	}
	if metric.EventCounter < 0 || metric.ByteCounter < 0 {
		return fmt.Errorf("negative GDC counter")
	}
	for _, rate := range []float64{metric.CurrentTrgRate, metric.CurrentDataRate, metric.AvgTrgRate, metric.AvgDataRate} {
		if rate < 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
			return fmt.Errorf("invalid GDC rate")
		}
	}
	s := sample{run: int32(message.RunNumber), host: message.Host, timestamp: message.Timestamp.UTC().Truncate(time.Microsecond), metric: metric}
	c.mu.Lock()
	defer c.mu.Unlock()
	previous, exists := c.latest[s.host]
	if exists && (s.run < previous.run || !s.timestamp.After(previous.timestamp)) {
		return nil
	}
	s.reset = exists && s.run == previous.run && (metric.EventCounter < previous.metric.EventCounter || metric.ByteCounter < previous.metric.ByteCounter)
	c.latest[s.host] = s
	key := sampleKey{s.run, s.host}
	c.summaries[key] = s
	last, selected := c.selected[key]
	if selected && last.run == s.run && !s.reset && s.timestamp.Sub(last.timestamp) < c.period {
		return nil
	}
	if len(c.pending) >= maxPendingSamples {
		return fmt.Errorf("rate history queue full: dropping new sample for run %d GDC %s", s.run, s.host)
	}
	c.pending = append(c.pending, s)
	c.selected[key] = s
	return nil
}

func connectToDatabase(conf duck.DatabaseConfiguration) (*sql.DB, error) {
	cfg := mysql.NewConfig()
	cfg.User, cfg.Passwd = conf.Username, conf.Password
	cfg.Net, cfg.Addr, cfg.DBName = "tcp", net.JoinHostPort(conf.Hostname, conf.Port), conf.Database
	cfg.ParseTime, cfg.Loc = true, time.UTC
	cfg.Params = map[string]string{"time_zone": "'+00:00'"}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 5*time.Second, 5*time.Second
	return sql.Open("mysql", cfg.FormatDSN())
}

func (c *collector) flush(ctx context.Context, db *sql.DB) error {
	c.mu.Lock()
	pending := append([]sample(nil), c.pending...)
	latest := make([]sample, 0, len(c.summaries))
	for _, s := range c.summaries {
		latest = append(latest, s)
	}
	c.mu.Unlock()
	if len(pending) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := database.New(tx)
	for _, s := range pending {
		if err := q.InsertRateSample(ctx, database.InsertRateSampleParams{
			Run: s.run, Gdc: s.host, SampleTime: s.timestamp,
			Events: uint64(s.metric.EventCounter), Bytes: uint64(s.metric.ByteCounter),
			CurrentTriggerRate: s.metric.CurrentTrgRate, CurrentByteRate: s.metric.CurrentDataRate,
			AvgTriggerRate: s.metric.AvgTrgRate, AvgByteRate: s.metric.AvgDataRate, CounterReset: s.reset,
		}); err != nil {
			return err
		}
	}
	// Keep the existing run summaries, grouping hosts by their actual message run.
	totals := make(map[int32]sample)
	stale := make(map[int32]bool)
	now := time.Now()
	for _, s := range latest {
		if now.Sub(s.timestamp) > 3*c.period {
			stale[s.run] = true
		}
		t := totals[s.run]
		if s.timestamp.After(t.timestamp) {
			t.timestamp = s.timestamp
		}
		t.metric.EventCounter += s.metric.EventCounter
		t.metric.ByteCounter += s.metric.ByteCounter
		t.metric.AvgTrgRate += s.metric.AvgTrgRate
		t.metric.AvgDataRate += s.metric.AvgDataRate
		totals[s.run] = t
	}
	for run, t := range totals {
		if stale[run] {
			continue
		}
		_, err := q.GetRateByRun(ctx, run)
		switch err {
		case sql.ErrNoRows:
			err = q.InsertRate(ctx, database.InsertRateParams{Run: run, LastUpdate: sql.NullTime{Time: t.timestamp, Valid: true}, Bytes: sql.NullInt64{Int64: int64(t.metric.ByteCounter), Valid: true}, Events: sql.NullInt64{Int64: int64(t.metric.EventCounter), Valid: true}, Avgtriggerrate: sql.NullFloat64{Float64: t.metric.AvgTrgRate, Valid: true}, Avgbyterate: sql.NullFloat64{Float64: t.metric.AvgDataRate, Valid: true}})
		case nil:
			err = q.UpdateRate(ctx, database.UpdateRateParams{Run: run, LastUpdate: sql.NullTime{Time: t.timestamp, Valid: true}, Bytes: sql.NullInt64{Int64: int64(t.metric.ByteCounter), Valid: true}, Events: sql.NullInt64{Int64: int64(t.metric.EventCounter), Valid: true}, Avgtriggerrate: sql.NullFloat64{Float64: t.metric.AvgTrgRate, Valid: true}, Avgbyterate: sql.NullFloat64{Float64: t.metric.AvgDataRate, Valid: true}})
		}
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	c.mu.Lock()
	c.pending = c.pending[len(pending):]
	// Retire completed runs only after their final queued samples were committed.
	activeRuns := make(map[int32]bool)
	for _, s := range c.latest {
		activeRuns[s.run] = true
	}
	for _, s := range c.pending {
		activeRuns[s.run] = true
	}
	for key := range c.summaries {
		if !activeRuns[key.run] {
			delete(c.summaries, key)
			delete(c.selected, key)
		}
	}
	c.mu.Unlock()
	return nil
}

func main() {
	configFilename := flag.String("config", "", "Configuration file path")
	period := flag.Int("period", 5, "Minimum source-time spacing between stored GDC samples, in seconds")
	verbose := flag.Bool("verbose", false, "Verbose output")
	flag.Parse()
	if *period <= 0 {
		log.Fatal("period must be positive")
	}
	configuration, err := duck.ReadConfigurationFile(*configFilename)
	if err != nil {
		log.Fatal(err)
	}
	db, err := connectToDatabase(configuration.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}
	// Fail before subscribing if the additive history migration has not been applied.
	if _, err := db.ExecContext(ctx, "SELECT run FROM rateSamples LIMIT 0"); err != nil {
		log.Fatalf("rateSamples unavailable; apply database/migrations/20261104_add_rate_samples.sql: %v", err)
	}
	c := newCollector(time.Duration(*period) * time.Second)
	_, err = duck.CreateNewSubscriptionWithFnReadout(configuration.Centrifugal, "rateReader", func(m *duck.Message) {
		if err := c.receive(m); err != nil {
			log.Printf("metric rejected: %v", err)
		}
	})
	if err != nil {
		log.Fatal(err)
	}
	ticker := time.NewTicker(c.period)
	defer ticker.Stop()
	flush := func() {
		writeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := c.flush(writeCtx, db); err != nil {
			log.Printf("database write failed; retaining samples for retry: %v", err)
		} else if *verbose {
			log.Print("rate samples flushed")
		}
	}
	for {
		select {
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		}
	}
}
