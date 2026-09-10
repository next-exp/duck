package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type captureTopiPublisher struct{ messages chan []byte }

func (p captureTopiPublisher) Publish(_ context.Context, _ database.Topiparam, body []byte) error {
	p.messages <- body
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetTopiConfigurationDoesNotExposeSecrets(t *testing.T) {
	q := mocks.NewQuerier(t)
	q.On("GetTopiParams", mock.Anything).Return(database.Topiparam{ApiToken: "secret", RabbitmqPassword: "password"}, nil)
	s := NewDuckAPIServerWithQuerier(q, "", "", false)
	resp, err := s.GetTopiConfiguration(context.Background(), connect.NewRequest(&pb.GetTopiConfigurationRequest{}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Configuration.HasApiToken)
	require.True(t, resp.Msg.Configuration.HasRabbitmqPassword)
}

func TestUpdateTopiConfigurationPreservesBlankSecrets(t *testing.T) {
	q := mocks.NewQuerier(t)
	q.On("GetTopiParams", mock.Anything).Return(database.Topiparam{ApiToken: "old-token", RabbitmqPassword: "old-password"}, nil)
	q.On("UpsertTopiParams", mock.Anything, mock.MatchedBy(func(p database.UpsertTopiParamsParams) bool {
		return p.ApiToken == "old-token" && p.RabbitmqPassword == "old-password"
	})).Return(nil)
	s := NewDuckAPIServerWithQuerier(q, "", "", false)
	_, err := s.UpdateTopiConfiguration(context.Background(), connect.NewRequest(&pb.UpdateTopiConfigurationRequest{Configuration: &pb.TopiConfiguration{RabbitmqPort: 5672}}))
	require.NoError(t, err)
}

func TestListTopiConfigurationsUsesBearerToken(t *testing.T) {
	var gotAuth, gotPath string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"configurations":[{"name":"kr.yml","parseError":""}],"defaultName":"kr.yml"}`)), Header: make(http.Header)}, nil
	})}
	q := mocks.NewQuerier(t)
	q.On("GetTopiParams", mock.Anything).Return(database.Topiparam{DaemonUrl: "http://topi.test", ApiToken: "monitor-token"}, nil)
	srv := NewDuckAPIServerWithQuerier(q, "", "", false)
	srv.topiHTTPClient = client
	resp, err := srv.ListTopiConfigurations(context.Background(), connect.NewRequest(&pb.ListTopiConfigurationsRequest{}))
	require.NoError(t, err)
	require.Equal(t, "Bearer monitor-token", gotAuth)
	require.Equal(t, topiListProcedure, gotPath)
	require.Len(t, resp.Msg.Configurations, 1)
	require.True(t, resp.Msg.Configurations[0].IsDefault)
}

func TestNotifyTopiStartIsAsynchronousAndUsesSelectedConfiguration(t *testing.T) {
	q := mocks.NewQuerier(t)
	q.On("GetTopiParams", mock.Anything).Return(database.Topiparam{Enabled: true, SelectedConfiguration: "kr.yml"}, nil)
	s := NewDuckAPIServerWithQuerier(q, "", "", false)
	messages := make(chan []byte, 1)
	s.topiPublisher = captureTopiPublisher{messages: messages}
	s.notifyTopi("start", 42)
	select {
	case body := <-messages:
		var message map[string]any
		require.NoError(t, json.Unmarshal(body, &message))
		require.Equal(t, float64(42), message["run_number"])
		require.Equal(t, "start", message["operation"])
		require.Equal(t, "kr.yml", message["config"])
	case <-time.After(time.Second):
		t.Fatal("TOPI notification was not published")
	}
}
