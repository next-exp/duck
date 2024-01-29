package main

import (
	"fmt"
	"net/http"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsRegistry holds all prometheus metrics for the LDC server
type MetricsRegistry struct {
	eventCounter    prometheus.Gauge
	sizeCounter     prometheus.Gauge
	packetErrorCounter prometheus.Gauge
	incompleteEventsCounter prometheus.Gauge
	equipmentDataChannelCounter prometheus.Gauge
	receiveChannelCounters *prometheus.GaugeVec
	networkBufferErrorCounter prometheus.Gauge
	timeToGdcHistogram *prometheus.HistogramVec
}

// NewMetricsRegistry creates and initializes a new MetricsRegistry
func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{
		eventCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ldc_event_count",
				Help: "No of events processed by LDC",
			},
		),
		sizeCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ldc_data_bytes",
				Help: "No of bytes received by LDC",
			},
		),
		packetErrorCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ldc_packet_error_count",
				Help: "No of sequence counter errors in packets received by LDC",
			},
		),
		incompleteEventsCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ldc_incomplete_evts_count",
				Help: "No of incomplete events waiting to be mounted by LDC",
			},
		),
		equipmentDataChannelCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ldc_equipment_data_channel_count",
				Help: "No of elements in LDC equipment data channel",
			},
		),
		receiveChannelCounters: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: "ldc_receive_channel_count", Help: "No elements in LDC receive channels"}, []string{"equipment"}),
		networkBufferErrorCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "ldc_network_buffer_error_count",
				Help: "No of overwrites in dirty positions in network buffer LDC",
			},
		),
		timeToGdcHistogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ldc_to_gdc_time_histogram",
				Help:    "Time taken to send each event data to GDC",
				Buckets: []float64{0, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 1100, 1200, 1300, 1400, 1500, 1600, 1700, 1800, 1900, 2000},
			},
			[]string{"gdc"},
		),
	}
}

// Register registers all metrics with the prometheus registry
func (m *MetricsRegistry) Register(r *prometheus.Registry) error {
	r.MustRegister(m.eventCounter)
	r.MustRegister(m.sizeCounter)
	r.MustRegister(m.packetErrorCounter)
	r.MustRegister(m.incompleteEventsCounter)
	r.MustRegister(m.equipmentDataChannelCounter)
	r.MustRegister(m.receiveChannelCounters)
	r.MustRegister(m.networkBufferErrorCounter)
	r.MustRegister(m.timeToGdcHistogram)
	return nil
}

// Reset resets all counter metrics to zero
func (m *MetricsRegistry) Reset() {
	m.eventCounter.Set(0)
	m.sizeCounter.Set(0)
	m.packetErrorCounter.Set(0)
	m.incompleteEventsCounter.Set(0)
	m.networkBufferErrorCounter.Set(0)
}

// GetEventCounter returns the current event counter value
func (m *MetricsRegistry) GetEventCounter() int64 {
	value, err := duck.GetPrometheusGaugeValue(m.eventCounter)
	if err != nil {
		return 0
	}
	return value
}

// GetBytesCounter returns the current bytes counter value
func (m *MetricsRegistry) GetBytesCounter() int64 {
	value, err := duck.GetPrometheusGaugeValue(m.sizeCounter)
	if err != nil {
		return 0
	}
	return value
}

// GetPacketErrorCounter returns the current packet error counter value
func (m *MetricsRegistry) GetPacketErrorCounter() int64 {
	value, err := duck.GetPrometheusGaugeValue(m.packetErrorCounter)
	if err != nil {
		return 0
	}
	return value
}

func startPrometheus(metrics *MetricsRegistry, port int, logger duck.DuckLogger) {
	registry := prometheus.NewRegistry()
	metrics.Register(registry)

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	prometheusAddr := fmt.Sprintf(":%d", port)
	go http.ListenAndServe(prometheusAddr, nil)
}
