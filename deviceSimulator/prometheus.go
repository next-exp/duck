package main

import (
	"fmt"
	"net/http"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	eventCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "simulator_event_count",
			Help: "Number of events replayed by the simulator",
		},
	)
	packetCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "simulator_packet_count",
			Help: "Number of UDP packets sent by the simulator",
		},
	)
	bytesCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "simulator_data_bytes",
			Help: "Number of bytes sent by the simulator",
		},
	)
	errorCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "simulator_error_count",
			Help: "Number of errors encountered by the simulator",
		},
	)
	replayRateGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "simulator_replay_rate",
			Help: "Current replay rate multiplier",
		},
	)
	errorInjectionCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "simulator_error_injection_count",
			Help: "Number of errors injected by the simulator (generate mode)",
		},
	)
)

func startPrometheus(port int) {
	prometheus.MustRegister(eventCounter)
	prometheus.MustRegister(packetCounter)
	prometheus.MustRegister(bytesCounter)
	prometheus.MustRegister(errorCounter)
	prometheus.MustRegister(replayRateGauge)
	prometheus.MustRegister(errorInjectionCounter)

	http.Handle("/metrics", promhttp.Handler())
	prometheusAddr := fmt.Sprintf(":%d", port)
	go http.ListenAndServe(prometheusAddr, nil)
}

func getEventCounter() int64 {
	value, err := duck.GetPrometheusGaugeValue(eventCounter)
	if err != nil {
		logger.Slog.Error("Error getting event counter", "error", err)
	}
	return value
}

func getPacketCounter() int64 {
	value, err := duck.GetPrometheusGaugeValue(packetCounter)
	if err != nil {
		logger.Slog.Error("Error getting packet counter", "error", err)
	}
	return value
}

func getBytesCounter() int64 {
	value, err := duck.GetPrometheusGaugeValue(bytesCounter)
	if err != nil {
		logger.Slog.Error("Error getting bytes counter", "error", err)
	}
	return value
}

func getErrorCounter() int64 {
	value, err := duck.GetPrometheusGaugeValue(errorCounter)
	if err != nil {
		logger.Slog.Error("Error getting error counter", "error", err)
	}
	return value
}

func getErrorInjectionCounter() int64 {
	value, err := duck.GetPrometheusGaugeValue(errorInjectionCounter)
	if err != nil {
		logger.Slog.Error("Error getting error injection counter", "error", err)
	}
	return value
}
