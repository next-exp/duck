package main

import (
	"context"
	"time"

	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/prometheus/client_golang/prometheus"
)

// apiMetrics reads run state on each scrape so it remains correct after an API restart.
type apiMetrics struct {
	queries         database.Querier
	transition      *RunTransition
	active          *prometheus.Desc
	runNumber       *prometheus.Desc
	readSuccess     *prometheus.Desc
	transitionState *prometheus.Desc
}

func newAPIMetrics(queries database.Querier, transition *RunTransition) *apiMetrics {
	return &apiMetrics{
		queries:         queries,
		transition:      transition,
		active:          prometheus.NewDesc("duck_api_run_active", "Whether the latest run has started and has not stopped (1=yes, 0=no).", nil, nil),
		runNumber:       prometheus.NewDesc("duck_api_run_number", "Latest run number in the database.", nil, nil),
		readSuccess:     prometheus.NewDesc("duck_api_run_state_read_success", "Whether the latest run state was read successfully from the database (1=yes, 0=no).", nil, nil),
		transitionState: prometheus.NewDesc("duck_api_run_transition_state", "Current run control transition (one state has value 1).", []string{"state"}, nil),
	}
}

func (m *apiMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.active
	ch <- m.runNumber
	ch <- m.readSuccess
	ch <- m.transitionState
}

func (m *apiMetrics) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	run, err := m.queries.GetLatestRunWithTimestamp(ctx)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(m.readSuccess, prometheus.GaugeValue, 0)
	} else {
		ch <- prometheus.MustNewConstMetric(m.readSuccess, prometheus.GaugeValue, 1)
		ch <- prometheus.MustNewConstMetric(m.runNumber, prometheus.GaugeValue, float64(run.ID))
		active := 0.0
		if run.Start.Valid && !run.Stop.Valid {
			active = 1
		}
		ch <- prometheus.MustNewConstMetric(m.active, prometheus.GaugeValue, active)
	}
	state, _ := m.transition.Status()
	for _, name := range []string{"idle", "starting", "stopping", "error"} {
		value := 0.0
		if name == stateString(state) {
			value = 1
		}
		ch <- prometheus.MustNewConstMetric(m.transitionState, prometheus.GaugeValue, value, name)
	}
}
