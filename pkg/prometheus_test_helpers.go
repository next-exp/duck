package duck

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PrometheusTestable is an interface for types that can register Prometheus metrics
type PrometheusTestable interface {
	Register(registry *prometheus.Registry) error
}

// scrapeMetricsEndpoint makes HTTP request to metrics endpoint and returns content
func ScrapeMetricsEndpoint(serverURL string) (string, error) {
	resp, err := http.Get(serverURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// CreateTestPrometheusServer creates a test HTTP server with prometheus handler
func CreateTestPrometheusServer(t *testing.T, metrics PrometheusTestable) (*httptest.Server, *prometheus.Registry) {
	t.Helper()
	registry := prometheus.NewRegistry()
	err := metrics.Register(registry)
	require.NoError(t, err)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	server := httptest.NewServer(handler)

	return server, registry
}

// AssertMetricExists verifies that a metric exists in the scraped metrics content
func AssertMetricExists(t *testing.T, metricsContent, metricName string) {
	t.Helper()
	if !strings.Contains(metricsContent, metricName) {
		t.Errorf("Metric %q not found in metrics output", metricName)
	}
}

// AssertMetricValue verifies that a metric has a specific value in the scraped content
func AssertMetricValue(t *testing.T, metricsContent, metricName, value string) {
	t.Helper()
	expectedLine := fmt.Sprintf("%s %s", metricName, value)
	if !strings.Contains(metricsContent, expectedLine) {
		t.Errorf("Expected line %q not found in metrics output", expectedLine)
	}
}

// AssertMetricContains verifies that a metric label/value combination exists
func AssertMetricContains(t *testing.T, metricsContent, metricPattern string) {
	t.Helper()
	if !strings.Contains(metricsContent, metricPattern) {
		t.Errorf("Expected pattern %q not found in metrics output", metricPattern)
	}
}

// AssertPrometheusFormat verifies basic Prometheus format elements
func AssertPrometheusFormat(t *testing.T, content string) {
	t.Helper()
	assert.Contains(t, content, "# HELP")
	assert.Contains(t, content, "# TYPE")
}

// VerifyUniqueMetricNames collects metric names and verifies uniqueness
func VerifyUniqueMetricNames(t *testing.T, registry *prometheus.Registry, expectedCount int, expectedNames []string) {
	t.Helper()

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

	if len(metricNames) != expectedCount {
		t.Errorf("Expected %d unique metrics, got %d", expectedCount, len(metricNames))
	}

	for _, name := range expectedNames {
		if !metricNames[name] {
			t.Errorf("Expected metric %q not found", name)
		}
	}
}
