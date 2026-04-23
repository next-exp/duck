package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ===== Shared Test Helpers =====

// createTestServer creates a test server with standard configuration
func createTestServer(t *testing.T, devVersion bool) (*DuckAPIServer, *mocks.Querier, *MockRPCClient) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{}
	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test_config.yml", "test_token", devVersion)
	logger = duck.NewDuckLogger("test", nil, 0)
	return server, mockQuerier, mockRPC
}

// setupConfigMocks sets up mocks for successful configuration read
func setupConfigMocks(mockQuerier *mocks.Querier) {
	mockQuerier.On("ListGDCs", mock.Anything).Return([]database.Gdc{
		{ID: 1, Name: sql.NullString{String: "gdc1", Valid: true}, Ip: sql.NullString{String: "192.168.1.100", Valid: true}, GrpcPort: sql.NullInt32{Int32: 6001, Valid: true}, Enabled: sql.NullBool{Bool: true, Valid: true}},
	}, nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{
		{ID: 1, Name: sql.NullString{String: "ldc1", Valid: true}, Ip: sql.NullString{String: "192.168.1.101", Valid: true}, GrpcPort: sql.NullInt32{Int32: 7001, Valid: true}, Enabled: sql.NullBool{Bool: true, Valid: true}},
	}, nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)
}

// setupActiveRunMocks sets up mocks for an active run (started but not stopped)
func setupActiveRunMocks(mockQuerier *mocks.Querier, runID int32) {
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    runID,
		Start: sql.NullTime{Time: time.Now(), Valid: true},
		Stop:  sql.NullTime{Valid: false},
	}, nil)
}

// setupStoppedRunMocks sets up mocks for a stopped run
func setupStoppedRunMocks(mockQuerier *mocks.Querier, runID int32) {
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    runID,
		Start: sql.NullTime{Time: time.Now(), Valid: true},
		Stop:  sql.NullTime{Time: time.Now(), Valid: true},
	}, nil)
}

// setupUnstartedRunMocks sets up mocks for an unstarted run
func setupUnstartedRunMocks(mockQuerier *mocks.Querier, runID int32) {
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).Return(database.Run{
		ID:    runID,
		Start: sql.NullTime{Valid: false},
		Stop:  sql.NullTime{Valid: false},
	}, nil)
}

// setupFileWriteMock sets up a mock for file write operations
func setupFileWriteMock(t *testing.T, fail bool) (restore func()) {
	oldWriteFile := writeFileFn
	if fail {
		writeFileFn = func(filename string, data []byte, perm fs.FileMode) error {
			return fmt.Errorf("disk full")
		}
	} else {
		writeFileFn = func(filename string, data []byte, perm fs.FileMode) error {
			return nil
		}
	}
	return func() {
		writeFileFn = oldWriteFile
	}
}

// ===== Argument Matcher Helpers =====

// matchRunStopTime verifies UpdateRunStopTimeParams
func matchRunStopTime(runID int32) interface{} {
	return mock.MatchedBy(func(arg database.UpdateRunStopTimeParams) bool {
		return arg.ID == runID && arg.Stop.Valid
	})
}

// matchRunStartTime verifies UpdateRunStartTimeParams
func matchRunStartTime(runID int32) interface{} {
	return mock.MatchedBy(func(arg database.UpdateRunStartTimeParams) bool {
		return arg.ID == runID && arg.Start.Valid
	})
}

// matchGDCEvents verifies InsertGDCEventsParams
func matchGDCEvents(runID, gdcID int32, events int64) interface{} {
	return mock.MatchedBy(func(arg database.InsertGDCEventsParams) bool {
		return arg.Run == runID && arg.GdcID.Int32 == gdcID && arg.Events.Int64 == events
	})
}

// matchGDCBytes verifies InsertGDCBytesParams
func matchGDCBytes(runID, gdcID int32, bytes int64) interface{} {
	return mock.MatchedBy(func(arg database.InsertGDCBytesParams) bool {
		return arg.Run == runID && arg.GdcID.Int32 == gdcID && arg.Bytes.Int64 == bytes
	})
}

// matchGDCErrorCount verifies InsertGDCErrorCountParams
func matchGDCErrorCount(runID, gdcID int32, errors int64) interface{} {
	return mock.MatchedBy(func(arg database.InsertGDCErrorCountParams) bool {
		return arg.Run == runID && arg.GdcID.Int32 == gdcID && arg.Errors.Int64 == errors
	})
}

// matchLDCEvents verifies InsertLDCEventsParams
func matchLDCEvents(runID, ldcID int32, events int64) interface{} {
	return mock.MatchedBy(func(arg database.InsertLDCEventsParams) bool {
		return arg.Run == runID && arg.LdcID.Int32 == ldcID && arg.Events.Int64 == events
	})
}

// matchLDCBytes verifies InsertLDCBytesParams
func matchLDCBytes(runID, ldcID int32, bytes int64) interface{} {
	return mock.MatchedBy(func(arg database.InsertLDCBytesParams) bool {
		return arg.Run == runID && arg.LdcID.Int32 == ldcID && arg.Bytes.Int64 == bytes
	})
}

// matchLDCErrorCount verifies InsertLDCErrorCountParams
func matchLDCErrorCount(runID, ldcID int32, errors int64) interface{} {
	return mock.MatchedBy(func(arg database.InsertLDCErrorCountParams) bool {
		return arg.Run == runID && arg.LdcID.Int32 == ldcID && arg.Errors.Int64 == errors
	})
}

// ===== startProcesses Tests =====

func TestStartProcesses_ConfigReadFailure(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{}
	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test", "token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	// Mock configuration read to fail
	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{}, errors.New("database connection failed"))

	// Execute
	startProcesses(server)

	// Assert: No RPC calls, no database operations beyond config read
	assert.Equal(t, 0, mockRPC.PingCalledCount())
	assert.Equal(t, 0, mockRPC.StartServerCalledCount())
	mockQuerier.AssertNotCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
}

func TestStartProcesses_PingLDCsFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// Mock ping to fail
	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return false, errors.New("ping failed")
	}

	// Execute
	startProcesses(server)

	// Assert: Ping was called but StartServer was not
	assert.Greater(t, mockRPC.PingCalledCount(), 0)
	assert.Equal(t, 0, mockRPC.StartServerCalledCount())
	mockQuerier.AssertNotCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
}

func TestStartProcesses_StartGDCsFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// Mock ping success but GDC start failure
	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return true, nil
	}
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		if port == 6001 { // GDC port
			return errors.New("GDC start failed")
		}
		return nil
	}

	// Execute
	startProcesses(server)

	// Assert: Ping was called, StartServer was called for GDC
	assert.Greater(t, mockRPC.PingCalledCount(), 0)
	assert.Greater(t, mockRPC.StartServerCalledCount(), 0)
	mockQuerier.AssertNotCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
}

func TestStartProcesses_StartLDCsFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// Mock ping and GDC success but LDC start failure
	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return true, nil
	}
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		if port == 7001 { // LDC port
			return errors.New("LDC start failed")
		}
		return nil
	}

	// Execute
	startProcesses(server)

	// Assert: Ping was called, StartServer was called for both GDC and LDC
	assert.Greater(t, mockRPC.PingCalledCount(), 0)
	assert.Greater(t, mockRPC.StartServerCalledCount(), 0)
	mockQuerier.AssertNotCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
}

func TestStartProcesses_GetRunFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	// Mock successful RPC calls
	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return true, nil
	}
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	// Mock getRun to fail
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).
		Return(database.Run{}, errors.New("run not found"))

	// Execute
	startProcesses(server)

	// Assert: All RPC calls were made, getRun was called
	assert.Equal(t, 1, mockRPC.PingCalledCount(), "expected 1 LDC ping call")
	assert.Equal(t, 2, mockRPC.StartServerCalledCount(), "expected 1 GDC + 1 LDC start calls")
	mockQuerier.AssertCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
	mockQuerier.AssertNotCalled(t, "UpdateRunStartTime", mock.Anything, mock.Anything)
}

func TestStartProcesses_UpdateRunStartTimeFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return true, nil
	}
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	// Mock getRun success but update failure
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).
		Return(database.Run{ID: 123}, nil)
	mockQuerier.On("UpdateRunStartTime", mock.Anything, matchRunStartTime(123)).
		Return(errors.New("update failed"))

	// Execute
	startProcesses(server)

	// Assert: UpdateRunStartTime was called
	mockQuerier.AssertCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
	mockQuerier.AssertCalled(t, "UpdateRunStartTime", mock.Anything, matchRunStartTime(123))
}

func TestStartProcesses_FileWriteFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return true, nil
	}
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).
		Return(database.Run{ID: 123}, nil)
	mockQuerier.On("UpdateRunStartTime", mock.Anything, matchRunStartTime(123)).
		Return(nil)

	// Setup file write to fail
	restore := setupFileWriteMock(t, true)
	defer restore()

	// Execute
	startProcesses(server)

	// Assert: Function completes despite file write failure (error logged only)
	mockQuerier.AssertCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
	mockQuerier.AssertCalled(t, "UpdateRunStartTime", mock.Anything, matchRunStartTime(123))
}

func TestStartProcesses_Success(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return true, nil
	}
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).
		Return(database.Run{ID: 123}, nil)
	mockQuerier.On("UpdateRunStartTime", mock.Anything, matchRunStartTime(123)).
		Return(nil)

	// Setup file write to succeed
	restore := setupFileWriteMock(t, false)
	defer restore()

	// Execute
	startProcesses(server)

	// Assert: All operations completed
	assert.Equal(t, 1, mockRPC.PingCalledCount(), "expected 1 LDC ping call")
	assert.Equal(t, 2, mockRPC.StartServerCalledCount(), "expected 1 GDC + 1 LDC start calls")
	mockQuerier.AssertCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
	mockQuerier.AssertCalled(t, "UpdateRunStartTime", mock.Anything, matchRunStartTime(123))
}

func TestStartProcesses_DevModeSkipsFileWrite(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, true) // devVersion = true

	setupConfigMocks(mockQuerier)

	mockRPC.PingFunc = func(ctx context.Context, ip string, port int) (bool, error) {
		return true, nil
	}
	mockRPC.StartServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).
		Return(database.Run{ID: 123}, nil)
	mockQuerier.On("UpdateRunStartTime", mock.Anything, matchRunStartTime(123)).
		Return(nil)

	// Track if writeFileFn was called
	writeFileCalled := false
	oldWriteFile := writeFileFn
	writeFileFn = func(filename string, data []byte, perm fs.FileMode) error {
		writeFileCalled = true
		return nil
	}
	defer func() { writeFileFn = oldWriteFile }()

	// Execute
	startProcesses(server)

	// Assert: File write was NOT called in dev mode
	assert.False(t, writeFileCalled)
	mockQuerier.AssertCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
}

// ===== stopProcesses Tests =====

func TestStopProcesses_ConfigReadFailure(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{}
	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test", "token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	// Mock configuration read to fail
	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{}, errors.New("database connection failed"))

	// Execute
	stopProcesses(server)

	// Assert: No RPC calls, no database operations beyond config read
	assert.Equal(t, 0, mockRPC.StopServerCalledCount())
	mockQuerier.AssertNotCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
}

func TestStopProcesses_GetRunFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	// Mock getRun to fail
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).
		Return(database.Run{}, errors.New("run not found"))

	// Execute
	stopProcesses(server)

	// Assert: Stop was called but no further operations
	assert.Equal(t, 1, mockRPC.StopServerCalledCount(), "expected 1 LDC stop call (stopProcesses only stops LDCs)")
	mockQuerier.AssertCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
	mockQuerier.AssertNotCalled(t, "UpdateRunStopTime", mock.Anything, mock.Anything)
}

func TestStopProcesses_RunNotStarted(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupUnstartedRunMocks(mockQuerier, 123)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	// Execute
	stopProcesses(server)

	// Assert: Stop was called but no update operations (run wasn't started)
	assert.Equal(t, 1, mockRPC.StopServerCalledCount(), "expected 1 LDC stop call (stopProcesses only stops LDCs)")
	mockQuerier.AssertCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
	mockQuerier.AssertNotCalled(t, "UpdateRunStopTime", mock.Anything, mock.Anything)
	mockQuerier.AssertNotCalled(t, "InsertGDCEvents", mock.Anything, mock.Anything)
	mockQuerier.AssertNotCalled(t, "InsertNewRun", mock.Anything)
}

func TestStopProcesses_RunAlreadyStopped(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupStoppedRunMocks(mockQuerier, 123)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	// Execute
	stopProcesses(server)

	// Assert: Stop was called but no update operations (run already stopped)
	assert.Equal(t, 1, mockRPC.StopServerCalledCount(), "expected 1 LDC stop call (stopProcesses only stops LDCs)")
	mockQuerier.AssertCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
	mockQuerier.AssertNotCalled(t, "UpdateRunStopTime", mock.Anything, mock.Anything)
	mockQuerier.AssertNotCalled(t, "InsertGDCEvents", mock.Anything, mock.Anything)
	mockQuerier.AssertNotCalled(t, "InsertNewRun", mock.Anything)
}

func TestStopProcesses_UpdateRunStopTimeFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupActiveRunMocks(mockQuerier, 123)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		return duck.RunStatistics{Events: 100, Bytes: 500, Errors: 0}, nil
	}

	mockQuerier.On("UpdateRunStopTime", mock.Anything, matchRunStopTime(123)).
		Return(errors.New("update failed"))
	mockQuerier.On("InsertGDCEvents", mock.Anything, matchGDCEvents(123, 1, 100)).Return(nil)
	mockQuerier.On("InsertGDCBytes", mock.Anything, matchGDCBytes(123, 1, 500)).Return(nil)
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, matchGDCErrorCount(123, 1, 0)).Return(nil)
	mockQuerier.On("InsertLDCEvents", mock.Anything, matchLDCEvents(123, 1, 100)).Return(nil)
	mockQuerier.On("InsertLDCBytes", mock.Anything, matchLDCBytes(123, 1, 500)).Return(nil)
	mockQuerier.On("InsertLDCErrorCount", mock.Anything, matchLDCErrorCount(123, 1, 0)).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	// Execute
	stopProcesses(server)

	// Assert: UpdateRunStopTime was called but failed, other operations still ran
	mockQuerier.AssertCalled(t, "UpdateRunStopTime", mock.Anything, matchRunStopTime(123))
	mockQuerier.AssertCalled(t, "InsertGDCEvents", mock.Anything, matchGDCEvents(123, 1, 100))
}

func TestStopProcesses_UpdateRunStatisticsFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupActiveRunMocks(mockQuerier, 123)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		return duck.RunStatistics{}, errors.New("stats failed")
	}

	mockQuerier.On("UpdateRunStopTime", mock.Anything, matchRunStopTime(123)).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	// Execute
	stopProcesses(server)

	// Assert: UpdateRunStopTime was called, statistics failed, addRun still ran
	mockQuerier.AssertCalled(t, "UpdateRunStopTime", mock.Anything, matchRunStopTime(123))
	mockQuerier.AssertCalled(t, "InsertNewRun", mock.Anything)
}

func TestStopProcesses_AddRunFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupActiveRunMocks(mockQuerier, 123)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		return duck.RunStatistics{Events: 100, Bytes: 500, Errors: 0}, nil
	}

	mockQuerier.On("UpdateRunStopTime", mock.Anything, matchRunStopTime(123)).Return(nil)
	mockQuerier.On("InsertGDCEvents", mock.Anything, matchGDCEvents(123, 1, 100)).Return(nil)
	mockQuerier.On("InsertGDCBytes", mock.Anything, matchGDCBytes(123, 1, 500)).Return(nil)
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, matchGDCErrorCount(123, 1, 0)).Return(nil)
	mockQuerier.On("InsertLDCEvents", mock.Anything, matchLDCEvents(123, 1, 100)).Return(nil)
	mockQuerier.On("InsertLDCBytes", mock.Anything, matchLDCBytes(123, 1, 500)).Return(nil)
	mockQuerier.On("InsertLDCErrorCount", mock.Anything, matchLDCErrorCount(123, 1, 0)).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(errors.New("insert failed"))

	// Execute
	stopProcesses(server)

	// Assert: All operations were attempted
	assert.Equal(t, 1, mockRPC.StopServerCalledCount(), "expected 1 LDC stop call (stopProcesses only stops LDCs)")
	assert.Equal(t, 2, mockRPC.FetchRunStatisticsCalledCount(), "expected 1 GDC + 1 LDC stats calls")
	mockQuerier.AssertCalled(t, "UpdateRunStopTime", mock.Anything, matchRunStopTime(123))
	mockQuerier.AssertCalled(t, "InsertGDCEvents", mock.Anything, matchGDCEvents(123, 1, 100))
	mockQuerier.AssertCalled(t, "InsertGDCBytes", mock.Anything, matchGDCBytes(123, 1, 500))
	mockQuerier.AssertCalled(t, "InsertGDCErrorCount", mock.Anything, matchGDCErrorCount(123, 1, 0))
	mockQuerier.AssertCalled(t, "InsertLDCEvents", mock.Anything, matchLDCEvents(123, 1, 100))
	mockQuerier.AssertCalled(t, "InsertLDCBytes", mock.Anything, matchLDCBytes(123, 1, 500))
	mockQuerier.AssertCalled(t, "InsertLDCErrorCount", mock.Anything, matchLDCErrorCount(123, 1, 0))
	mockQuerier.AssertCalled(t, "InsertNewRun", mock.Anything)
}

func TestStopProcesses_Success(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupActiveRunMocks(mockQuerier, 123)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		return duck.RunStatistics{Events: 100, Bytes: 500, Errors: 0}, nil
	}

	mockQuerier.On("UpdateRunStopTime", mock.Anything, matchRunStopTime(123)).Return(nil)
	mockQuerier.On("InsertGDCEvents", mock.Anything, matchGDCEvents(123, 1, 100)).Return(nil)
	mockQuerier.On("InsertGDCBytes", mock.Anything, matchGDCBytes(123, 1, 500)).Return(nil)
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, matchGDCErrorCount(123, 1, 0)).Return(nil)
	mockQuerier.On("InsertLDCEvents", mock.Anything, matchLDCEvents(123, 1, 100)).Return(nil)
	mockQuerier.On("InsertLDCBytes", mock.Anything, matchLDCBytes(123, 1, 500)).Return(nil)
	mockQuerier.On("InsertLDCErrorCount", mock.Anything, matchLDCErrorCount(123, 1, 0)).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	// Execute
	stopProcesses(server)

	// Assert: All operations completed
	assert.Equal(t, 1, mockRPC.StopServerCalledCount(), "expected 1 LDC stop call (stopProcesses only stops LDCs)")
	assert.Equal(t, 2, mockRPC.FetchRunStatisticsCalledCount(), "expected 1 GDC + 1 LDC stats calls")
	mockQuerier.AssertCalled(t, "UpdateRunStopTime", mock.Anything, matchRunStopTime(123))
	mockQuerier.AssertCalled(t, "InsertGDCEvents", mock.Anything, matchGDCEvents(123, 1, 100))
	mockQuerier.AssertCalled(t, "InsertGDCBytes", mock.Anything, matchGDCBytes(123, 1, 500))
	mockQuerier.AssertCalled(t, "InsertGDCErrorCount", mock.Anything, matchGDCErrorCount(123, 1, 0))
	mockQuerier.AssertCalled(t, "InsertLDCEvents", mock.Anything, matchLDCEvents(123, 1, 100))
	mockQuerier.AssertCalled(t, "InsertLDCBytes", mock.Anything, matchLDCBytes(123, 1, 500))
	mockQuerier.AssertCalled(t, "InsertLDCErrorCount", mock.Anything, matchLDCErrorCount(123, 1, 0))
	mockQuerier.AssertCalled(t, "InsertNewRun", mock.Anything)
}

// Serialization of concurrent StopRun requests is now enforced by the RunTransition
// state machine in the HTTP handler layer (see TestStopRun_AlreadyStopping in
// apiHandlerControl_test.go). The stopProcesses function itself is only ever called
// from one goroutine at a time.

// ===== forceStopProcesses Tests =====

func TestForceStopProcesses_ConfigReadFailure(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{}
	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test", "token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	// Mock configuration read to fail
	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{}, errors.New("database connection failed"))

	// Execute
	forceStopProcesses(server)

	// Assert: No RPC calls, no database operations beyond config read
	assert.Equal(t, 0, mockRPC.StopServerCalledCount())
	mockQuerier.AssertNotCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
}

func TestForceStopProcesses_GetRunFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	// Mock getRun to fail
	mockQuerier.On("GetLatestRunWithTimestamp", mock.Anything).
		Return(database.Run{}, errors.New("run not found"))

	// Execute
	forceStopProcesses(server)

	// Assert: Stop was called for both GDCs and LDCs but no further operations
	assert.Equal(t, 2, mockRPC.StopServerCalledCount(), "expected 1 GDC + 1 LDC stop calls")
	mockQuerier.AssertCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
	mockQuerier.AssertNotCalled(t, "UpdateRunStopTime", mock.Anything, mock.Anything)
}

func TestForceStopProcesses_UpdateRunStopTimeFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupActiveRunMocks(mockQuerier, 123)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	mockQuerier.On("UpdateRunStopTime", mock.Anything, matchRunStopTime(123)).
		Return(errors.New("update failed"))
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	// Execute
	forceStopProcesses(server)

	// Assert: UpdateRunStopTime was called but failed, addRun still ran
	assert.Equal(t, 2, mockRPC.StopServerCalledCount(), "expected 1 GDC + 1 LDC stop calls")
	mockQuerier.AssertCalled(t, "UpdateRunStopTime", mock.Anything, matchRunStopTime(123))
	mockQuerier.AssertCalled(t, "InsertNewRun", mock.Anything)
}

func TestForceStopProcesses_AddRunFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupActiveRunMocks(mockQuerier, 123)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	mockQuerier.On("UpdateRunStopTime", mock.Anything, matchRunStopTime(123)).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(errors.New("insert failed"))

	// Execute
	forceStopProcesses(server)

	// Assert: All operations were attempted
	assert.Equal(t, 2, mockRPC.StopServerCalledCount(), "expected 1 GDC + 1 LDC stop calls")
	mockQuerier.AssertCalled(t, "UpdateRunStopTime", mock.Anything, matchRunStopTime(123))
	mockQuerier.AssertCalled(t, "InsertNewRun", mock.Anything)
}

func TestForceStopProcesses_SkipsStatisticsUpdate(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupActiveRunMocks(mockQuerier, 123)

	statsCalled := false
	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		statsCalled = true
		return duck.RunStatistics{Events: 100, Bytes: 500, Errors: 0}, nil
	}

	mockQuerier.On("UpdateRunStopTime", mock.Anything, matchRunStopTime(123)).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	// Execute
	forceStopProcesses(server)

	// Assert: Statistics update was skipped (as designed for force stop)
	assert.Equal(t, 2, mockRPC.StopServerCalledCount(), "expected 1 GDC + 1 LDC stop calls")
	assert.False(t, statsCalled, "updateRunStatistics should not be called in forceStopProcesses")
	mockQuerier.AssertNotCalled(t, "InsertGDCEvents", mock.Anything, mock.Anything)
	mockQuerier.AssertNotCalled(t, "InsertLDCEvents", mock.Anything, mock.Anything)
}

func TestForceStopProcesses_Success(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)
	setupActiveRunMocks(mockQuerier, 123)

	mockRPC.StopServerFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	mockQuerier.On("UpdateRunStopTime", mock.Anything, matchRunStopTime(123)).Return(nil)
	mockQuerier.On("InsertNewRun", mock.Anything).Return(nil)

	// Execute
	forceStopProcesses(server)

	// Assert: All operations completed (except statistics)
	assert.Equal(t, 2, mockRPC.StopServerCalledCount(), "expected 1 GDC + 1 LDC stop calls")
	mockQuerier.AssertCalled(t, "GetLatestRunWithTimestamp", mock.Anything)
	mockQuerier.AssertCalled(t, "UpdateRunStopTime", mock.Anything, matchRunStopTime(123))
	mockQuerier.AssertCalled(t, "InsertNewRun", mock.Anything)
}

// ===== updateRunStatistics Tests =====

func TestUpdateRunStatistics_ConfigReadFailure(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{}
	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test", "token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	// Mock configuration read to fail
	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{}, errors.New("database connection failed"))

	// Execute
	err := updateRunStatistics(server, 123)

	// Assert: Error returned
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not read configuration")
}

func TestUpdateRunStatistics_MultipleDevices(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	// Setup multiple GDCs and LDCs
	mockQuerier.On("ListGDCs", mock.Anything).Return([]database.Gdc{
		{ID: 1, Name: sql.NullString{String: "gdc1", Valid: true}, Ip: sql.NullString{String: "192.168.1.100", Valid: true}, GrpcPort: sql.NullInt32{Int32: 6001, Valid: true}, Enabled: sql.NullBool{Bool: true, Valid: true}},
		{ID: 2, Name: sql.NullString{String: "gdc2", Valid: true}, Ip: sql.NullString{String: "192.168.1.101", Valid: true}, GrpcPort: sql.NullInt32{Int32: 6002, Valid: true}, Enabled: sql.NullBool{Bool: true, Valid: true}},
	}, nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{
		{ID: 1, Name: sql.NullString{String: "ldc1", Valid: true}, Ip: sql.NullString{String: "192.168.1.101", Valid: true}, GrpcPort: sql.NullInt32{Int32: 7001, Valid: true}, Enabled: sql.NullBool{Bool: true, Valid: true}},
		{ID: 2, Name: sql.NullString{String: "ldc2", Valid: true}, Ip: sql.NullString{String: "192.168.1.102", Valid: true}, GrpcPort: sql.NullInt32{Int32: 7002, Valid: true}, Enabled: sql.NullBool{Bool: true, Valid: true}},
	}, nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)

	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		return duck.RunStatistics{Events: 100, Bytes: 500, Errors: 0}, nil
	}

	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertLDCErrorCount", mock.Anything, mock.Anything).Return(nil)

	// Execute
	err := updateRunStatistics(server, 123)

	// Assert: Statistics updated for all devices (2 GDCs + 2 LDCs = 4 FetchRunStatistics calls)
	assert.NoError(t, err)
	assert.Equal(t, 4, mockRPC.FetchRunStatisticsCalledCount())
}

func TestUpdateRunStatistics_EmptyDeviceLists(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	// Setup empty device lists
	mockQuerier.On("ListGDCs", mock.Anything).Return([]database.Gdc{}, nil)
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{}, nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetLatestRun", mock.Anything).Return(int32(1), nil)
	mockQuerier.On("GetDuckParams", mock.Anything).Return(database.Duckparam{}, nil)
	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, nil)
	mockQuerier.On("GetTestDeviceParams", mock.Anything).Return([]database.Testdeviceparam{}, nil)

	// Execute
	err := updateRunStatistics(server, 123)

	// Assert: No error, no RPC calls
	assert.NoError(t, err)
	assert.Equal(t, 0, mockRPC.FetchRunStatisticsCalledCount())
}

func TestUpdateRunStatistics_PartialFailure(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier) // 1 GDC at 192.168.1.100, 1 LDC at 192.168.1.101

	// GDC succeeds, LDC fails — deterministic via IP to avoid call-order races
	mockRPC.FetchRunStatisticsFunc = func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
		if ip == "192.168.1.100" {
			return duck.RunStatistics{Events: 100, Bytes: 500, Errors: 0}, nil
		}
		return duck.RunStatistics{}, errors.New("stats failed")
	}

	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCBytes", mock.Anything, mock.Anything).Return(nil)
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, mock.Anything).Return(nil)

	// Execute
	err := updateRunStatistics(server, 123)

	// Assert: Returns nil even with partial failure (errors are logged)
	assert.NoError(t, err)
}

// ===== getProcessStates Tests =====

func TestGetProcessStates_ConfigReadFailure(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{}
	server := NewDuckAPIServerWithMocks(mockQuerier, mockRPC, "test", "token", false)
	logger = duck.NewDuckLogger("test", nil, 0)

	// Mock configuration read to fail
	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{}, errors.New("database connection failed"))

	// Execute
	getProcessStates(server)

	// Assert: No RPC calls
	assert.Equal(t, 0, mockRPC.GetStateCalledCount())
}

func TestGetProcessStates_Success(t *testing.T) {
	server, mockQuerier, mockRPC := createTestServer(t, false)

	setupConfigMocks(mockQuerier)

	mockRPC.GetStateFunc = func(ctx context.Context, ip string, port int) error {
		return nil
	}

	// Execute
	getProcessStates(server)

	// Assert: GetState called for both LDCs and GDCs
	assert.Equal(t, 2, mockRPC.GetStateCalledCount(), "expected 1 GDC + 1 LDC getState calls")
}
