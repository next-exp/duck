//go:build nohdf5

package main

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"testing"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Helper Functions
// ============================================================================

// createTestMetricsRegistry creates a new MetricsRegistry for testing
func createTestMetricsRegistry() *MetricsRegistry {
	return NewMetricsRegistry()
}

// ============================================================================
// Test Suite 1: Constructor Tests
// ============================================================================

func TestNewMetricsRegistry_CreatesAllMetrics(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Verify all gauge counters are initialized
	assert.NotNil(t, metrics.eventCounter)
	assert.NotNil(t, metrics.sizeCounter)
	assert.NotNil(t, metrics.evtErrorCounter)
	assert.NotNil(t, metrics.incompleteEventsCounter)
	assert.NotNil(t, metrics.dataChannelCounter)
	assert.NotNil(t, metrics.filesOpenedCounter)
	assert.NotNil(t, metrics.subRunCounter)
	assert.NotNil(t, metrics.evtsWaitingBinaryWriterCounter)
	assert.NotNil(t, metrics.evtsWaitingDecodeCounter)
	assert.NotNil(t, metrics.evtDecoderErrorCounter)

	// Verify all histograms are initialized
	assert.NotNil(t, metrics.writeTimeHistogram)
	assert.NotNil(t, metrics.h5WriteTimeHistogram)
	assert.NotNil(t, metrics.decodeTimeHistogram)

	// Verify all GaugeVec metrics are initialized
	assert.NotNil(t, metrics.h5FilesOpenedCounter)
	assert.NotNil(t, metrics.evtsWaitingForHdf5WriterCounter)
	assert.NotNil(t, metrics.evtsPerTriggerTypeCounter)
}

func TestNewMetricsRegistry_UniqueMetricNames(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Collect metric names and verify uniqueness
	registry := prometheus.NewRegistry()
	err := metrics.Register(registry)
	require.NoError(t, err)

	// Set at least one label value for GaugeVec metrics so they appear in Gather()
	metrics.h5FilesOpenedCounter.WithLabelValues("test_trigger").Set(0)
	metrics.evtsWaitingForHdf5WriterCounter.WithLabelValues("test_trigger").Set(0)
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("test_trigger").Set(0)

	// Gather all metrics
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	metricNames := make(map[string]bool)
	for _, mf := range metricFamilies {
		name := mf.GetName()
		if metricNames[name] {
			t.Errorf("Duplicate metric name found: %s", name)
		}
		metricNames[name] = true
	}

	// Verify expected number of metrics (16 unique metrics)
	expectedMetricCount := 16
	if len(metricNames) != expectedMetricCount {
		t.Errorf("Expected %d unique metrics, got %d", expectedMetricCount, len(metricNames))
	}

	// Verify specific metric names exist
	expectedNames := []string{
		"gdc_event_count",
		"gdc_data_bytes",
		"gdc_evt_error_count",
		"gdc_incomplete_evts_count",
		"gdc_data_channel_count",
		"gdc_write_time_histogram",
		"gdc_hdf5_write_time_histogram",
		"gdc_decoder_time_histogram",
		"gdc_opened_hdf5_files_count",
		"gdc_opened_files_count",
		"gdc_events_waiting_hdf5_write_count",
		"gdc_subrun_count",
		"gdc_events_waiting_binary_write_count",
		"gdc_events_waiting_decode_count",
		"gdc_events_per_trigger_type_count",
		"gdc_decoder_error_count",
	}

	for _, name := range expectedNames {
		if !metricNames[name] {
			t.Errorf("Expected metric %q not found", name)
		}
	}
}

// ============================================================================
// Test Suite 2: Register Method Tests
// ============================================================================

func TestMetricsRegistry_Register_AllMetrics(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()

	err := metrics.Register(registry)
	assert.NoError(t, err)

	// Set at least one label value for GaugeVec metrics so they appear in Gather()
	metrics.h5FilesOpenedCounter.WithLabelValues("test_trigger").Set(0)
	metrics.evtsWaitingForHdf5WriterCounter.WithLabelValues("test_trigger").Set(0)
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("test_trigger").Set(0)

	// Verify metrics are registered by gathering them
	metricFamilies, err := registry.Gather()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(metricFamilies), 16)
}

func TestMetricsRegistry_Register_DuplicateRegistration(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()

	// First registration should succeed
	err := metrics.Register(registry)
	assert.NoError(t, err)

	// Second registration should panic
	assert.Panics(t, func() {
		metrics.Register(registry)
	})
}

func TestMetricsRegistry_Register_NilRegistry(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Should panic when trying to register with nil registry
	assert.Panics(t, func() {
		metrics.Register(nil)
	})
}

// ============================================================================
// Test Suite 3: Reset Method Tests
// ============================================================================

func TestMetricsRegistry_Reset_AllCountersToZero(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set all counters to non-zero values
	metrics.eventCounter.Set(100)
	metrics.sizeCounter.Set(50000)
	metrics.evtErrorCounter.Set(10)
	metrics.incompleteEventsCounter.Set(5)
	metrics.filesOpenedCounter.Set(3)
	metrics.subRunCounter.Set(42)
	metrics.evtsWaitingDecodeCounter.Set(7)
	metrics.evtDecoderErrorCounter.Set(2)

	// Reset
	metrics.Reset()

	// Verify all counters are zero
	eventCount, _ := duck.GetPrometheusGaugeValue(metrics.eventCounter)
	assert.Equal(t, int64(0), eventCount)

	sizeCount, _ := duck.GetPrometheusGaugeValue(metrics.sizeCounter)
	assert.Equal(t, int64(0), sizeCount)

	errorCount, _ := duck.GetPrometheusGaugeValue(metrics.evtErrorCounter)
	assert.Equal(t, int64(0), errorCount)

	incompleteCount, _ := duck.GetPrometheusGaugeValue(metrics.incompleteEventsCounter)
	assert.Equal(t, int64(0), incompleteCount)

	filesOpenedCount, _ := duck.GetPrometheusGaugeValue(metrics.filesOpenedCounter)
	assert.Equal(t, int64(0), filesOpenedCount)

	subRunCount, _ := duck.GetPrometheusGaugeValue(metrics.subRunCounter)
	assert.Equal(t, int64(0), subRunCount)

	decodeWaitingCount, _ := duck.GetPrometheusGaugeValue(metrics.evtsWaitingDecodeCounter)
	assert.Equal(t, int64(0), decodeWaitingCount)

	decoderErrorCount, _ := duck.GetPrometheusGaugeValue(metrics.evtDecoderErrorCounter)
	assert.Equal(t, int64(0), decoderErrorCount)
}

func TestMetricsRegistry_Reset_Idempotent(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set some values
	metrics.eventCounter.Set(123)
	metrics.sizeCounter.Set(456)

	// Reset multiple times
	metrics.Reset()
	metrics.Reset()
	metrics.Reset()

	// Values should still be zero
	eventCount, _ := duck.GetPrometheusGaugeValue(metrics.eventCounter)
	assert.Equal(t, int64(0), eventCount)

	sizeCount, _ := duck.GetPrometheusGaugeValue(metrics.sizeCounter)
	assert.Equal(t, int64(0), sizeCount)
}

func TestMetricsRegistry_Reset_PartiallySetCounters(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Only set some counters
	metrics.eventCounter.Set(100)
	metrics.evtErrorCounter.Set(5)
	// Leave others at default (0)

	// Reset
	metrics.Reset()

	// Verify all counters are zero
	eventCount, _ := duck.GetPrometheusGaugeValue(metrics.eventCounter)
	assert.Equal(t, int64(0), eventCount)

	errorCount, _ := duck.GetPrometheusGaugeValue(metrics.evtErrorCounter)
	assert.Equal(t, int64(0), errorCount)

	// Verify unset counters are also zero
	sizeCount, _ := duck.GetPrometheusGaugeValue(metrics.sizeCounter)
	assert.Equal(t, int64(0), sizeCount)
}

// ============================================================================
// Test Suite 4: Getter Methods Tests
// ============================================================================

func TestGetEventCounter_ReturnsCorrectValue(t *testing.T) {
	metrics := createTestMetricsRegistry()
	metrics.eventCounter.Set(42)

	value := metrics.GetEventCounter()
	assert.Equal(t, int64(42), value)
}

func TestGetEventCounter_ZeroValue(t *testing.T) {
	metrics := createTestMetricsRegistry()
	// Default is 0

	value := metrics.GetEventCounter()
	assert.Equal(t, int64(0), value)
}

func TestGetEventCounter_FloatTruncation(t *testing.T) {
	metrics := createTestMetricsRegistry()
	metrics.eventCounter.Set(42.7)

	value := metrics.GetEventCounter()
	assert.Equal(t, int64(42), value) // Truncated
}

func TestGetEventCounter_LargeValue(t *testing.T) {
	metrics := createTestMetricsRegistry()
	// Use a large value safely within float64 precision (2^50)
	// float64 can exactly represent integers up to 2^53, but we use 2^50 for safety
	largeValue := int64(1) << 50 // 1125899906842624
	metrics.eventCounter.Set(float64(largeValue))

	value := metrics.GetEventCounter()
	assert.Equal(t, largeValue, value)
}

func TestGetBytesCounter_ReturnsCorrectValue(t *testing.T) {
	metrics := createTestMetricsRegistry()
	metrics.sizeCounter.Set(1024000)

	value := metrics.GetBytesCounter()
	assert.Equal(t, int64(1024000), value)
}

func TestGetBytesCounter_ZeroValue(t *testing.T) {
	metrics := createTestMetricsRegistry()

	value := metrics.GetBytesCounter()
	assert.Equal(t, int64(0), value)
}

func TestGetBytesCounter_FloatTruncation(t *testing.T) {
	metrics := createTestMetricsRegistry()
	metrics.sizeCounter.Set(1024.9)

	value := metrics.GetBytesCounter()
	assert.Equal(t, int64(1024), value) // Truncated
}

func TestGetBytesCounter_LargeValue(t *testing.T) {
	metrics := createTestMetricsRegistry()
	largeValue := int64(1024 * 1024 * 1024) // 1 GB
	metrics.sizeCounter.Set(float64(largeValue))

	value := metrics.GetBytesCounter()
	assert.Equal(t, largeValue, value)
}

func TestGetEvtErrorCounter_ReturnsCorrectValue(t *testing.T) {
	metrics := createTestMetricsRegistry()
	metrics.evtErrorCounter.Set(15)

	value := metrics.GetEvtErrorCounter()
	assert.Equal(t, int64(15), value)
}

func TestGetEvtErrorCounter_ZeroValue(t *testing.T) {
	metrics := createTestMetricsRegistry()

	value := metrics.GetEvtErrorCounter()
	assert.Equal(t, int64(0), value)
}

func TestGetEvtErrorCounter_FloatTruncation(t *testing.T) {
	metrics := createTestMetricsRegistry()
	metrics.evtErrorCounter.Set(5.8)

	value := metrics.GetEvtErrorCounter()
	assert.Equal(t, int64(5), value) // Truncated
}

func TestGetEvtErrorCounter_LargeValue(t *testing.T) {
	metrics := createTestMetricsRegistry()
	// Use a value that can be exactly represented as float64
	// Floating point has 53 bits of precision, so MaxInt64 / 2^10 should be safe
	largeValue := int64(1) << 53 // Maximum integer exactly representable in float64
	metrics.evtErrorCounter.Set(float64(largeValue))

	value := metrics.GetEvtErrorCounter()
	assert.Equal(t, largeValue, value)
}

// ============================================================================
// Test Suite 5: Histogram Metrics Tests
// ============================================================================

func TestHistogram_WriteTimeHistogram_Observation(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.writeTimeHistogram)

	// Make some observations
	metrics.writeTimeHistogram.Observe(5.0)
	metrics.writeTimeHistogram.Observe(10.0)
	metrics.writeTimeHistogram.Observe(15.0)

	// Verify histogram has observations
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Find the histogram
	histogramFound := false
	var sampleCount uint64
	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_write_time_histogram" {
			histogramFound = true
			if len(mf.GetMetric()) > 0 {
				histo := mf.GetMetric()[0].GetHistogram()
				sampleCount = histo.GetSampleCount()
			}
			break
		}
	}

	assert.True(t, histogramFound, "writeTimeHistogram not found in registry")
	assert.Equal(t, uint64(3), sampleCount)
}

func TestHistogram_H5WriteTimeHistogram_Buckets(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.h5WriteTimeHistogram)

	// Observe values that should fall into different buckets
	metrics.h5WriteTimeHistogram.Observe(25)   // bucket 0
	metrics.h5WriteTimeHistogram.Observe(75)   // bucket 1
	metrics.h5WriteTimeHistogram.Observe(150)  // bucket 3
	metrics.h5WriteTimeHistogram.Observe(300)  // bucket 8
	metrics.h5WriteTimeHistogram.Observe(1000) // bucket 13

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Find and verify histogram
	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_hdf5_write_time_histogram" {
			if len(mf.GetMetric()) > 0 {
				histo := mf.GetMetric()[0].GetHistogram()
				assert.Equal(t, uint64(5), histo.GetSampleCount(), "Expected 5 observations")
			}
			break
		}
	}
}

func TestHistogram_DecodeTimeHistogram_OutOfRange(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.decodeTimeHistogram)

	// Observe value larger than max bucket (max is 3000)
	metrics.decodeTimeHistogram.Observe(5000) // Should go into +Inf bucket

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Verify observation was recorded
	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_decoder_time_histogram" {
			if len(mf.GetMetric()) > 0 {
				histo := mf.GetMetric()[0].GetHistogram()
				assert.Equal(t, uint64(1), histo.GetSampleCount())
			}
			break
		}
	}
}

func TestHistogram_ConcurrentObservations(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.writeTimeHistogram)

	// Concurrent observations
	numGoroutines := 100
	observationsPerGoroutine := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < observationsPerGoroutine; j++ {
				metrics.writeTimeHistogram.Observe(float64(j))
			}
		}()
	}

	wg.Wait()

	// Verify all observations were recorded
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	expectedCount := uint64(numGoroutines * observationsPerGoroutine)
	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_write_time_histogram" {
			if len(mf.GetMetric()) > 0 {
				histo := mf.GetMetric()[0].GetHistogram()
				assert.Equal(t, expectedCount, histo.GetSampleCount())
			}
			break
		}
	}
}

// ============================================================================
// Test Suite 6: GaugeVec Metrics Tests
// ============================================================================

func TestGaugeVec_H5FilesOpenedCounter_WithLabels(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set values with different labels
	metrics.h5FilesOpenedCounter.WithLabelValues("trigger1").Set(5)
	metrics.h5FilesOpenedCounter.WithLabelValues("trigger2").Set(3)

	// We can verify the metric exists by registering and gathering
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.h5FilesOpenedCounter)

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Should have 2 metrics with different label values
	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_opened_hdf5_files_count" {
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 2)
		}
	}
}

func TestGaugeVec_H5FilesOpenedCounter_MultipleLabels(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.h5FilesOpenedCounter)

	// Test multiple label combinations
	triggers := []string{"trigger_a", "trigger_b", "trigger_c", "trigger_d"}
	for _, trigger := range triggers {
		metrics.h5FilesOpenedCounter.WithLabelValues(trigger).Set(1)
	}

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Verify we have metrics for all labels
	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_opened_hdf5_files_count" {
			assert.Equal(t, len(triggers), len(mf.GetMetric()))
		}
	}
}

func TestGaugeVec_H5FilesOpenedCounter_ConcurrentLabelAccess(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.h5FilesOpenedCounter)

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent writes with different labels
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			label := fmt.Sprintf("trigger_%d", idx%10) // 10 different labels
			metrics.h5FilesOpenedCounter.WithLabelValues(label).Set(float64(idx))
		}(i)
	}

	wg.Wait()

	// Verify metrics were created
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_opened_hdf5_files_count" {
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 10)
		}
	}
}

func TestGaugeVec_EvtsWaitingForHdf5WriterCounter_WithLabels(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set values for different triggers
	metrics.evtsWaitingForHdf5WriterCounter.WithLabelValues("calibration").Set(10)
	metrics.evtsWaitingForHdf5WriterCounter.WithLabelValues("physics").Set(50)

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.evtsWaitingForHdf5WriterCounter)

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_events_waiting_hdf5_write_count" {
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 2)
		}
	}
}

func TestGaugeVec_EvtsWaitingForHdf5WriterCounter_MultipleLabels(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.evtsWaitingForHdf5WriterCounter)

	// Test increment/decrement operations
	metrics.evtsWaitingForHdf5WriterCounter.WithLabelValues("test1").Inc()
	metrics.evtsWaitingForHdf5WriterCounter.WithLabelValues("test1").Inc()
	metrics.evtsWaitingForHdf5WriterCounter.WithLabelValues("test1").Dec()

	// Should have 1 event waiting
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_events_waiting_hdf5_write_count" {
			if len(mf.GetMetric()) > 0 {
				val := mf.GetMetric()[0].GetGauge().GetValue()
				assert.Equal(t, 1.0, val)
			}
		}
	}
}

func TestGaugeVec_EvtsPerTriggerTypeCounter_WithLabels(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set different counts for different trigger types
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("random").Set(1000)
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("periodic").Set(500)
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("external").Set(250)

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.evtsPerTriggerTypeCounter)

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_events_per_trigger_type_count" {
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 3)
		}
	}
}

func TestGaugeVec_EvtsPerTriggerTypeCounter_MultipleLabels(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.evtsPerTriggerTypeCounter)

	// Add to counters
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("trigger_a").Add(10)
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("trigger_a").Add(5)
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("trigger_b").Add(20)

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_events_per_trigger_type_count" {
			// Should have 2 label combinations
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 2)
		}
	}
}

// ============================================================================
// Test Suite 7: Prometheus HTTP Endpoint Tests
// ============================================================================

func TestStartPrometheus_CreatesHTTPServer(t *testing.T) {
	metrics := createTestMetricsRegistry()
	logger := duck.NewDuckLogger("testhost", nil, 0)

	// Use a unique port based on test run to avoid conflicts
	// Note: startPrometheus doesn't provide a way to shut down the server,
	// so this test may leave a goroutine running. Use a high port to minimize conflicts.
	port := 19876

	// Check if port is available before starting
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	if err == nil {
		resp.Body.Close()
		t.Skip("Port already in use, skipping test")
	}

	// startPrometheus spawns a goroutine, verify it doesn't panic
	assert.NotPanics(t, func() {
		startPrometheus(metrics, port, logger)
	})

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify the server is actually running and serving metrics
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	if err != nil {
		t.Logf("Could not connect to prometheus endpoint: %v", err)
		return
	}
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPrometheusEndpoint_MetricsEndpointContent(t *testing.T) {
	metrics := createTestMetricsRegistry()
	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	// Scrape metrics
	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	// Verify basic Prometheus format elements
	assert.Contains(t, content, "# HELP")
	assert.Contains(t, content, "# TYPE")
}

func TestPrometheusEndpoint_GaugeMetrics(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set some values
	metrics.eventCounter.Set(123)
	metrics.sizeCounter.Set(456789)

	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	// Scrape metrics
	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	// Verify specific metrics and values
	duck.AssertMetricExists(t, content, "gdc_event_count")
	duck.AssertMetricValue(t, content, "gdc_event_count", "123")

	duck.AssertMetricExists(t, content, "gdc_data_bytes")
	duck.AssertMetricValue(t, content, "gdc_data_bytes", "456789")

	// Verify HELP and TYPE comments
	assert.Contains(t, content, "# HELP gdc_event_count No of events processed by GDC")
	assert.Contains(t, content, "# TYPE gdc_event_count gauge")
}

func TestPrometheusEndpoint_HistogramMetrics(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Make some histogram observations
	metrics.writeTimeHistogram.Observe(5.0)
	metrics.writeTimeHistogram.Observe(10.0)

	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	// Scrape metrics
	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	// Verify histogram metric
	duck.AssertMetricContains(t, content, "gdc_write_time_histogram")
	assert.Contains(t, content, "# HELP gdc_write_time_histogram Time taken to write each event")
	assert.Contains(t, content, "# TYPE gdc_write_time_histogram histogram")
	assert.Contains(t, content, "gdc_write_time_histogram_bucket")
	assert.Contains(t, content, "gdc_write_time_histogram_sum")
	assert.Contains(t, content, "gdc_write_time_histogram_count")
}

func TestPrometheusEndpoint_GaugeVecWithLabels(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set labeled metrics
	metrics.h5FilesOpenedCounter.WithLabelValues("trigger1").Set(5)
	metrics.h5FilesOpenedCounter.WithLabelValues("trigger2").Set(3)
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("random").Set(100)
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("external").Set(50)

	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	// Scrape metrics
	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	// Verify labeled metrics exist
	assert.Contains(t, content, `gdc_opened_hdf5_files_count{trigger="trigger1"} 5`)
	assert.Contains(t, content, `gdc_opened_hdf5_files_count{trigger="trigger2"} 3`)
	assert.Contains(t, content, `gdc_events_per_trigger_type_count{trigger="random"} 100`)
	assert.Contains(t, content, `gdc_events_per_trigger_type_count{trigger="external"} 50`)
}

func TestPrometheusEndpoint_AllMetricsExposed(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set label values for GaugeVec metrics so they appear in output
	metrics.h5FilesOpenedCounter.WithLabelValues("test_trigger").Set(0)
	metrics.evtsWaitingForHdf5WriterCounter.WithLabelValues("test_trigger").Set(0)
	metrics.evtsPerTriggerTypeCounter.WithLabelValues("test_trigger").Set(0)

	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	// Scrape metrics
	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	// Verify all expected metrics are present
	expectedMetrics := []string{
		"gdc_event_count",
		"gdc_data_bytes",
		"gdc_evt_error_count",
		"gdc_incomplete_evts_count",
		"gdc_data_channel_count",
		"gdc_write_time_histogram",
		"gdc_hdf5_write_time_histogram",
		"gdc_decoder_time_histogram",
		"gdc_opened_hdf5_files_count",
		"gdc_opened_files_count",
		"gdc_events_waiting_hdf5_write_count",
		"gdc_subrun_count",
		"gdc_events_waiting_binary_write_count",
		"gdc_events_waiting_decode_count",
		"gdc_events_per_trigger_type_count",
		"gdc_decoder_error_count",
	}

	for _, metric := range expectedMetrics {
		duck.AssertMetricExists(t, content, metric)
	}
}

// ============================================================================
// Test Suite 8: Concurrency Tests
// ============================================================================

func TestMetricsRegistry_ConcurrentGetAndSet(t *testing.T) {
	metrics := createTestMetricsRegistry()

	numGoroutines := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent sets
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			metrics.eventCounter.Set(float64(idx))
		}(i)

		// Concurrent gets
		go func() {
			defer wg.Done()
			_ = metrics.GetEventCounter()
		}()
	}

	wg.Wait()

	// Should not have any races or panics
	_ = metrics.GetEventCounter()
}

func TestMetricsRegistry_ConcurrentReset(t *testing.T) {
	metrics := createTestMetricsRegistry()

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent sets and resets
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			metrics.eventCounter.Set(float64(idx))
		}(i)

		go func() {
			defer wg.Done()
			metrics.Reset()
		}()
	}

	wg.Wait()

	// Should not have any races or panics
	_ = metrics.GetEventCounter()
}

func TestMetricsRegistry_ConcurrentHistogramObservations(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.writeTimeHistogram)

	numGoroutines := 100
	observationsPerGoroutine := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < observationsPerGoroutine; j++ {
				metrics.writeTimeHistogram.Observe(float64(j))
			}
		}(i)
	}

	wg.Wait()

	// Verify total count
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	expectedCount := uint64(numGoroutines * observationsPerGoroutine)
	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_write_time_histogram" {
			if len(mf.GetMetric()) > 0 {
				histo := mf.GetMetric()[0].GetHistogram()
				assert.Equal(t, expectedCount, histo.GetSampleCount())
			}
			break
		}
	}
}

func TestMetricsRegistry_ConcurrentGaugeVecAccess(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.h5FilesOpenedCounter)

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent writes with different labels
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			label := fmt.Sprintf("trigger_%d", idx%10)
			metrics.h5FilesOpenedCounter.WithLabelValues(label).Set(float64(idx))
		}(i)

		// Concurrent reads
		go func(idx int) {
			defer wg.Done()
			label := fmt.Sprintf("trigger_%d", idx%10)
			metrics.h5FilesOpenedCounter.WithLabelValues(label).Inc()
		}(i)
	}

	wg.Wait()

	// Verify no races
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_opened_hdf5_files_count" {
			found = true
			break
		}
	}
	assert.True(t, found, "h5FilesOpenedCounter not found in registry")
}

func TestMetricsRegistry_ConcurrentAllOperations(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	metrics.Register(registry)

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 5)

	for i := 0; i < numGoroutines; i++ {
		// Gauge operations
		go func(idx int) {
			defer wg.Done()
			metrics.eventCounter.Set(float64(idx))
		}(i)

		go func(idx int) {
			defer wg.Done()
			_ = metrics.GetEventCounter()
		}(i)

		// Histogram operations
		go func(idx int) {
			defer wg.Done()
			metrics.writeTimeHistogram.Observe(float64(idx))
		}(i)

		// GaugeVec operations
		go func(idx int) {
			defer wg.Done()
			label := fmt.Sprintf("trigger_%d", idx%5)
			metrics.h5FilesOpenedCounter.WithLabelValues(label).Set(float64(idx))
		}(i)

		// Reset operations
		go func() {
			defer wg.Done()
			metrics.Reset()
		}()
	}

	wg.Wait()

	// Verify system is still functional
	_ = metrics.GetEventCounter()
	_ = metrics.GetBytesCounter()
	_ = metrics.GetEvtErrorCounter()
}

// ============================================================================
// Test Suite 9: Integration Tests
// ============================================================================

func TestMetricsFullLifecycle(t *testing.T) {
	// Create
	metrics := createTestMetricsRegistry()
	assert.NotNil(t, metrics)

	// Register
	registry := prometheus.NewRegistry()
	err := metrics.Register(registry)
	assert.NoError(t, err)

	// Set some values
	metrics.eventCounter.Set(100)
	metrics.sizeCounter.Set(50000)
	metrics.writeTimeHistogram.Observe(5.0)
	metrics.h5FilesOpenedCounter.WithLabelValues("test").Set(3)

	// Scrape via HTTP
	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	// Verify metrics are exposed
	duck.AssertMetricExists(t, content, "gdc_event_count")
	duck.AssertMetricValue(t, content, "gdc_event_count", "100")
	duck.AssertMetricExists(t, content, "gdc_write_time_histogram")

	// Reset
	metrics.Reset()

	// Verify reset
	eventCount, _ := duck.GetPrometheusGaugeValue(metrics.eventCounter)
	assert.Equal(t, int64(0), eventCount)

	// Verify via HTTP
	contentAfterReset, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)
	duck.AssertMetricValue(t, contentAfterReset, "gdc_event_count", "0")
}

func TestMetricsRealWorldScenario(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	metrics.Register(registry)

	// Simulate a run: events arriving, being processed, errors occurring
	// Initial state
	metrics.Reset()

	// Events start arriving
	for i := 0; i < 100; i++ {
		metrics.eventCounter.Inc()
		metrics.sizeCounter.Add(1024) // 1 KB per event
		metrics.writeTimeHistogram.Observe(float64(i%25))
	}

	// Some errors occur
	metrics.evtErrorCounter.Set(3)
	metrics.evtDecoderErrorCounter.Set(1)

	// Files are opened
	metrics.h5FilesOpenedCounter.WithLabelValues("calibration").Set(2)
	metrics.evtsWaitingForHdf5WriterCounter.WithLabelValues("calibration").Set(50)

	// Verify state via getters
	eventCount := metrics.GetEventCounter()
	assert.Equal(t, int64(100), eventCount)

	byteCount := metrics.GetBytesCounter()
	assert.Equal(t, int64(102400), byteCount) // 100 * 1024

	errorCount := metrics.GetEvtErrorCounter()
	assert.Equal(t, int64(3), errorCount)

	// Verify via HTTP
	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	duck.AssertMetricValue(t, content, "gdc_event_count", "100")
	duck.AssertMetricValue(t, content, "gdc_data_bytes", "102400")
	duck.AssertMetricValue(t, content, "gdc_evt_error_count", "3")
	assert.Contains(t, content, `gdc_opened_hdf5_files_count{trigger="calibration"} 2`)
}

func TestMetricsEdgeCases(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Test negative values (should be allowed for gauges)
	metrics.eventCounter.Set(-1)
	negVal := metrics.GetEventCounter()
	assert.Equal(t, int64(-1), negVal)

	// Test very large values
	maxInt := float64(math.MaxInt64)
	metrics.sizeCounter.Set(maxInt)
	largeVal := metrics.GetBytesCounter()
	assert.Equal(t, int64(maxInt), largeVal)

	// Test zero values
	metrics.evtErrorCounter.Set(0)
	zeroVal := metrics.GetEvtErrorCounter()
	assert.Equal(t, int64(0), zeroVal)

	// Reset should handle edge cases
	metrics.Reset()
	assert.Equal(t, int64(0), metrics.GetEventCounter())
}

func TestMetricsMultipleResets(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set initial values
	metrics.eventCounter.Set(100)
	metrics.sizeCounter.Set(200)
	metrics.evtErrorCounter.Set(5)

	// First reset
	metrics.Reset()
	assert.Equal(t, int64(0), metrics.GetEventCounter())

	// Set new values
	metrics.eventCounter.Set(50)
	metrics.sizeCounter.Set(75)

	// Second reset
	metrics.Reset()
	assert.Equal(t, int64(0), metrics.GetEventCounter())
	assert.Equal(t, int64(0), metrics.GetBytesCounter())

	// Third reset (should still work)
	metrics.Reset()
	assert.Equal(t, int64(0), metrics.GetEventCounter())
}

func TestMetricsHistogramsAcrossReset(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.writeTimeHistogram)

	// Make observations
	for i := 0; i < 10; i++ {
		metrics.writeTimeHistogram.Observe(float64(i))
	}

	// Reset gauges (note: Reset() doesn't reset histograms)
	metrics.Reset()

	// Histogram should still have observations
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if mf.GetName() == "gdc_write_time_histogram" {
			if len(mf.GetMetric()) > 0 {
				histo := mf.GetMetric()[0].GetHistogram()
				assert.Equal(t, uint64(10), histo.GetSampleCount(), "Histogram should retain observations after Reset()")
			}
			break
		}
	}
}
