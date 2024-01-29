package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ===== Helper Functions =====

// setupMixedDeviceMocks creates mocks with both enabled and disabled devices
func setupMixedDeviceMocks(mockQuerier *mocks.Querier) {
	mockQuerier.On("ListGDCs", mock.Anything).Return([]database.Gdc{
		{
			ID:      1,
			Name:    sql.NullString{String: "gdc1-enabled", Valid: true},
			Ip:      sql.NullString{String: "192.168.1.100", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 6001, Valid: true},
			Enabled: sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:      2,
			Name:    sql.NullString{String: "gdc2-disabled", Valid: true},
			Ip:      sql.NullString{String: "192.168.1.101", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 6002, Valid: true},
			Enabled: sql.NullBool{Bool: false, Valid: true},
		},
	}, nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{
		{
			ID:       1,
			Name:     sql.NullString{String: "ldc1-enabled", Valid: true},
			Ip:       sql.NullString{String: "192.168.1.110", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 7001, Valid: true},
			Enabled:  sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:       2,
			Name:     sql.NullString{String: "ldc2-disabled", Valid: true},
			Ip:       sql.NullString{String: "192.168.1.111", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 7002, Valid: true},
			Enabled:  sql.NullBool{Bool: false, Valid: true},
		},
	}, nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)
}

// ===== A. Disabled Element Tests =====

func TestStartProcesses_DisabledGDC_Skipped(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	// Setup with one enabled, one disabled GDC
	mockQuerier.On("ListGDCs", mock.Anything).Return([]database.Gdc{
		{
			ID:       1,
			Name:     sql.NullString{String: "gdc1-enabled", Valid: true},
			Ip:       sql.NullString{String: "192.168.1.100", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 6001, Valid: true},
			Enabled:  sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:       2,
			Name:     sql.NullString{String: "gdc2-disabled", Valid: true},
			Ip:       sql.NullString{String: "192.168.1.101", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 6002, Valid: true},
			Enabled:  sql.NullBool{Bool: false, Valid: true},
		},
	}, nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{
		{ID: 1, Name: sql.NullString{String: "ldc1", Valid: true}, Ip: sql.NullString{String: "192.168.1.110", Valid: true}, GrpcPort: sql.NullInt32{Int32: 7001, Valid: true}, Enabled: sql.NullBool{Bool: true, Valid: true}},
	}, nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)

	// Track which IPs were called
	var calledIPs []string
	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		calledIPs = append(calledIPs, ip)
		return true, nil
	}
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		calledIPs = append(calledIPs, ip)
		return nil
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{ID: 123}, nil)
	mockQuerier.On("UpdateRunStartTime", mock.Anything, mock.Anything).Return(nil)

	startProcesses(server)

	// Assert: Only enabled GDC IP was called
	assert.Contains(t, calledIPs, "192.168.1.100", "enabled GDC should be called")
	assert.NotContains(t, calledIPs, "192.168.1.101", "disabled GDC should not be called")
}

func TestStartProcesses_DisabledLDC_Skipped(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	// Setup with one enabled, one disabled LDC
	mockQuerier.On("ListGDCs", mock.Anything).Return([]database.Gdc{
		{ID: 1, Name: sql.NullString{String: "gdc1", Valid: true}, Ip: sql.NullString{String: "192.168.1.100", Valid: true}, GrpcPort: sql.NullInt32{Int32: 6001, Valid: true}, Enabled: sql.NullBool{Bool: true, Valid: true}},
	}, nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{
		{
			ID:       1,
			Name:     sql.NullString{String: "ldc1-enabled", Valid: true},
			Ip:       sql.NullString{String: "192.168.1.110", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 7001, Valid: true},
			Enabled:  sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:       2,
			Name:     sql.NullString{String: "ldc2-disabled", Valid: true},
			Ip:       sql.NullString{String: "192.168.1.111", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 7002, Valid: true},
			Enabled:  sql.NullBool{Bool: false, Valid: true},
		},
	}, nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)

	// Track which IPs were pinged
	var pingedIPs []string
	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		pingedIPs = append(pingedIPs, ip)
		return true, nil
	}
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{ID: 123}, nil)
	mockQuerier.On("UpdateRunStartTime", mock.Anything, mock.Anything).Return(nil)

	startProcesses(server)

	// Assert: Only enabled LDC IP was pinged
	assert.Contains(t, pingedIPs, "192.168.1.110", "enabled LDC should be pinged")
	assert.NotContains(t, pingedIPs, "192.168.1.111", "disabled LDC should not be pinged")
}

func TestStopProcesses_DisabledLDC_Skipped(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	// Setup with one enabled, one disabled LDC
	mockQuerier.On("ListGDCs", mock.Anything).Return([]database.Gdc{
		{ID: 1, Name: sql.NullString{String: "gdc1", Valid: true}, Ip: sql.NullString{String: "192.168.1.100", Valid: true}, GrpcPort: sql.NullInt32{Int32: 6001, Valid: true}, Enabled: sql.NullBool{Bool: true, Valid: true}},
	}, nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{
		{
			ID:       1,
			Name:     sql.NullString{String: "ldc1-enabled", Valid: true},
			Ip:       sql.NullString{String: "192.168.1.110", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 7001, Valid: true},
			Enabled:  sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:       2,
			Name:     sql.NullString{String: "ldc2-disabled", Valid: true},
			Ip:       sql.NullString{String: "192.168.1.111", Valid: true},
			GrpcPort: sql.NullInt32{Int32: 7002, Valid: true},
			Enabled:  sql.NullBool{Bool: false, Valid: true},
		},
	}, nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)

	// Track which IPs were stopped
	var stoppedIPs []string
	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		stoppedIPs = append(stoppedIPs, ip)
		return nil
	}
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		return duck.RunStatistics{Events: 100, Bytes: 500, Errors: 0}, nil
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    123,
		Start: sql.NullTime{Time: time.Now(), Valid: true},
		Stop:  sql.NullTime{Valid: false},
	}, nil)
	mockQuerier.On("UpdateRunStopTime", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCErrorCount", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	stopProcesses(server)

	// Assert: Only enabled LDC IP was stopped
	assert.Contains(t, stoppedIPs, "192.168.1.110", "enabled LDC should be stopped")
	assert.NotContains(t, stoppedIPs, "192.168.1.111", "disabled LDC should not be stopped")
}

func TestForceStopProcesses_DisabledDevices_Skipped(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupMixedDeviceMocks(mockQuerier)

	// Track which IPs were stopped
	var stoppedIPs []string
	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		stoppedIPs = append(stoppedIPs, ip)
		return nil
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    123,
		Start: sql.NullTime{Time: time.Now(), Valid: true},
		Stop:  sql.NullTime{Valid: false},
	}, nil)
	mockQuerier.On("UpdateRunStopTime", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	forceStopProcesses(server)

	// Assert: Only enabled devices were stopped
	assert.Contains(t, stoppedIPs, "192.168.1.100", "enabled GDC should be stopped")
	assert.NotContains(t, stoppedIPs, "192.168.1.101", "disabled GDC should not be stopped")
	assert.Contains(t, stoppedIPs, "192.168.1.110", "enabled LDC should be stopped")
	assert.NotContains(t, stoppedIPs, "192.168.1.111", "disabled LDC should not be stopped")
}

func TestUpdateRunStatistics_DisabledDevices_Skipped(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupMixedDeviceMocks(mockQuerier)

	// Track which IPs had stats fetched
	var statsFetchedIPs []string
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		statsFetchedIPs = append(statsFetchedIPs, ip)
		return duck.RunStatistics{Events: 100, Bytes: 500, Errors: 0}, nil
	}

	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCErrorCount", mock.Anything, mock.Anything).Return(nil)

	err := updateRunStatistics(server, 123)

	assert.NoError(t, err)
	assert.Contains(t, statsFetchedIPs, "192.168.1.100", "enabled GDC should have stats fetched")
	assert.NotContains(t, statsFetchedIPs, "192.168.1.101", "disabled GDC should not have stats fetched")
	assert.Contains(t, statsFetchedIPs, "192.168.1.110", "enabled LDC should have stats fetched")
	assert.NotContains(t, statsFetchedIPs, "192.168.1.111", "disabled LDC should not have stats fetched")
}

func TestGetProcessStates_DisabledDevices_Skipped(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupMixedDeviceMocks(mockQuerier)

	// Track which IPs had state fetched
	var stateFetchedIPs []string
	mockRPC.GetStateFunc = func(ctx context.Context, ip string, port int) error {
		stateFetchedIPs = append(stateFetchedIPs, ip)
		return nil
	}

	getProcessStates(server)

	// Assert: Only enabled devices had state fetched
	assert.Contains(t, stateFetchedIPs, "192.168.1.100", "enabled GDC should have state fetched")
	assert.NotContains(t, stateFetchedIPs, "192.168.1.101", "disabled GDC should not have state fetched")
	assert.Contains(t, stateFetchedIPs, "192.168.1.110", "enabled LDC should have state fetched")
	assert.NotContains(t, stateFetchedIPs, "192.168.1.111", "disabled LDC should not have state fetched")
}

func TestStartProcesses_MixedEnabledDisabled(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupMixedDeviceMocks(mockQuerier)

	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return true, nil
	}
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{ID: 123}, nil)
	mockQuerier.On("UpdateRunStartTime", mock.Anything, mock.Anything).Return(nil)

	startProcesses(server)

	// Assert: Only enabled devices were processed
	assert.Equal(t, 1, mockRPC.PingCalledCount(), "only enabled LDC should be pinged")
	assert.Equal(t, 2, mockRPC.StartServerCalledCount(), "enabled GDC + enabled LDC should be started")
}

// ===== B. Timeout Tests =====

func TestStartProcesses_PingTimeout(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// Ping blocks until context is cancelled (times out)
	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		<-ctx.Done()
		return false, ctx.Err()
	}

	// Don't mock GetLatestRunWithTimestamp - it won't be called due to ping failure

	startProcesses(server)

	// Assert: Ping was called but StartServer was not (ping timed out)
	assert.Greater(t, mockRPC.PingCalledCount(), 0, "ping should be called")
	assert.Equal(t, 0, mockRPC.StartServerCalledCount(), "StartServer should not be called after ping timeout")
	// Verify GetLatestRunWithTimestamp was not called due to ping failure
	mockQuerier.AssertNotCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
}

func TestStartProcesses_StartServerTimeout(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return true, nil
	}
	// StartServer blocks until context is cancelled
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		<-ctx.Done()
		return ctx.Err()
	}

	// Don't mock GetLatestRunWithTimestamp - it won't be called due to start failure

	startProcesses(server)

	// Assert: RPC calls were made but run update was not (start timed out)
	assert.Greater(t, mockRPC.PingCalledCount(), 0, "ping should be called")
	assert.Greater(t, mockRPC.StartServerCalledCount(), 0, "StartServer should be called")
	// Verify GetLatestRunWithTimestamp and UpdateRunStartTime were not called due to start timeout
	mockQuerier.AssertNotCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
	mockQuerier.AssertNotCalled(t, "UpdateRunStartTime", mock.Anything, mock.Anything)
}

func TestStopProcesses_StopServerTimeout(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupActiveRunMocks(mockQuerier, 123)

	// StopServer blocks until context is cancelled
	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		<-ctx.Done()
		return ctx.Err()
	}
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		return duck.RunStatistics{Events: 100, Bytes: 500, Errors: 0}, nil
	}

	// Even though StopServer times out, other operations may be attempted
	mockQuerier.On("UpdateRunStopTime", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCErrorCount", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	stopProcesses(server)

	// Assert: StopServer was called despite timeout
	assert.Greater(t, mockRPC.StopServerCalledCount(), 0, "StopServer should be called")
}

func TestUpdateRunStatistics_FetchTimeout(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// FetchRunStatistics blocks until context is cancelled
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		<-ctx.Done()
		return duck.RunStatistics{}, ctx.Err()
	}

	// Don't mock database inserts - they won't be called due to RPC timeout

	err := updateRunStatistics(server, 123)

	// Assert: Function returns nil (errors are logged)
	assert.NoError(t, err)
	assert.Greater(t, mockRPC.FetchRunStatisticsCalledCount(), 0, "FetchRunStatistics should be called")
	// Verify database inserts were not called due to RPC timeout
	mockQuerier.AssertNotCalled(t, "InsertGDCEvents", mock.Anything, mock.Anything)
	mockQuerier.AssertNotCalled(t, "InsertLDCEvents", mock.Anything, mock.Anything)
}

func TestGetProcessStates_GetStateTimeout(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// GetState blocks until context is cancelled
	mockRPC.GetStateFunc = func(ctx context.Context, ip string, port int) error {
		<-ctx.Done()
		return ctx.Err()
	}

	getProcessStates(server)

	// Assert: GetState was called
	assert.Greater(t, mockRPC.GetStateCalledCount(), 0, "GetState should be called")
}

// ===== C. Error Logging Tests =====

func TestStartProcesses_LogsConfigError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{}
	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test", "token", false)

	// Create logger with capture
	testLogger, logBuf := duck.CreateTestLoggerWithCapture(t)
	logger = testLogger

	// Mock configuration read to fail
	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{}, errors.New("database connection failed"))

	startProcesses(server)

	// Assert: Error was logged
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "database connection failed", "error should be logged")
}

func TestStartProcesses_LogsPingError(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// Create logger with capture
	testLogger, logBuf := duck.CreateTestLoggerWithCapture(t)
	logger = testLogger

	// Mock ping to fail
	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return false, errors.New("ping failed")
	}

	startProcesses(server)

	// Assert: Ping error was logged
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "ping failed", "ping error should be logged")
}

func TestStopProcesses_LogsStopError(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// Create logger with capture
	testLogger, logBuf := duck.CreateTestLoggerWithCapture(t)
	logger = testLogger

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return errors.New("stop failed")
	}
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		return duck.RunStatistics{Events: 100, Bytes: 500, Errors: 0}, nil
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    123,
		Start: sql.NullTime{Time: time.Now(), Valid: true},
		Stop:  sql.NullTime{Valid: false},
	}, nil)
	mockQuerier.On("UpdateRunStopTime", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCErrorCount", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	stopProcesses(server)

	// Assert: Stop error was logged but function continued
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "stop failed", "stop error should be logged")
	// Verify that other operations were still attempted
	assert.Greater(t, mockRPC.FetchRunStatisticsCalledCount(), 0, "stats should still be fetched")
}

func TestStopProcesses_LogsStatsError(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// Create logger with capture
	testLogger, logBuf := duck.CreateTestLoggerWithCapture(t)
	logger = testLogger

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		return duck.RunStatistics{}, errors.New("stats fetch failed")
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    123,
		Start: sql.NullTime{Time: time.Now(), Valid: true},
		Stop:  sql.NullTime{Valid: false},
	}, nil)
	mockQuerier.On("UpdateRunStopTime", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	stopProcesses(server)

	// Assert: Stats error was logged but function continued
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "stats fetch failed", "stats error should be logged")
	mockQuerier.AssertCalled(t, "InsertNewRun", mock.Anything)
}

func TestForceStopProcesses_LogsError(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// Create logger with capture
	testLogger, logBuf := duck.CreateTestLoggerWithCapture(t)
	logger = testLogger

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return errors.New("force stop failed")
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    123,
		Start: sql.NullTime{Time: time.Now(), Valid: true},
		Stop:  sql.NullTime{Valid: false},
	}, nil)
	mockQuerier.On("UpdateRunStopTime", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	forceStopProcesses(server)

	// Assert: Error was logged
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "force stop failed", "force stop error should be logged")
}

// ===== D. Context Cancellation Tests =====
// Note: These tests verify RPC layer behavior with context timeout/cancellation.
// The production code creates fresh contexts with deadlines, so we expect
// DeadlineExceeded errors after the 10-second timeout.

func TestPingDevices_RespectsContextCancellation(t *testing.T) {
	mockRPC := &MockRPCClient{
		PingFunc: func(ctx context.Context, ip string, port int) (bool, error) {
			// Block until context is cancelled
			<-ctx.Done()
			return false, ctx.Err()
		},
	}

	ldc := duck.LDCConfiguration{
		ID:       1,
		Name:     "ldc1",
		IP:       "192.168.1.101",
		GRPCPort: 7001,
		Enabled:  true,
	}
	errorChan := make(chan error, 1)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Run ping - production code creates its own context with deadline
	go func() {
		pingDevices(ldc, ldc.GRPCPort, errorChan, mockRPC)
	}()

	// Assert: Ping should return context deadline exceeded error
	err := <-errorChan
	assert.Error(t, err)
	// Error is wrapped, so use errors.Is to check for DeadlineExceeded
	assert.True(t, errors.Is(err, context.DeadlineExceeded), "error should be or contain DeadlineExceeded")
}

func TestStartServer_RespectsContextCancellation(t *testing.T) {
	mockRPC := &MockRPCClient{
		StartServerFunc: func(ctx context.Context, ip string, port int) error {
			// Block until context is cancelled
			<-ctx.Done()
			return ctx.Err()
		},
	}

	errorChan := make(chan error, 1)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Run startServer - production code creates its own context with deadline
	go func() {
		startServer("192.168.1.100", "gdc1", 6001, errorChan, mockRPC)
	}()

	// Assert: StartServer should return context deadline exceeded error
	err := <-errorChan
	assert.Error(t, err)
	// Error is wrapped, so use errors.Is to check for DeadlineExceeded
	assert.True(t, errors.Is(err, context.DeadlineExceeded), "error should be or contain DeadlineExceeded")
}

func TestStopServer_RespectsContextCancellation(t *testing.T) {
	mockRPC := &MockRPCClient{
		StopServerFunc: func(ctx context.Context, ip string, port int) error {
			// Block until context is cancelled
			<-ctx.Done()
			return ctx.Err()
		},
	}

	errorChan := make(chan error, 1)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Run stopServer - production code creates its own context with deadline
	go func() {
		stopServer("192.168.1.100", "gdc1", 6001, errorChan, mockRPC)
	}()

	// Assert: StopServer should return context deadline exceeded error
	err := <-errorChan
	assert.Error(t, err)
	// Error is wrapped, so use errors.Is to check for DeadlineExceeded
	assert.True(t, errors.Is(err, context.DeadlineExceeded), "error should be or contain DeadlineExceeded")
}

func TestUpdateGDCStatistics_RespectsContextCancellation(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{
		FetchRunStatisticsFunc: func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
			// Block until context is cancelled
			<-ctx.Done()
			return duck.RunStatistics{}, ctx.Err()
		},
	}

	gdc := duck.GDCConfiguration{
		ID:       1,
		Name:     "gdc1",
		IP:       "192.168.1.100",
		GRPCPort: 6001,
	}
	run := 123

	logger = duck.NewDuckLogger("test", nil, 0)

	// Run update - production code creates its own context with deadline
	updateGDCStatistics(gdc, mockQuerier, run, mockRPC)

	// Assert: FetchRunStatistics was called and returned context error
	assert.Greater(t, mockRPC.FetchRunStatisticsCalledCount(), 0, "FetchRunStatistics should be called")
	// No database inserts should occur due to RPC error
	mockQuerier.AssertNotCalled(t, "InsertGDCEvents", mock.Anything, mock.Anything)
	mockQuerier.AssertNotCalled(t, "InsertGDCBytes", mock.Anything, mock.Anything)
	mockQuerier.AssertNotCalled(t, "InsertGDCErrorCount", mock.Anything, mock.Anything)
}
