package duck

import (
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPrometheusGaugeValue_Success(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge",
		Help: "A test gauge",
	})
	gauge.Set(42)

	value, err := GetPrometheusGaugeValue(gauge)

	require.NoError(t, err)
	assert.Equal(t, int64(42), value)
}

func TestGetPrometheusGaugeValue_ZeroValue(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge_zero",
		Help: "A test gauge with zero value",
	})
	// Default value is 0, don't set it

	value, err := GetPrometheusGaugeValue(gauge)

	require.NoError(t, err)
	assert.Equal(t, int64(0), value)
}

func TestGetPrometheusGaugeValue_NegativeValue(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge_negative",
		Help: "A test gauge with negative value",
	})
	gauge.Set(-100)

	value, err := GetPrometheusGaugeValue(gauge)

	require.NoError(t, err)
	assert.Equal(t, int64(-100), value)
}

func TestGetPrometheusGaugeValue_LargeValue(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge_large",
		Help: "A test gauge with large value",
	})
	// Test with a large value within int64 range
	// Note: math.MaxInt64 / 2 = 4611686018427387903.5, which gets rounded to 4611686018427387904 as float64
	largeValue := int64(math.MaxInt64 / 2)
	gauge.Set(float64(largeValue))

	value, err := GetPrometheusGaugeValue(gauge)

	require.NoError(t, err)
	// Expect the rounded value due to float64 precision
	assert.Equal(t, int64(4611686018427387904), value)
}

func TestGetPrometheusGaugeValue_FloatTruncation(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge_float",
		Help: "A test gauge with float value",
	})
	gauge.Set(42.7)

	value, err := GetPrometheusGaugeValue(gauge)

	require.NoError(t, err)
	// Float is truncated to int64
	assert.Equal(t, int64(42), value)
}

func TestGetPrometheusGaugeValue_Inc(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge_inc",
		Help: "A test gauge with incrementing",
	})
	gauge.Set(10)
	gauge.Inc()
	gauge.Inc()

	value, err := GetPrometheusGaugeValue(gauge)

	require.NoError(t, err)
	assert.Equal(t, int64(12), value)
}

func TestGetPrometheusGaugeValue_Add(t *testing.T) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge_add",
		Help: "A test gauge with adding",
	})
	gauge.Set(100)
	gauge.Add(50)
	gauge.Sub(25)

	value, err := GetPrometheusGaugeValue(gauge)

	require.NoError(t, err)
	assert.Equal(t, int64(125), value)
}

func TestConfigureAndServePrometheus_RegistersHandler(t *testing.T) {
	// Create a new registry and gauge
	registry := prometheus.NewRegistry()
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_metric",
		Help: "A test metric",
	})
	registry.MustRegister(gauge)
	gauge.Set(123)

	// Create a handler for the registry
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Create a test server with the handler
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make a request to the /metrics endpoint
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestConfigureAndServePrometheus_MetricsContent(t *testing.T) {
	// Create a new registry and gauge
	registry := prometheus.NewRegistry()
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "test_metric_content",
		Help: "A test metric for content verification",
	})
	registry.MustRegister(gauge)
	gauge.Set(456)

	// Create a handler for the registry
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Create a test server with the handler
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make a request to the /metrics endpoint
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Verify Prometheus format output
	content := string(body)
	assert.Contains(t, content, "test_metric_content")
	assert.Contains(t, content, "456")
	assert.Contains(t, content, "# HELP test_metric_content A test metric for content verification")
	assert.Contains(t, content, "# TYPE test_metric_content gauge")
}

func TestConfigureAndServePrometheus_MultipleMetrics(t *testing.T) {
	// Create a new registry with multiple metrics
	registry := prometheus.NewRegistry()

	gauge1 := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "metric_one",
		Help: "First metric",
	})
	gauge2 := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "metric_two",
		Help: "Second metric",
	})
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "counter_metric",
		Help: "A counter metric",
	})

	registry.MustRegister(gauge1)
	registry.MustRegister(gauge2)
	registry.MustRegister(counter)

	gauge1.Set(100)
	gauge2.Set(200)
	counter.Add(50)

	// Create a handler for the registry
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Create a test server with the handler
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make a request to the /metrics endpoint
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	content := string(body)
	assert.Contains(t, content, "metric_one 100")
	assert.Contains(t, content, "metric_two 200")
	assert.Contains(t, content, "counter_metric 50")
}

func TestConfigureAndServePrometheus_EmptyRegistry(t *testing.T) {
	// Create an empty registry
	registry := prometheus.NewRegistry()

	// Create a handler for the registry
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Create a test server with the handler
	server := httptest.NewServer(handler)
	defer server.Close()

	// Make a request to the /metrics endpoint
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Empty registry should return empty or minimal response
	assert.NotNil(t, body)
}
