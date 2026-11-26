package main

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRestartCommands(t *testing.T) {
	for _, role := range []string{"ldc", "gdc"} {
		for profile, want := range map[string]string{"next100": "docker compose --project-directory /home/duck/duck-daq -p duck-daq restart " + role, "duck-prod": "docker container restart duck-prod-" + role + "-1", "": "/home/dateuser/duck/restart_" + role + ".sh"} {
			cmd, err := restartCommand(profile, role)
			require.NoError(t, err)
			require.Equal(t, want, cmd)
		}
	}
	_, err := restartCommand("duck-prod", "api")
	require.Error(t, err)
}
func TestRestartReturnsBeforeWorkerAndExcludesStart(t *testing.T) {
	q := mocks.NewQuerier(t)
	gate := make(chan struct{})
	q.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{}, nil)
	q.On("ListGDCs", mock.Anything).Run(func(mock.Arguments) { <-gate }).Return(nil, errors.New("inventory unavailable"))
	s := NewDuckAPIServerWithMocks(q, &MockRPCClient{}, "", "", false)
	result, err := s.beginRestart(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, result.Msg.OperationId)
	ok, _ := s.runTransition.tryBeginStart()
	require.False(t, ok)
	_, err = s.beginRestart(context.Background())
	require.Error(t, err)
	close(gate)
	require.True(t, s.runTransition.WaitForIdle(time.Second))
	status, err := s.GetRunTransitionStatus(context.Background(), connect.NewRequest(&pb.GetRunTransitionStatusRequest{}))
	require.NoError(t, err)
	require.Contains(t, status.Msg.RestartError, "inventory unavailable")
}
