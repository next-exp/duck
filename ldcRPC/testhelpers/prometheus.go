package testhelpers

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// GetPrometheusGaugeVecValue retrieves the current value of a GaugeVec metric with specific labels
func GetPrometheusGaugeVecValue(gaugeVec *prometheus.GaugeVec, labels prometheus.Labels) (int64, error) {
	value := int64(0)
	var m = &dto.Metric{}
	err := gaugeVec.With(labels).Write(m)
	value = int64(m.Gauge.GetValue())
	return value, err
}

// GetPrometheusHistogramSampleCount retrieves the sample count from a HistogramVec by gathering from registry
// Note: This requires the histogram to be registered with a Prometheus registry
func GetPrometheusHistogramSampleCount(histogram *prometheus.HistogramVec, registry *prometheus.Registry, metricName string, labelFilter prometheus.Labels) (uint64, error) {
	metricFamilies, err := registry.Gather()
	if err != nil {
		return 0, err
	}

	for _, mf := range metricFamilies {
		if mf.GetName() == metricName {
			for _, metric := range mf.GetMetric() {
				// Check if labels match
				if labelMatches(metric.GetLabel(), labelFilter) {
					histo := metric.GetHistogram()
					if histo != nil {
						return histo.GetSampleCount(), nil
					}
				}
			}
		}
	}
	return 0, nil
}

// labelMatches checks if the metric labels match the filter
func labelMatches(labels []*dto.LabelPair, filter prometheus.Labels) bool {
	if len(filter) == 0 {
		return true
	}
	for k, v := range filter {
		found := false
		for _, label := range labels {
			if label.GetName() == k && label.GetValue() == v {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
