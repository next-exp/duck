package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

func metricMessage(run int, host string, at time.Time, events int) *duck.Message {
	data, _ := json.Marshal(duck.Metrics{EventCounter: events, ByteCounter: events * 100, CurrentTrgRate: 12, CurrentDataRate: 1200, AvgTrgRate: 10, AvgDataRate: 1000})
	return &duck.Message{Type: duck.MessageMetric, RunNumber: run, Host: host, Timestamp: at, Value: string(data)}
}

func TestSamplingRunTransitionsAndReset(t *testing.T) {
	c := newCollector(5 * time.Second)
	at := time.Now().UTC().Truncate(time.Microsecond)
	for _, m := range []*duck.Message{
		metricMessage(10, "gdc1", at, 100),
		metricMessage(10, "gdc1", at.Add(time.Second), 110), // below sampling period
		metricMessage(10, "gdc1", at, 100),                  // duplicate
		metricMessage(10, "gdc1", at.Add(5*time.Second), 150),
		metricMessage(10, "gdc1", at.Add(6*time.Second), 2),   // reset must be captured immediately
		metricMessage(11, "gdc1", at.Add(7*time.Second), 1),   // new run does not wait five seconds
		metricMessage(10, "gdc1", at.Add(8*time.Second), 200), // delayed old run
	} {
		if err := c.receive(m); err != nil {
			t.Fatal(err)
		}
	}
	if len(c.pending) != 4 {
		t.Fatalf("samples: %+v", c.pending)
	}
	if !c.pending[2].reset || c.pending[3].reset {
		t.Fatal("incorrect reset flags")
	}
	if c.pending[3].run != 11 {
		t.Fatal("run attribution lost")
	}
	if c.pending[0].metric.CurrentTrgRate != 12 || c.pending[0].metric.CurrentDataRate != 1200 {
		t.Fatal("current values lost")
	}
	if !c.pending[0].timestamp.Equal(at) {
		t.Fatal("source timestamp lost")
	}
	if len(c.summaries) != 2 {
		t.Fatal("previous run summary lost before flush")
	}
}

func TestMissingInvalidAndNonGDCMetrics(t *testing.T) {
	at := time.Now()
	for _, alter := range []func(*duck.Message){
		func(m *duck.Message) { m.RunNumber = 0 },
		func(m *duck.Message) { m.Timestamp = time.Time{} },
		func(m *duck.Message) { m.Value = `{"EventCounter":-1}` },
		func(m *duck.Message) { m.Value = `{"CurrentTrgRate":-1}` },
		func(m *duck.Message) { m.Value = `broken` },
	} {
		c := newCollector(5 * time.Second)
		m := metricMessage(1, "gdc1", at, 10)
		alter(m)
		if c.receive(m) == nil || len(c.pending) != 0 {
			t.Fatal("accepted invalid metric")
		}
	}
	c := newCollector(5 * time.Second)
	if err := c.receive(metricMessage(1, "ldc1", at, 10)); err != nil || len(c.pending) != 0 {
		t.Fatal("accepted LDC metric")
	}
}

func TestQueueBoundAndConcurrentSnapshots(t *testing.T) {
	c := newCollector(time.Microsecond)
	at := time.Now()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < 100; i++ {
		mock.ExpectBegin().WillReturnError(fmt.Errorf("offline"))
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = c.flush(context.Background(), db)
		}
	}()
	for host := 0; host < 4; host++ {
		wg.Add(1)
		go func(host int) {
			defer wg.Done()
			for i := 0; i < 1100; i++ {
				_ = c.receive(metricMessage(1, fmt.Sprintf("gdc%d", host), at.Add(time.Duration(i)*time.Millisecond), i))
			}
		}(host)
	}
	wg.Wait()
	if len(c.pending) != maxPendingSamples {
		t.Fatalf("unbounded queue: %d", len(c.pending))
	}
	if err := c.receive(metricMessage(1, "gdc-extra", at, 1)); err == nil {
		t.Fatal("overflow must be reported")
	}
}

func expectHistory(mock sqlmock.Sqlmock) {
	mock.ExpectExec("INSERT INTO rateSamples").WithArgs(int32(10), "gdc1", sqlmock.AnyArg(), uint64(100), uint64(10000), float64(12), float64(1200), float64(10), float64(1000), false).WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestFlushRetryAndSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	c := newCollector(5 * time.Second)
	if err := c.receive(metricMessage(10, "gdc1", time.Now(), 100)); err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO rateSamples").WillReturnError(fmt.Errorf("temporary database failure"))
	mock.ExpectRollback()
	if c.flush(context.Background(), db) == nil || len(c.pending) != 1 {
		t.Fatal("failure discarded sample")
	}
	mock.ExpectBegin()
	expectHistory(mock)
	mock.ExpectQuery("SELECT run FROM rates").WithArgs(int32(10)).WillReturnRows(sqlmock.NewRows([]string{"run"}))
	mock.ExpectExec("INSERT INTO rates").WithArgs(int32(10), sqlmock.AnyArg(), int64(10000), int64(100), float64(10), float64(1000)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := c.flush(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if len(c.pending) != 0 {
		t.Fatal("committed samples retained")
	}
	// No incoming metrics: do not create fresh-looking samples or rewrite summaries.
	if err := c.flush(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStaleMetricHistoryDoesNotRewriteSummary(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	c := newCollector(5 * time.Second)
	_ = c.receive(metricMessage(10, "gdc1", time.Now().Add(-time.Minute), 100))
	mock.ExpectBegin()
	expectHistory(mock)
	mock.ExpectCommit()
	if err := c.flush(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Set RATE_READER_TEST_DSN to an isolated, disposable MySQL database.
func TestMySQLHistoryAndRunSummaries(t *testing.T) {
	dsn := os.Getenv("RATE_READER_TEST_DSN")
	if dsn == "" {
		t.Skip("RATE_READER_TEST_DSN not set")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	migration, err := os.ReadFile("../database/migrations/20261104_add_rate_samples.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(migration)); err != nil {
		t.Fatalf("migration not idempotent: %v", err)
	}
	if _, err = db.Exec(`CREATE TABLE rates (run INT NOT NULL, last_update TIMESTAMP NULL, avgByteRate FLOAT NULL, avgTriggerRate FLOAT NULL, bytes BIGINT UNSIGNED NULL, events BIGINT UNSIGNED NULL)`); err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP TABLE rates, rateSamples")
	at := time.Now().UTC().Truncate(time.Microsecond)
	c := newCollector(5 * time.Second)
	_ = c.receive(metricMessage(10, "gdc1", at, 100))
	_ = c.receive(metricMessage(10, "gdc2", at, 200))
	duplicate := c.pending[0]
	if err := c.flush(ctx, db); err != nil {
		t.Fatal(err)
	}
	c.pending = append(c.pending, duplicate) // model uncertain-commit retry
	if err := c.flush(ctx, db); err != nil {
		t.Fatal(err)
	}
	// A run/time index must exist in both the migration and fresh-schema path.
	var indexedColumns string
	if err := db.QueryRow(`SELECT GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='rateSamples' AND index_name='rateSamples_run_time'`).Scan(&indexedColumns); err != nil || indexedColumns != "run,sample_time" {
		t.Fatalf("missing run/time index: %q %v", indexedColumns, err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM rateSamples WHERE run=10").Scan(&count); err != nil || count != 2 {
		t.Fatalf("duplicate history rows: %d %v", count, err)
	}
	_ = c.receive(metricMessage(11, "gdc1", at.Add(time.Second), 5))
	if err := c.flush(ctx, db); err != nil {
		t.Fatal(err)
	}
	var events int
	if err := db.QueryRow("SELECT events FROM rates WHERE run=10").Scan(&events); err != nil || events != 300 {
		t.Fatalf("old run summary corrupted: %d %v", events, err)
	}
	if err := db.QueryRow("SELECT events FROM rates WHERE run=11").Scan(&events); err != nil || events != 5 {
		t.Fatalf("new run summary wrong: %d %v", events, err)
	}
	var currentRate, avgRate float64
	var timestamp time.Time
	if err := db.QueryRow("SELECT sample_time,current_trigger_rate,avg_trigger_rate FROM rateSamples WHERE run=10 AND gdc='gdc1'").Scan(&timestamp, &currentRate, &avgRate); err != nil {
		t.Fatal(err)
	}
	if !timestamp.Equal(at) || currentRate != 12 || avgRate != 10 {
		t.Fatalf("stored values differ: %v %v %v", timestamp, currentRate, avgRate)
	}
}

func TestCommitFailureRetainsSamples(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	c := newCollector(5 * time.Second)
	_ = c.receive(metricMessage(10, "gdc1", time.Now().Add(-time.Minute), 100))
	mock.ExpectBegin()
	expectHistory(mock)
	mock.ExpectCommit().WillReturnError(fmt.Errorf("connection lost during commit"))
	if c.flush(context.Background(), db) == nil || len(c.pending) != 1 {
		t.Fatal("uncertain commit lost sample")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
