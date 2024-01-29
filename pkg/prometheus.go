package duck

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

func GetPrometheusGaugeValue(gauge prometheus.Gauge) (int64, error) {
	value := int64(0)
	var m = &dto.Metric{}
	err := gauge.Write(m)
	value = int64(m.Gauge.GetValue())
	return value, err
}

func servePrometheus(port int, logger DuckLogger) {
	prometheusAddr := fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(prometheusAddr, nil)
	if err != nil {
		logger.Slog.Error(err.Error())
	}
}

func ConfigureAndServePrometheus(r *prometheus.Registry, port int, logger DuckLogger) {
	handler := promhttp.HandlerFor(r, promhttp.HandlerOpts{})
	http.Handle("/metrics", handler)
	go servePrometheus(port, logger)
	//InfoLog.Printf("Prometheus listening on port %d\n", port)
}
