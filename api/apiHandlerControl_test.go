package main

import (
	"context"
	"database/sql"
	"testing"

	duck "github.com/jmbenlloch/next_duck/pkg"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"connectrpc.com/connect"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// These tests verify that the ConnectRPC handlers:
// 1. Return proper response structures with Success=true and correct messages
// 2. Delegate to the correct control functions (verified via RPC calls)
//
// Since control functions are package-level and can't be easily mocked,
// we verify delegation by checking that they make the expected RPC calls.

// Helper function to create test data
func getTestGDCs() []database.Gdc {
	return []database.Gdc{
		{
			ID:       1,
			Name:     sql.NullString{String: "gdc1", Valid: true},
			Ip:       sql.NullString{String: "192.168.1.100", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 6001, Valid: true},
			Enabled:  sql.NullBool{Bool: true, Valid: true},
		},
	}
}

func getTestLDCs() []database.Ldc {
	return []database.Ldc{
		{
			ID:       1,
			Name:     sql.NullString{String: "ldc1", Valid: true},
			Ip:       sql.NullString{String: "192.168.1.101", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 7001, Valid: true},
			Enabled:  sql.NullBool{Bool: true, Valid: true},
		},
	}
}

// ===== StartRun Handler Tests =====

func TestStartRun(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{
		PingFunc: func(ctx context.Context, ip string, port int) (bool, error) {
			return true, nil
		},
		StartServerFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test_config.yml", "test_token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	// Setup database mocks with actual test data
	mockQuerier.On("ListGDCs", mock.Anything).Return(getTestGDCs(), nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return(getTestLDCs(), nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{ID: 1, Start: sql.NullTime{Valid: true}}, nil)
	mockQuerier.On("UpdateRunStartTime", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)

	req := connect.NewRequest(&pb.StartRunRequest{})

	// Execute
	resp, err := server.StartRun(context.Background(), req)

	// Assert - handler returns valid response
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "starting processes", resp.Msg.Message)

	// Assert - StartRun should have called Ping (via sendPingToAllLDCs)
	// and StartServer (via startGDCs/startLDCs)
	assert.Greater(t, mockRPC.PingCalledCount(), 0, "StartRun should ping LDCs")
	assert.Greater(t, mockRPC.StartServerCalledCount(), 0, "StartRun should start servers")
}

// ===== StopRun Handler Tests =====

func TestStopRun(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{
		StopServerFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test_config.yml", "test_token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	// Setup mocks
	mockQuerier.On("ListGDCs", mock.Anything).Return(getTestGDCs(), nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return(getTestLDCs(), nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{ID: 1, Start: sql.NullTime{Valid: true}, Stop: sql.NullTime{Valid: false}}, nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)
	mockQuerier.On("UpdateRunStopTime", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCErrorCount", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	req := connect.NewRequest(&pb.StopRunRequest{})

	// Execute
	resp, err := server.StopRun(context.Background(), req)

	// Assert - handler returns valid response
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "stopping processes", resp.Msg.Message)

	// Assert - StopRun should have called StopServer (via stopLDCs)
	assert.Greater(t, mockRPC.StopServerCalledCount(), 0, "StopRun should stop LDCs")
}

// ===== ForceStopRun Handler Tests =====

func TestForceStopRun(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{
		StopServerFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test_config.yml", "test_token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	// Setup mocks
	mockQuerier.On("ListGDCs", mock.Anything).Return(getTestGDCs(), nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return(getTestLDCs(), nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{ID: 1, Start: sql.NullTime{Valid: true}}, nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)
	mockQuerier.On("UpdateRunStopTime", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	req := connect.NewRequest(&pb.ForceStopRunRequest{})

	// Execute
	resp, err := server.ForceStopRun(context.Background(), req)

	// Assert - handler returns valid response
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "force stopping processes", resp.Msg.Message)

	// Assert - ForceStopRun should have called StopServer (via stopGDCs and stopLDCs)
	assert.Greater(t, mockRPC.StopServerCalledCount(), 0, "ForceStopRun should stop servers")
}

// ===== GetProcessStates Handler Tests =====

func TestGetProcessStates(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{
		GetStateFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test_config.yml", "test_token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	// Setup mocks
	mockQuerier.On("ListGDCs", mock.Anything).Return(getTestGDCs(), nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return(getTestLDCs(), nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)

	req := connect.NewRequest(&pb.GetProcessStatesRequest{})

	// Execute
	resp, err := server.GetProcessStates(context.Background(), req)

	// Assert - handler returns valid response
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "get processes states", resp.Msg.Message)

	// Assert - GetProcessStates should have called GetState for both LDCs and GDCs
	assert.Equal(t, 2, mockRPC.GetStateCalledCount(), "GetProcessStates should get states from both LDCs and GDCs")
}

// ===== RestartServices Handler Tests =====

func TestRestartServices(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{}

	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test_config.yml", "test_token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	req := connect.NewRequest(&pb.RestartServicesRequest{})

	// Execute
	resp, err := server.RestartServices(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "restarting services", resp.Msg.Message)
}

func TestRestartServices_WithConfig(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{}

	// Create server with specific config file
	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "specific_config.yml", "test_token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	req := connect.NewRequest(&pb.RestartServicesRequest{})

	// Execute
	resp, err := server.RestartServices(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "restarting services", resp.Msg.Message)
}
