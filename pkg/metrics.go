package duck

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// TickerProvider provides tickers for testing
type TickerProvider interface {
	NewTicker(d time.Duration) *time.Ticker
}

// RealTicker uses actual time (production implementation)
type RealTicker struct{}

func (t RealTicker) NewTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}

type Measurement struct {
	Timestamp  time.Time
	DataSize   int64
	EvtCounter int64
}

func ProcessMetrics(metrics chan int, logger DuckLogger,
	eventCounter prometheus.Gauge, sizeCounter prometheus.Gauge, ctx context.Context,
	tickerProvider TickerProvider) {
	ticker := tickerProvider.NewTicker(1000 * time.Millisecond)
	totalSize := 0
	evtCounter := 0

	avgTrgRate := 0.
	avgDataRate := 0.
	currentTrgRate := 0.
	currentDataRate := 0.

	startTime := time.Now()
	latestMeasurements := make([]Measurement, 0)

	for {
		select {
		case <-ctx.Done():
			logger.Slog.Debug("finish process metrics routine")
			return
		case tick := <-ticker.C:
			elapsedTime := tick.Sub(startTime).Seconds()
			if elapsedTime > 0 {
				avgDataRate = float64(totalSize) / elapsedTime
				avgTrgRate = float64(evtCounter) / elapsedTime
			}

			// Keep only the latest 5 measurements for current rates
			latestMeasurements = append(latestMeasurements, Measurement{
				Timestamp:  tick,
				DataSize:   int64(totalSize),
				EvtCounter: int64(evtCounter),
			})
			if len(latestMeasurements) > 5 {
				latestMeasurements = latestMeasurements[1:]
			}
			elapsedTimeCurrent := tick.Sub(latestMeasurements[0].Timestamp).Seconds()
			firstMeasurement := latestMeasurements[0]
			lastMeasurement := latestMeasurements[len(latestMeasurements)-1]
			if elapsedTimeCurrent > 0 {
				currentDataRate = float64(lastMeasurement.DataSize-firstMeasurement.DataSize) / elapsedTimeCurrent
				currentTrgRate = float64(lastMeasurement.EvtCounter-firstMeasurement.EvtCounter) / elapsedTimeCurrent
			}

			metricData := Metrics{
				EventCounter:    evtCounter,
				ByteCounter:     totalSize,
				CurrentTrgRate:  currentTrgRate,
				CurrentDataRate: currentDataRate,
				AvgTrgRate:      avgTrgRate,
				AvgDataRate:     avgDataRate,
			}
			message, err := json.Marshal(metricData)
			if err != nil {
				message := fmt.Sprintf("Error marshalling metrics %s", err.Error())
				logger.Slog.Error(message)
				continue
			}
			logger.Metric(string(message))
		case size := <-metrics:
			if size < 0 {
				// Log warning about negative value
				logger.Slog.Log(ctx, slog.LevelWarn, fmt.Sprintf("Ignoring negative metric size: %d", size))
				continue
			}
			eventCounter.Inc()
			sizeCounter.Add(float64(size))
			evtCounter++
			totalSize += size
		}
	}
}
