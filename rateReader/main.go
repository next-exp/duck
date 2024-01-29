package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/jmbenlloch/next_duck/pkg/database"
)

func connectToDatabase(dbConf duck.DatabaseConfiguration) (*sql.DB, error) {
	dbURI := fmt.Sprintf("%s:%s@(%s:%s)/%s?parseTime=true", dbConf.Username,
		dbConf.Password, dbConf.Hostname, dbConf.Port, dbConf.Database)
	db, err := sql.Open("mysql", dbURI)
	return db, err
}

var (
	totalBytes     int64
	totalEvents    int64
	totalByteRate  float64
	totalEventRate float64
)

type RunTimestamp struct {
	Run            int       `db:"id"`
	Timestamp      time.Time `db:"start"`
	Bytes          int64     `db:"bytes"`
	Events         int64     `db:"events"`
	AvgTriggerRate float64   `db:"avgTriggerRate"`
	AvgByteRate    float64   `db:"avgByteRate"`
}

func getRun(db *sql.DB) (int, error) {
	queries := database.New(db)
	run, err := queries.GetLatestRun(context.Background())
	return int(run), err
}

func writeRateData(db *sql.DB) error {
	queries := database.New(db)
	ctx := context.Background()

	run, err := getRun(db)
	if err != nil {
		fmt.Printf("Error getting run: %v\n", err)
		return err
	}

	timestamp := time.Now()

	// Check if rate record exists for this run
	_, err = queries.GetRateByRun(ctx, int32(run))
	if err != nil {
		if err == sql.ErrNoRows {
			// Insert new rate record
			err = queries.InsertRate(ctx, database.InsertRateParams{
				Run:            int32(run),
				LastUpdate:     sql.NullTime{Time: timestamp, Valid: true},
				Bytes:          sql.NullInt64{Int64: totalBytes, Valid: true},
				Events:         sql.NullInt64{Int64: totalEvents, Valid: true},
				Avgtriggerrate: sql.NullFloat64{Float64: totalEventRate, Valid: true},
				Avgbyterate:    sql.NullFloat64{Float64: totalByteRate, Valid: true},
			})
			if err != nil {
				fmt.Printf("Error inserting rate data: %v\n", err)
			}
		} else {
			fmt.Printf("Error checking rate data: %v\n", err)
		}
		return err
	}

	// Update existing rate record
	err = queries.UpdateRate(ctx, database.UpdateRateParams{
		LastUpdate:     sql.NullTime{Time: timestamp, Valid: true},
		Bytes:          sql.NullInt64{Int64: totalBytes, Valid: true},
		Events:         sql.NullInt64{Int64: totalEvents, Valid: true},
		Avgtriggerrate: sql.NullFloat64{Float64: totalEventRate, Valid: true},
		Avgbyterate:    sql.NullFloat64{Float64: totalByteRate, Valid: true},
		Run:            int32(run),
	})
	if err != nil {
		fmt.Printf("Error updating rate data: %v\n", err)
	}
	return err
}

func updateDatabase(dbConfiguration duck.DatabaseConfiguration, period int, verbose bool) {
	ticker := time.NewTicker(time.Duration(period) * time.Second)
	defer ticker.Stop()

	dbConn, err := connectToDatabase(dbConfiguration)
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		return
	}
	defer dbConn.Close()

	for range ticker.C {
		if verbose {
			fmt.Printf("Updating database with totalBytes: %d, totalEvents: %d, totalByteRate: %.2f, totalEventRate: %.2f\n",
				totalBytes, totalEvents, totalByteRate, totalEventRate)
		}

		// Add database update logic here
		err = writeRateData(dbConn)
		if err != nil {
			fmt.Printf("Reconnecting to database due to error: %v\n", err)
			dbConn.Close()
			dbConn, err = connectToDatabase(dbConfiguration)
			if err != nil {
				fmt.Printf("Error reconnecting to database: %v\n", err)
			}
		}
	}
}

func main() {
	configFilename := flag.String("config", "", "Configuration file path")
	period := flag.Int("period", 5, "Period to update database in seconds")
	verbose := flag.Bool("verbose", false, "Verbose output")
	flag.Parse()

	configuration, err := duck.ReadConfigurationFile(*configFilename)
	if err != nil {
		panic(err)
	}

	bytes := make(map[string]int64)
	events := make(map[string]int64)
	byteRate := make(map[string]float64)
	eventRate := make(map[string]float64)

	userCentrifugal := "rateReader"
	fn := func(message *duck.Message) {
		if message.Type == duck.MessageMetric {
			var metric duck.Metrics
			err := json.Unmarshal([]byte(message.Value), &metric)
			if err != nil {
				fmt.Printf("Error unmarshalling message value: %v\n", err)
				return
			}

			// Use only GDC data
			if !strings.HasPrefix(message.Host, "gdc") {
				return
			}

			bytes[message.Host] = int64(metric.ByteCounter)
			events[message.Host] = int64(metric.EventCounter)
			byteRate[message.Host] = metric.AvgDataRate
			eventRate[message.Host] = metric.AvgTrgRate

			// Update total counters
			totalBytes = 0
			for _, v := range bytes {
				totalBytes += v
			}

			totalEvents = 0
			for _, v := range events {
				totalEvents += v
			}

			totalByteRate = 0
			for _, v := range byteRate {
				totalByteRate += v
			}

			totalEventRate = 0
			for _, v := range eventRate {
				totalEventRate += v
			}
		}
	}
	_, err = duck.CreateNewSubscriptionWithFnReadout(configuration.Centrifugal, userCentrifugal, fn)
	if err != nil {
		panic(err)
	}

	updateDatabase(configuration.Database, *period, *verbose)
}
