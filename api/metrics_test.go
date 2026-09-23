package main

import (
	"database/sql"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/mock"
)

func TestAPIMetrics(t *testing.T) {
	for _, tc := range []struct {
		name   string
		run    database.Run
		err    error
		want   []string
		absent []string
	}{
		{"active", database.Run{ID: 42, Start: sql.NullTime{Valid: true}}, nil, []string{"duck_api_run_active 1", "duck_api_run_number 42", "duck_api_run_state_read_success 1"}, nil},
		{"inactive", database.Run{ID: 43}, nil, []string{"duck_api_run_active 0", "duck_api_run_state_read_success 1"}, nil},
		{"database error", database.Run{}, errors.New("unavailable"), []string{"duck_api_run_state_read_success 0"}, []string{"duck_api_run_active ", "duck_api_run_number "}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := new(mocks.Querier)
			q.On("GetLatestRunWithTimestamp", mock.Anything).Return(tc.run, tc.err).Once()
			registry := prometheus.NewRegistry()
			registry.MustRegister(newAPIMetrics(q, &RunTransition{}))
			recorder := httptest.NewRecorder()
			promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
			if recorder.Code != 200 {
				t.Fatalf("status = %d: %s", recorder.Code, recorder.Body.String())
			}
			body := recorder.Body.String()
			for _, s := range tc.want {
				if !strings.Contains(body, s) {
					t.Errorf("missing %q", s)
				}
			}
			for _, s := range tc.absent {
				if strings.Contains(body, s) {
					t.Errorf("unexpected %q", s)
				}
			}
			if !strings.Contains(body, `duck_api_run_transition_state{state="idle"} 1`) {
				t.Error("missing idle transition")
			}
			q.AssertExpectations(t)
		})
	}
}
