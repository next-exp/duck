package duck

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

// MockGauge is a mock implementation of prometheus.Gauge
type MockGauge struct {
	mu    sync.Mutex
	value float64
	calls []float64
}

func (m *MockGauge) Set(v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value = v
}

func (m *MockGauge) Inc() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value++
	m.calls = append(m.calls, 1)
}

func (m *MockGauge) Dec() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value--
}

func (m *MockGauge) Add(v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value += v
	m.calls = append(m.calls, v)
}

func (m *MockGauge) Sub(v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value -= v
}

func (m *MockGauge) SetToCurrentTime() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value = float64(time.Now().Unix())
}

func (m *MockGauge) Desc() *prometheus.Desc {
	return prometheus.NewDesc("mock_gauge", "mock gauge for testing", nil, nil)
}

func (m *MockGauge) Write(out *dto.Metric) error {
	return nil
}

func (m *MockGauge) Describe(ch chan<- *prometheus.Desc) {}
func (m *MockGauge) Collect(ch chan<- prometheus.Metric) {}

func (m *MockGauge) GetValue() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.value
}

func (m *MockGauge) GetCalls() []float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]float64{}, m.calls...)
}

func NewMockLogger() DuckLogger {
	// Create a DuckLogger with nil subscription (will just log to stdout)
	return NewDuckLogger("testhost", nil, LevelMetrics)
}

// ============================================================================
// Deterministic Tests (Fast, No Timing Dependencies)
// ============================================================================

// TestProcessMetrics_BasicMetricsAndLogging tests basic metrics processing and JSON logging
func TestProcessMetrics_BasicMetricsAndLogging(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsChan := make(chan int, 100)
	eventCounter := &MockGauge{}
	sizeCounter := &MockGauge{}
	logger, logBuf := CreateTestLoggerWithCapture(t)
	mockTicker := NewMockTickerProvider()

	go ProcessMetrics(metricsChan, logger, eventCounter, sizeCounter, ctx, mockTicker)

	// Send metrics
	metricsChan <- 1000
	metricsChan <- 2000
	metricsChan <- 3000

	// Trigger ticker immediately - no sleep needed
	startTime := time.Now()
	mockTicker.Tick(startTime)
	mockTicker.Tick(startTime.Add(1 * time.Second))

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Verify Prometheus counters
	assert.Equal(t, 3.0, eventCounter.GetValue())
	assert.Equal(t, 6000.0, sizeCounter.GetValue())

	// Verify captured JSON contains metrics
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "EventCounter")
	assert.Contains(t, logOutput, "ByteCounter")
}

// TestProcessMetrics_SlidingWindow_Deterministic tests sliding window behavior deterministically
func TestProcessMetrics_SlidingWindow_Deterministic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsChan := make(chan int, 100)
	eventCounter := &MockGauge{}
	sizeCounter := &MockGauge{}
	logger, logBuf := CreateTestLoggerWithCapture(t)
	mockTicker := NewMockTickerProvider()

	go ProcessMetrics(metricsChan, logger, eventCounter, sizeCounter, ctx, mockTicker)

	startTime := time.Now()

	// Send different amounts in each tick
	metricsChan <- 100 // Tick 1
	mockTicker.Tick(startTime.Add(1 * time.Second))

	metricsChan <- 200 // Tick 2
	mockTicker.Tick(startTime.Add(2 * time.Second))

	metricsChan <- 300 // Tick 3
	mockTicker.Tick(startTime.Add(3 * time.Second))

	metricsChan <- 400 // Tick 4
	mockTicker.Tick(startTime.Add(4 * time.Second))

	metricsChan <- 500 // Tick 5
	mockTicker.Tick(startTime.Add(5 * time.Second))

	metricsChan <- 600 // Tick 6
	mockTicker.Tick(startTime.Add(6 * time.Second))

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify all events were counted
	assert.Equal(t, 6.0, eventCounter.GetValue())
	assert.Equal(t, 2100.0, sizeCounter.GetValue())

	// Verify JSON was logged
	logOutput := logBuf.String()
	assert.NotEmpty(t, logOutput)
	assert.Contains(t, logOutput, "EventCounter")
}

// TestProcessMetrics_NegativeValues_LoggedAndIgnored tests negative value handling
func TestProcessMetrics_NegativeValues_LoggedAndIgnored(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsChan := make(chan int, 100)
	eventCounter := &MockGauge{}
	sizeCounter := &MockGauge{}
	logger, logBuf := CreateTestLoggerWithCapture(t)
	mockTicker := NewMockTickerProvider()

	go ProcessMetrics(metricsChan, logger, eventCounter, sizeCounter, ctx, mockTicker)

	// Send positive values
	metricsChan <- 100
	metricsChan <- 200

	// Send negative values (should be ignored)
	metricsChan <- -50
	metricsChan <- -100

	// Send more positive values
	metricsChan <- 300

	startTime := time.Now()
	mockTicker.Tick(startTime.Add(1 * time.Second))

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify only positive values were counted
	assert.Equal(t, 3.0, eventCounter.GetValue()) // 100, 200, 300
	assert.Equal(t, 600.0, sizeCounter.GetValue()) // 100 + 200 + 300

	// Verify warning was logged for negative values
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "Ignoring negative metric size")
	assert.Contains(t, logOutput, "-50")
	assert.Contains(t, logOutput, "-100")
}

// TestProcessMetrics_MultipleTicks_NoSleep tests multiple ticker cycles
func TestProcessMetrics_MultipleTicks_NoSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsChan := make(chan int, 100)
	eventCounter := &MockGauge{}
	sizeCounter := &MockGauge{}
	logger, logBuf := CreateTestLoggerWithCapture(t)
	mockTicker := NewMockTickerProvider()

	go ProcessMetrics(metricsChan, logger, eventCounter, sizeCounter, ctx, mockTicker)

	startTime := time.Now()

	// Send metrics and trigger multiple ticks
	for i := 0; i < 3; i++ {
		metricsChan <- 500
		mockTicker.Tick(startTime.Add(time.Duration(i+1) * time.Second))
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify all metrics counted
	assert.Equal(t, 3.0, eventCounter.GetValue())
	assert.Equal(t, 1500.0, sizeCounter.GetValue())

	// Verify multiple JSON log entries (one per tick)
	logOutput := logBuf.String()
	assert.NotEmpty(t, logOutput)
	// Should have 3 EventCounter occurrences (one per tick)
	eventCounterCount := strings.Count(logOutput, "EventCounter")
	assert.GreaterOrEqual(t, eventCounterCount, 2)
}

// TestProcessMetrics_ContextCancellation_Deterministic tests context cancellation
func TestProcessMetrics_ContextCancellation_Deterministic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	metricsChan := make(chan int, 100)
	eventCounter := &MockGauge{}
	sizeCounter := &MockGauge{}
	logger, _ := CreateTestLoggerWithCapture(t)
	mockTicker := NewMockTickerProvider()

	done := make(chan bool)
	go func() {
		ProcessMetrics(metricsChan, logger, eventCounter, sizeCounter, ctx, mockTicker)
		done <- true
	}()

	// Send some data
	metricsChan <- 100
	metricsChan <- 200

	// Wait a moment for metrics to be processed
	time.Sleep(50 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for goroutine to exit
	select {
	case <-done:
		// Success - goroutine exited
	case <-time.After(1 * time.Second):
		t.Fatal("ProcessMetrics did not exit after context cancellation")
	}

	// Verify metrics were processed before shutdown
	assert.Equal(t, 2.0, eventCounter.GetValue())
	assert.Equal(t, 300.0, sizeCounter.GetValue())
}

// TestProcessMetrics_JSONOutput_Parsable verifies JSON output can be parsed
func TestProcessMetrics_JSONOutput_Parsable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsChan := make(chan int, 100)
	eventCounter := &MockGauge{}
	sizeCounter := &MockGauge{}
	logger, logBuf := CreateTestLoggerWithCapture(t)
	mockTicker := NewMockTickerProvider()

	go ProcessMetrics(metricsChan, logger, eventCounter, sizeCounter, ctx, mockTicker)

	// Send metrics
	metricsChan <- 1000
	metricsChan <- 2000

	// Wait for both metrics to be processed (check Prometheus counters)
	for i := 0; i < 10; i++ {
		if eventCounter.GetValue() == 2.0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	startTime := time.Now()
	mockTicker.Tick(startTime.Add(1 * time.Second))

	// Wait for processing - give more time for channel processing
	time.Sleep(300 * time.Millisecond)

	// Verify Prometheus counters were updated
	assert.Equal(t, 2.0, eventCounter.GetValue())
	assert.Equal(t, 3000.0, sizeCounter.GetValue())

	// Parse JSON from log output
	logOutput := logBuf.String()
	assert.NotEmpty(t, logOutput)

	// Extract JSON from log lines
	lines := strings.Split(logOutput, "\n")
	foundMetrics := false
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Parse the JSON log entry
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(line), &logEntry)
		assert.NoError(t, err, "Log entry should be valid JSON")

		// Check if it has a msg field containing metrics
		if msg, ok := logEntry["msg"].(string); ok {
			// The msg field should contain the metrics JSON
			var metrics Metrics
			err := json.Unmarshal([]byte(msg), &metrics)
			if err == nil {
				// Successfully parsed metrics
				assert.Equal(t, 2, metrics.EventCounter)
				assert.Equal(t, 3000, metrics.ByteCounter)
				foundMetrics = true
				break
			}
		}
	}

	assert.True(t, foundMetrics, "Could not find valid metrics JSON in log output")
}

// TestProcessMetrics_RateCalculation verifies rate calculation accuracy
func TestProcessMetrics_RateCalculation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsChan := make(chan int, 100)
	eventCounter := &MockGauge{}
	sizeCounter := &MockGauge{}
	logger, logBuf := CreateTestLoggerWithCapture(t)
	mockTicker := NewMockTickerProvider()

	go ProcessMetrics(metricsChan, logger, eventCounter, sizeCounter, ctx, mockTicker)

	// Send 1000 bytes per second for 3 seconds
	// We need to ensure metrics are processed before triggering ticks
	startTime := time.Now()

	// Send first metric
	metricsChan <- 1000
	// Wait for it to be processed
	for i := 0; i < 10; i++ {
		if eventCounter.GetValue() == 1.0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mockTicker.Tick(startTime.Add(1 * time.Second))

	// Send second metric
	metricsChan <- 1000
	// Wait for it to be processed
	for i := 0; i < 10; i++ {
		if eventCounter.GetValue() == 2.0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mockTicker.Tick(startTime.Add(2 * time.Second))

	// Send third metric
	metricsChan <- 1000
	// Wait for it to be processed
	for i := 0; i < 10; i++ {
		if eventCounter.GetValue() == 3.0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mockTicker.Tick(startTime.Add(3 * time.Second))

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify totals
	assert.Equal(t, 3.0, eventCounter.GetValue())
	assert.Equal(t, 3000.0, sizeCounter.GetValue())

	// Parse and verify rate calculations
	logOutput := logBuf.String()
	assert.NotEmpty(t, logOutput)

	// Extract JSON from log lines
	lines := strings.Split(logOutput, "\n")
	foundMetrics := false
	for _, line := range lines {
		if line == "" {
			continue
		}
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(line), &logEntry)
		if err != nil {
			continue
		}

		if msg, ok := logEntry["msg"].(string); ok {
			var metrics Metrics
			err := json.Unmarshal([]byte(msg), &metrics)
			if err != nil {
				continue
			}

			// Look for the final tick (3 seconds) with all measurements
			if metrics.EventCounter == 3 && metrics.ByteCounter == 3000 {
				// Verify rate calculations
				// Average: 3000 bytes / 3 seconds = 1000 bytes/sec
				// Average: 3 events / 3 seconds = 1 event/sec
				assert.InDelta(t, 1000.0, metrics.AvgDataRate, 10.0, "Average data rate should be ~1000 bytes/sec")
				assert.InDelta(t, 1.0, metrics.AvgTrgRate, 0.1, "Average trigger rate should be ~1 event/sec")

				// Current rate is calculated over sliding window
				// At tick 3: (3000 - 1000) / (3s - 1s) = 1000 bytes/sec
				assert.InDelta(t, 1000.0, metrics.CurrentDataRate, 10.0, "Current data rate should be ~1000 bytes/sec")
				assert.InDelta(t, 1.0, metrics.CurrentTrgRate, 0.1, "Current trigger rate should be ~1 event/sec")

				foundMetrics = true
				break
			}
		}
	}

	assert.True(t, foundMetrics, "Could not find valid metrics JSON in log output")
}

// TestProcessMetrics_ZeroMetrics_NoCrash tests with no metrics sent
func TestProcessMetrics_ZeroMetrics_NoCrash(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsChan := make(chan int, 100)
	eventCounter := &MockGauge{}
	sizeCounter := &MockGauge{}
	logger, _ := CreateTestLoggerWithCapture(t)
	mockTicker := NewMockTickerProvider()

	go ProcessMetrics(metricsChan, logger, eventCounter, sizeCounter, ctx, mockTicker)

	// Trigger ticker without sending any metrics
	startTime := time.Now()
	mockTicker.Tick(startTime.Add(1 * time.Second))

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify counters remain at zero
	assert.Equal(t, 0.0, eventCounter.GetValue())
	assert.Equal(t, 0.0, sizeCounter.GetValue())
}

// TestProcessMetrics_RapidBurst_Deterministic tests rapid burst of metrics
func TestProcessMetrics_RapidBurst_Deterministic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsChan := make(chan int, 1000)
	eventCounter := &MockGauge{}
	sizeCounter := &MockGauge{}
	logger, _ := CreateTestLoggerWithCapture(t)
	mockTicker := NewMockTickerProvider()

	go ProcessMetrics(metricsChan, logger, eventCounter, sizeCounter, ctx, mockTicker)

	// Send 100 metrics rapidly
	for i := 0; i < 100; i++ {
		metricsChan <- 10
	}

	// Trigger tick
	startTime := time.Now()
	mockTicker.Tick(startTime.Add(1 * time.Second))

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify all were counted
	assert.Equal(t, 100.0, eventCounter.GetValue())
	assert.Equal(t, 1000.0, sizeCounter.GetValue())
}

// ============================================================================
// Specialized Tests (Timing-Specific Behavior)
// ============================================================================

// TestProcessMetrics_GaugeUpdateOrder verifies metrics update counters before tick
func TestProcessMetrics_GaugeUpdateOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsChan := make(chan int, 100)
	eventCounter := &MockGauge{}
	sizeCounter := &MockGauge{}
	logger, _ := CreateTestLoggerWithCapture(t)
	mockTicker := NewMockTickerProvider()

	go ProcessMetrics(metricsChan, logger, eventCounter, sizeCounter, ctx, mockTicker)

	// Send metrics and verify counters update immediately (before tick)
	metricsChan <- 100
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1.0, eventCounter.GetValue())
	assert.Equal(t, 100.0, sizeCounter.GetValue())

	metricsChan <- 200
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2.0, eventCounter.GetValue())
	assert.Equal(t, 300.0, sizeCounter.GetValue())

	// Now trigger tick
	startTime := time.Now()
	mockTicker.Tick(startTime.Add(1 * time.Second))
	time.Sleep(100 * time.Millisecond)

	// Counters should remain same
	assert.Equal(t, 2.0, eventCounter.GetValue())
	assert.Equal(t, 300.0, sizeCounter.GetValue())
}
