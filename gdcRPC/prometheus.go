package main

import (
	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsRegistry holds all prometheus metrics for the GDC server
type MetricsRegistry struct {
	eventCounter                    prometheus.Gauge
	sizeCounter                     prometheus.Gauge
	evtErrorCounter                 prometheus.Gauge
	incompleteEventsCounter         prometheus.Gauge
	dataChannelCounter              prometheus.Gauge
	writeTimeHistogram              prometheus.Histogram
	h5WriteTimeHistogram            prometheus.Histogram
	decodeTimeHistogram             prometheus.Histogram
	h5FilesOpenedCounter            *prometheus.GaugeVec
	filesOpenedCounter              prometheus.Gauge
	evtsWaitingForHdf5WriterCounter *prometheus.GaugeVec
	subRunCounter                   prometheus.Gauge
	evtsWaitingBinaryWriterCounter  prometheus.Gauge
	evtsWaitingDecodeCounter        prometheus.Gauge
	evtsPerTriggerTypeCounter       *prometheus.GaugeVec
	evtDecoderErrorCounter          prometheus.Gauge
}

// NewMetricsRegistry creates and initializes a new MetricsRegistry
func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{
		eventCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gdc_event_count",
				Help: "No of events processed by GDC",
			},
		),
		sizeCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gdc_data_bytes",
				Help: "No of bytes received by GDC",
			},
		),
		evtErrorCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gdc_evt_error_count",
				Help: "No of events errors detected by GDC",
			},
		),
		incompleteEventsCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gdc_incomplete_evts_count",
				Help: "No of incomplete events waiting to be mounted by GDC",
			},
		),
		dataChannelCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gdc_data_channel_count",
				Help: "No of elements waiting in GDC dataChannel",
			},
		),
		writeTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gdc_write_time_histogram",
				Help:    "Time taken to write each event",
				Buckets: []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25},
			}),
		h5WriteTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gdc_hdf5_write_time_histogram",
				Help:    "Time taken to write each event into an hdf5 file",
				Buckets: []float64{0, 50, 75, 100, 125, 150, 175, 200, 250, 300, 350, 400, 450, 500, 1000, 2000},
			}),
		decodeTimeHistogram: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gdc_decoder_time_histogram",
				Help:    "Time taken to decode each event",
				Buckets: []float64{0, 500, 600, 700, 800, 900, 1000, 1500, 2000, 3000},
			}),
		h5FilesOpenedCounter: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gdc_opened_hdf5_files_count",
				Help: "No of hdf5 currently opened in the GDC",
			},
			[]string{"trigger"},
		),
		filesOpenedCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gdc_opened_files_count",
				Help: "No of binary files currently opened in the GDC",
			},
		),
		evtsWaitingForHdf5WriterCounter: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gdc_events_waiting_hdf5_write_count",
				Help: "No of events waiting to be written in an hdf5 file",
			},
			[]string{"trigger"},
		),
		subRunCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gdc_subrun_count",
				Help: "Current subrun number",
			},
		),
		evtsWaitingBinaryWriterCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gdc_events_waiting_binary_write_count",
				Help: "No of events awaiting to be written in a binary file",
			},
		),
		evtsWaitingDecodeCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gdc_events_waiting_decode_count",
				Help: "No of events waiting to be decoded",
			},
		),
		evtsPerTriggerTypeCounter: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gdc_events_per_trigger_type_count",
				Help: "No of decoded events per trigger type",
			},
			[]string{"trigger"},
		),
		evtDecoderErrorCounter: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "gdc_decoder_error_count",
				Help: "No of decoder errors",
			},
		),
	}
}

// Register registers all metrics with the prometheus registry
func (m *MetricsRegistry) Register(r *prometheus.Registry) error {
	r.MustRegister(m.eventCounter)
	r.MustRegister(m.sizeCounter)
	r.MustRegister(m.evtErrorCounter)
	r.MustRegister(m.incompleteEventsCounter)
	r.MustRegister(m.dataChannelCounter)
	r.MustRegister(m.writeTimeHistogram)
	r.MustRegister(m.h5WriteTimeHistogram)
	r.MustRegister(m.decodeTimeHistogram)
	r.MustRegister(m.h5FilesOpenedCounter)
	r.MustRegister(m.filesOpenedCounter)
	r.MustRegister(m.evtsWaitingForHdf5WriterCounter)
	r.MustRegister(m.subRunCounter)
	r.MustRegister(m.evtsWaitingDecodeCounter)
	r.MustRegister(m.evtsWaitingBinaryWriterCounter)
	r.MustRegister(m.evtsPerTriggerTypeCounter)
	r.MustRegister(m.evtDecoderErrorCounter)
	return nil
}

// Reset resets all counter metrics to zero
func (m *MetricsRegistry) Reset() {
	m.eventCounter.Set(0)
	m.sizeCounter.Set(0)
	m.evtErrorCounter.Set(0)
	m.incompleteEventsCounter.Set(0)
	m.filesOpenedCounter.Set(0)
	m.subRunCounter.Set(0)
	m.evtsWaitingDecodeCounter.Set(0)
	m.evtDecoderErrorCounter.Set(0)
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

// GetEvtErrorCounter returns the current error counter value
func (m *MetricsRegistry) GetEvtErrorCounter() int64 {
	value, err := duck.GetPrometheusGaugeValue(m.evtErrorCounter)
	if err != nil {
		return 0
	}
	return value
}

func startPrometheus(metrics *MetricsRegistry, port int, goStats bool, logger duck.DuckLogger) {
	r := prometheus.NewRegistry()
	if goStats {
		r.MustRegister(prometheus.NewGoCollector())
		r.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}
	err := metrics.Register(r)
	if err != nil {
		logger.Slog.Error("Failed to register metrics")
	}
	duck.ConfigureAndServePrometheus(r, port, logger)
}
