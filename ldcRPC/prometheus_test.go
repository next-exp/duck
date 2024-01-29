package main

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

func TestGetEventCounter_ReturnsCorrectValue(t *testing.T) {
	metrics := NewMetricsRegistry()
	metrics.eventCounter.Set(42)

	value := metrics.GetEventCounter()
	assert.Equal(t, int64(42), value)
}

func TestGetEventCounter_ZeroValue(t *testing.T) {
	metrics := NewMetricsRegistry()
	// Default is 0

	value := metrics.GetEventCounter()
	assert.Equal(t, int64(0), value)
}

func TestGetEventCounter_FloatTruncation(t *testing.T) {
	metrics := NewMetricsRegistry()
	metrics.eventCounter.Set(42.7)

	value := metrics.GetEventCounter()
	assert.Equal(t, int64(42), value) // Truncated
}

func TestGetEventCounter_LargeValue(t *testing.T) {
	metrics := NewMetricsRegistry()
	largeValue := int64(1) << 50 // 1125899906842624
	metrics.eventCounter.Set(float64(largeValue))

	value := metrics.GetEventCounter()
	assert.Equal(t, largeValue, value)
}

func TestGetBytesCounter_ReturnsCorrectValue(t *testing.T) {
	metrics := NewMetricsRegistry()
	metrics.sizeCounter.Set(1024000)

	value := metrics.GetBytesCounter()
	assert.Equal(t, int64(1024000), value)
}

func TestGetBytesCounter_ZeroValue(t *testing.T) {
	metrics := NewMetricsRegistry()

	value := metrics.GetBytesCounter()
	assert.Equal(t, int64(0), value)
}

func TestGetBytesCounter_FloatTruncation(t *testing.T) {
	metrics := NewMetricsRegistry()
	metrics.sizeCounter.Set(1024.9)

	value := metrics.GetBytesCounter()
	assert.Equal(t, int64(1024), value) // Truncated
}

func TestGetBytesCounter_LargeValue(t *testing.T) {
	metrics := NewMetricsRegistry()
	largeValue := int64(1024 * 1024 * 1024) // 1 GB
	metrics.sizeCounter.Set(float64(largeValue))

	value := metrics.GetBytesCounter()
	assert.Equal(t, largeValue, value)
}

func TestGetPacketErrorCounter_ReturnsCorrectValue(t *testing.T) {
	metrics := NewMetricsRegistry()
	metrics.packetErrorCounter.Set(5)

	value := metrics.GetPacketErrorCounter()
	assert.Equal(t, int64(5), value)
}

func TestGetPacketErrorCounter_ZeroValue(t *testing.T) {
	metrics := NewMetricsRegistry()

	value := metrics.GetPacketErrorCounter()
	assert.Equal(t, int64(0), value)
}

func TestGetPacketErrorCounter_FloatTruncation(t *testing.T) {
	metrics := NewMetricsRegistry()
	metrics.packetErrorCounter.Set(5.8)

	value := metrics.GetPacketErrorCounter()
	assert.Equal(t, int64(5), value) // Truncated
}

func TestGetPacketErrorCounter_LargeValue(t *testing.T) {
	metrics := NewMetricsRegistry()
	largeValue := int64(1) << 53
	metrics.packetErrorCounter.Set(float64(largeValue))

	value := metrics.GetPacketErrorCounter()
	assert.Equal(t, largeValue, value)
}

// ============================================================================
// Test Suite 5: Histogram Metrics Tests
// ============================================================================

func TestCounters_IncrementCorrectly(t *testing.T) {
	metrics := NewMetricsRegistry()

	// Setup - reset counters
	metrics.eventCounter.Set(0)
	metrics.sizeCounter.Set(0)
	metrics.packetErrorCounter.Set(0)

	// Act - increment multiple times
	for i := 0; i < 10; i++ {
		metrics.eventCounter.Inc()
		metrics.sizeCounter.Add(100)
		metrics.packetErrorCounter.Inc()
	}

	// Assert
	assert.Equal(t, int64(10), metrics.GetEventCounter())
	assert.Equal(t, int64(1000), metrics.GetBytesCounter())
	assert.Equal(t, int64(10), metrics.GetPacketErrorCounter())
}

func TestHistogram_TimeToGdcHistogram_Observation(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.timeToGdcHistogram)

	// Make some observations
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(5.0)
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(10.0)
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(15.0)

	// Verify histogram has observations
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Find the histogram
	histogramFound := false
	var sampleCount uint64
	for _, mf := range metricFamilies {
		if mf.GetName() == "ldc_to_gdc_time_histogram" {
			histogramFound = true
			if len(mf.GetMetric()) > 0 {
				histo := mf.GetMetric()[0].GetHistogram()
				sampleCount = histo.GetSampleCount()
			}
			break
		}
	}

	assert.True(t, histogramFound, "timeToGdcHistogram not found in registry")
	assert.Equal(t, uint64(3), sampleCount)
}

func TestHistogram_TimeToGdcHistogram_Buckets(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.timeToGdcHistogram)

	// Observe values that should fall into different buckets
	testValues := []float64{50, 150, 350, 550, 750, 950, 1150, 1550, 1950}
	for _, v := range testValues {
		metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(v)
	}

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Find and verify histogram
	for _, mf := range metricFamilies {
		if mf.GetName() == "ldc_to_gdc_time_histogram" {
			if len(mf.GetMetric()) > 0 {
				histo := mf.GetMetric()[0].GetHistogram()
				assert.Equal(t, uint64(9), histo.GetSampleCount(), "Expected 9 observations")
			}
			break
		}
	}
}

func TestHistogram_TimeToGdcHistogram_OutOfRange(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.timeToGdcHistogram)

	// Observe value larger than max bucket (max is 2000)
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(5000) // Should go into +Inf bucket

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Verify observation was recorded
	for _, mf := range metricFamilies {
		if mf.GetName() == "ldc_to_gdc_time_histogram" {
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
	registry.MustRegister(metrics.timeToGdcHistogram)

	// Concurrent observations
	numGoroutines := 100
	observationsPerGoroutine := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < observationsPerGoroutine; j++ {
				metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(float64(j))
			}
		}()
	}

	wg.Wait()

	// Verify all observations were recorded
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	expectedCount := uint64(numGoroutines * observationsPerGoroutine)
	for _, mf := range metricFamilies {
		if mf.GetName() == "ldc_to_gdc_time_histogram" {
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

func TestGaugeVec_ReceiveChannelCounters_WithLabels(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set values with different labels
	metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": "1"}).Set(10)
	metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": "2"}).Set(20)

	// Verify by registering and gathering
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.receiveChannelCounters)

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Should have 2 metrics with different label values
	for _, mf := range metricFamilies {
		if mf.GetName() == "ldc_receive_channel_count" {
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 2)
		}
	}
}

func TestGaugeVec_ReceiveChannelCounters_MultipleLabels(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.receiveChannelCounters)

	// Test multiple label combinations
	equipmentIDs := []string{"1", "2", "3", "5"}
	for _, id := range equipmentIDs {
		metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": id}).Set(1)
	}

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Verify we have metrics for all labels
	for _, mf := range metricFamilies {
		if mf.GetName() == "ldc_receive_channel_count" {
			assert.Equal(t, len(equipmentIDs), len(mf.GetMetric()))
		}
	}
}

func TestGaugeVec_ReceiveChannelCounters_ConcurrentLabelAccess(t *testing.T) {
	metrics := createTestMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.receiveChannelCounters)

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent writes with different labels
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			label := fmt.Sprintf("%d", idx%10) // 10 different labels
			metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": label}).Set(float64(idx))
		}(i)
	}

	wg.Wait()

	// Verify metrics were created
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if mf.GetName() == "ldc_receive_channel_count" {
			assert.GreaterOrEqual(t, len(mf.GetMetric()), 10)
		}
	}
}

// ============================================================================
// Test Suite 7: Prometheus HTTP Endpoint Tests
// ============================================================================

func TestIncompleteEventsCounter_IncrementAndDecrement(t *testing.T) {
	metrics := NewMetricsRegistry()

	// Setup - reset counter
	metrics.incompleteEventsCounter.Set(0)

	// Act - increment 3 times (3 incomplete events)
	metrics.incompleteEventsCounter.Inc()
	metrics.incompleteEventsCounter.Inc()
	metrics.incompleteEventsCounter.Inc()

	// Assert - verify 3 incomplete events
	count, err := duck.GetPrometheusGaugeValue(metrics.incompleteEventsCounter)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Simulate one event completing
	metrics.incompleteEventsCounter.Dec()

	// Assert - verify 2 incomplete events remaining
	count, err = duck.GetPrometheusGaugeValue(metrics.incompleteEventsCounter)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestMetricsRegistry_Reset_AllCountersToZero(t *testing.T) {
	metrics := NewMetricsRegistry()

	// Set all counters to non-zero values
	metrics.eventCounter.Set(100)
	metrics.sizeCounter.Set(2048)
	metrics.packetErrorCounter.Set(5)
	metrics.incompleteEventsCounter.Set(10)
	metrics.networkBufferErrorCounter.Set(7)

	// Reset
	metrics.Reset()

	// Verify all counters are zero
	assert.Equal(t, int64(0), metrics.GetEventCounter())
	assert.Equal(t, int64(0), metrics.GetBytesCounter())
	assert.Equal(t, int64(0), metrics.GetPacketErrorCounter())
}

func TestMetricsRegistry_Reset_Idempotent(t *testing.T) {
	metrics := NewMetricsRegistry()

	// Set some values
	metrics.eventCounter.Set(123)
	metrics.sizeCounter.Set(456)

	// Reset multiple times
	metrics.Reset()
	metrics.Reset()
	metrics.Reset()

	// Values should still be zero
	assert.Equal(t, int64(0), metrics.GetEventCounter())
	assert.Equal(t, int64(0), metrics.GetBytesCounter())
}

func TestMetricsRegistry_Reset_PartiallySetCounters(t *testing.T) {
	metrics := NewMetricsRegistry()

	// Only set some counters
	metrics.eventCounter.Set(100)
	metrics.packetErrorCounter.Set(5)
	// Leave others at default (0)

	// Reset
	metrics.Reset()

	// Verify all counters are zero
	assert.Equal(t, int64(0), metrics.GetEventCounter())
	assert.Equal(t, int64(0), metrics.GetPacketErrorCounter())

	// Verify unset counters are also zero
	assert.Equal(t, int64(0), metrics.GetBytesCounter())
}

func TestMetricsRegistry_HistogramsAcrossReset(t *testing.T) {
	metrics := NewMetricsRegistry()
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.timeToGdcHistogram)

	// Make observations
	for i := 0; i < 10; i++ {
		metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(float64(i))
	}

	// Reset gauges (note: Reset() doesn't reset histograms)
	metrics.Reset()

	// Histogram should still have observations
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if mf.GetName() == "ldc_to_gdc_time_histogram" {
			if len(mf.GetMetric()) > 0 {
				histo := mf.GetMetric()[0].GetHistogram()
				assert.Equal(t, uint64(10), histo.GetSampleCount(), "Histogram should retain observations after Reset()")
			}
			break
		}
	}
}

// ============================================================================
// Test Suite 4: Getter Methods Tests
// ============================================================================

func TestNewMetricsRegistry_CreatesAllMetrics(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Verify all gauge counters are initialized
	assert.NotNil(t, metrics.eventCounter)
	assert.NotNil(t, metrics.sizeCounter)
	assert.NotNil(t, metrics.packetErrorCounter)
	assert.NotNil(t, metrics.incompleteEventsCounter)
	assert.NotNil(t, metrics.equipmentDataChannelCounter)
	assert.NotNil(t, metrics.receiveChannelCounters)
	assert.NotNil(t, metrics.networkBufferErrorCounter)
	assert.NotNil(t, metrics.timeToGdcHistogram)
}

func TestNewMetricsRegistry_UniqueMetricNames(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Collect metric names and verify uniqueness
	registry := prometheus.NewRegistry()
	err := metrics.Register(registry)
	require.NoError(t, err)

	// Set at least one label value for GaugeVec/HistogramVec metrics
	metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": "test_eq"}).Set(0)
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "test_gdc"}).Observe(0)

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

	// Verify expected number of metrics (8 unique metrics)
	expectedMetricCount := 8
	if len(metricNames) != expectedMetricCount {
		t.Errorf("Expected %d unique metrics, got %d", expectedMetricCount, len(metricNames))
	}

	// Verify specific metric names exist
	expectedNames := []string{
		"ldc_event_count",
		"ldc_data_bytes",
		"ldc_packet_error_count",
		"ldc_incomplete_evts_count",
		"ldc_equipment_data_channel_count",
		"ldc_receive_channel_count",
		"ldc_network_buffer_error_count",
		"ldc_to_gdc_time_histogram",
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

	// Set at least one label value for Vec metrics
	metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": "test"}).Set(0)
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "test"}).Observe(0)

	// Verify metrics are registered by gathering them
	metricFamilies, err := registry.Gather()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(metricFamilies), 8)
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

// TestMetricsRegistry_ServesMetricsEndpoint tests that metrics can be scraped via HTTP
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
	duck.AssertMetricExists(t, content, "ldc_event_count")
	duck.AssertMetricValue(t, content, "ldc_event_count", "123")

	duck.AssertMetricExists(t, content, "ldc_data_bytes")
	duck.AssertMetricValue(t, content, "ldc_data_bytes", "456789")

	// Verify HELP and TYPE comments
	assert.Contains(t, content, "# HELP ldc_event_count No of events processed by LDC")
	assert.Contains(t, content, "# TYPE ldc_event_count gauge")
}

func TestPrometheusEndpoint_HistogramMetrics(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Make some histogram observations
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(5.0)
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(10.0)

	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	// Scrape metrics
	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	// Verify histogram metric
	duck.AssertMetricContains(t, content, "ldc_to_gdc_time_histogram")
	assert.Contains(t, content, "# HELP ldc_to_gdc_time_histogram Time taken to send each event data to GDC")
	assert.Contains(t, content, "# TYPE ldc_to_gdc_time_histogram histogram")
	assert.Contains(t, content, "ldc_to_gdc_time_histogram_bucket")
	assert.Contains(t, content, "ldc_to_gdc_time_histogram_sum")
	assert.Contains(t, content, "ldc_to_gdc_time_histogram_count")
}

func TestPrometheusEndpoint_GaugeVecWithLabels(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set labeled metrics
	metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": "1"}).Set(10)
	metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": "2"}).Set(20)

	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	// Scrape metrics
	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	// Verify labeled metrics exist
	assert.Contains(t, content, `ldc_receive_channel_count{equipment="1"} 10`)
	assert.Contains(t, content, `ldc_receive_channel_count{equipment="2"} 20`)
}

func TestPrometheusEndpoint_AllMetricsExposed(t *testing.T) {
	metrics := createTestMetricsRegistry()

	// Set label values for Vec metrics so they appear in output
	metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": "1"}).Set(0)
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(0)

	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	// Scrape metrics
	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	// Verify all expected metrics are present
	expectedMetrics := []string{
		"ldc_event_count",
		"ldc_data_bytes",
		"ldc_packet_error_count",
		"ldc_incomplete_evts_count",
		"ldc_equipment_data_channel_count",
		"ldc_receive_channel_count",
		"ldc_network_buffer_error_count",
		"ldc_to_gdc_time_histogram",
	}

	for _, metric := range expectedMetrics {
		duck.AssertMetricExists(t, content, metric)
	}
}

// ============================================================================
// Test Suite 8: Concurrency Tests
// ============================================================================

// TestMetricsRegistry_ScrapableByPrometheus tests Prometheus format compatibility
func TestMetricsRegistry_ScrapableByPrometheus(t *testing.T) {
	// Setup
	metrics := NewMetricsRegistry()
	registry := prometheus.NewRegistry()
	err := metrics.Register(registry)
	require.NoError(t, err)

	// Set various metric types
	metrics.eventCounter.Set(100)
	metrics.incompleteEventsCounter.Set(5)
	metrics.networkBufferErrorCounter.Set(2)

	// Create test server
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Act
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// Assert - check Prometheus format elements
	// HELP comments
	assert.Contains(t, bodyStr, "# HELP ldc_event_count")
	assert.Contains(t, bodyStr, "# HELP ldc_data_bytes")
	assert.Contains(t, bodyStr, "# HELP ldc_incomplete_evts_count")

	// TYPE declarations
	assert.Contains(t, bodyStr, "# TYPE ldc_event_count gauge")
	assert.Contains(t, bodyStr, "# TYPE ldc_data_bytes gauge")
	assert.Contains(t, bodyStr, "# TYPE ldc_incomplete_evts_count gauge")
}

// TestMetricsRegistry_AllMetricsRegistered tests that all metrics are properly registered
func TestMetricsRegistry_AllMetricsRegistered(t *testing.T) {
	// Setup
	metrics := NewMetricsRegistry()
	registry := prometheus.NewRegistry()
	err := metrics.Register(registry)
	require.NoError(t, err)

	// Set values to make metrics appear in output
	metrics.eventCounter.Set(1)
	metrics.sizeCounter.Set(1)
	metrics.packetErrorCounter.Set(1)
	metrics.incompleteEventsCounter.Set(1)
	metrics.equipmentDataChannelCounter.Set(1)
	metrics.networkBufferErrorCounter.Set(1)
	// For vector metrics, we need to set with labels
	metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": "1"}).Set(1)
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(100)

	// Create test server
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Act
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// Assert - all metric names should be present
	expectedMetrics := []string{
		"ldc_event_count",
		"ldc_data_bytes",
		"ldc_packet_error_count",
		"ldc_incomplete_evts_count",
		"ldc_equipment_data_channel_count",
		"ldc_receive_channel_count",
		"ldc_network_buffer_error_count",
		"ldc_to_gdc_time_histogram",
	}

	for _, metric := range expectedMetrics {
		assert.Contains(t, bodyStr, metric, "Missing metric: %s", metric)
	}
}

// TestMetricsRegistry_ReceiveChannelCounters_PerEquipment tests per-equipment receive channel metrics
func TestMetricsRegistry_ReceiveChannelCounters_PerEquipment(t *testing.T) {
	// Setup
	metrics := NewMetricsRegistry()
	registry := prometheus.NewRegistry()
	err := metrics.Register(registry)
	require.NoError(t, err)

	// Set values for multiple equipments
	equipmentIDs := []string{"1", "2", "3", "5"}
	values := []float64{10, 20, 30, 50}

	for i, id := range equipmentIDs {
		metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": id}).Set(values[i])
	}

	// Create test server
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Act
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// Assert - each equipment should have its own metric with label
	for _, id := range equipmentIDs {
		expectedLine := strings.Contains(bodyStr, "ldc_receive_channel_count{equipment=\""+id+"\"}")
		assert.True(t, expectedLine, "Missing metric for equipment %s", id)
		// Verify the equipment label is present
		assert.Contains(t, bodyStr, "equipment=\""+id+"\"", "Missing equipment label %s", id)
	}
}

// TestMetricsRegistry_HistogramBuckets_Configured tests that histogram has correct buckets
func TestMetricsRegistry_HistogramBuckets_Configured(t *testing.T) {
	// Setup
	metrics := NewMetricsRegistry()
	registry := prometheus.NewRegistry()
	err := metrics.Register(registry)
	require.NoError(t, err)

	// Add observations that fall into different buckets
	testValues := []float64{50, 150, 350, 550, 750, 950, 1150, 1550, 1950}
	for _, v := range testValues {
		metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(v)
	}

	// Create test server
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Act
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// Assert - histogram buckets should be present
	assert.Contains(t, bodyStr, "ldc_to_gdc_time_histogram_bucket")
	assert.Contains(t, bodyStr, "ldc_to_gdc_time_histogram_sum")
	assert.Contains(t, bodyStr, "ldc_to_gdc_time_histogram_count")

	// Check specific bucket boundaries exist
	expectedBuckets := []string{"100", "200", "500", "1000", "1500", "2000"}
	for _, bucket := range expectedBuckets {
		assert.Contains(t, bodyStr, "le=\""+bucket+"\"", "Missing bucket boundary %s", bucket)
	}
}

// TestMetricsRegistry_TimeToGdcHistogram_MultipleGDCs tests histogram with multiple GDCs
func TestMetricsRegistry_TimeToGdcHistogram_MultipleGDCs(t *testing.T) {
	// Setup
	metrics := NewMetricsRegistry()
	registry := prometheus.NewRegistry()
	err := metrics.Register(registry)
	require.NoError(t, err)

	// Add observations for multiple GDCs
	gdcIDs := []string{"0", "1", "2"}
	for _, gdc := range gdcIDs {
		metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": gdc}).Observe(100)
		metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": gdc}).Observe(200)
	}

	// Create test server
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Act
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// Assert - each GDC should have histogram entries
	for _, gdc := range gdcIDs {
		assert.Contains(t, bodyStr, "gdc=\""+gdc+"\"", "Missing GDC label %s", gdc)
	}

	// Each GDC should have count of 2
	assert.Contains(t, bodyStr, "ldc_to_gdc_time_histogram_count")
}

// TestMetricsRegistry_NetworkBufferErrorCounter_Increments tests network buffer error tracking
func TestMetricsRegistry_NetworkBufferErrorCounter_Increments(t *testing.T) {
	// Setup
	metrics := NewMetricsRegistry()
	registry := prometheus.NewRegistry()
	err := metrics.Register(registry)
	require.NoError(t, err)

	// Reset and increment
	metrics.networkBufferErrorCounter.Set(0)
	for i := 0; i < 5; i++ {
		metrics.networkBufferErrorCounter.Inc()
	}

	// Create test server
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	server := httptest.NewServer(handler)
	defer server.Close()

	// Act
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(body)

	// Assert
	assert.Contains(t, bodyStr, "ldc_network_buffer_error_count 5")
}

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
	registry.MustRegister(metrics.timeToGdcHistogram)

	numGoroutines := 100
	observationsPerGoroutine := 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < observationsPerGoroutine; j++ {
				metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(float64(j))
			}
		}(i)
	}

	wg.Wait()

	// Verify total count
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	expectedCount := uint64(numGoroutines * observationsPerGoroutine)
	for _, mf := range metricFamilies {
		if mf.GetName() == "ldc_to_gdc_time_histogram" {
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
	registry.MustRegister(metrics.receiveChannelCounters)

	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent writes with different labels
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			label := fmt.Sprintf("%d", idx%10)
			metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": label}).Set(float64(idx))
		}(i)

		// Concurrent reads
		go func(idx int) {
			defer wg.Done()
			label := fmt.Sprintf("%d", idx%10)
			metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": label}).Inc()
		}(i)
	}

	wg.Wait()

	// Verify no races
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "ldc_receive_channel_count" {
			found = true
			break
		}
	}
	assert.True(t, found, "receiveChannelCounters not found in registry")
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
			metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(float64(idx))
		}(i)

		// GaugeVec operations
		go func(idx int) {
			defer wg.Done()
			label := fmt.Sprintf("%d", idx%5)
			metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": label}).Set(float64(idx))
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
	_ = metrics.GetPacketErrorCounter()
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
	metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "1"}).Observe(5.0)
	metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": "1"}).Set(3)

	// Scrape via HTTP
	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	// Verify metrics are exposed
	duck.AssertMetricExists(t, content, "ldc_event_count")
	duck.AssertMetricValue(t, content, "ldc_event_count", "100")
	duck.AssertMetricExists(t, content, "ldc_to_gdc_time_histogram")

	// Reset
	metrics.Reset()

	// Verify reset
	assert.Equal(t, int64(0), metrics.GetEventCounter())

	// Verify via HTTP
	contentAfterReset, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)
	duck.AssertMetricValue(t, contentAfterReset, "ldc_event_count", "0")
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
		metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": "0"}).Observe(float64(i%25))
	}

	// Some errors occur
	metrics.packetErrorCounter.Set(3)
	metrics.incompleteEventsCounter.Set(5)

	// Equipment channels have data
	metrics.equipmentDataChannelCounter.Set(50)

	// Verify state via getters
	eventCount := metrics.GetEventCounter()
	assert.Equal(t, int64(100), eventCount)

	byteCount := metrics.GetBytesCounter()
	assert.Equal(t, int64(102400), byteCount) // 100 * 1024

	errorCount := metrics.GetPacketErrorCounter()
	assert.Equal(t, int64(3), errorCount)

	// Verify via HTTP
	server, _ := duck.CreateTestPrometheusServer(t, metrics)
	defer server.Close()

	content, err := duck.ScrapeMetricsEndpoint(server.URL)
	require.NoError(t, err)

	duck.AssertMetricValue(t, content, "ldc_event_count", "100")
	duck.AssertMetricValue(t, content, "ldc_data_bytes", "102400")
	duck.AssertMetricValue(t, content, "ldc_packet_error_count", "3")
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
	metrics.packetErrorCounter.Set(0)
	zeroVal := metrics.GetPacketErrorCounter()
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
	metrics.packetErrorCounter.Set(5)

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
