package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRPCClient implements RPCClient for testing
type MockRPCClient struct {
	PingFunc               func(ctx context.Context, ip string, port int) (bool, error)
	StartServerFunc        func(ctx context.Context, ip string, port int) error
	StopServerFunc         func(ctx context.Context, ip string, port int) error
	GetStateFunc           func(ctx context.Context, ip string, port int) error
	FetchRunStatisticsFunc func(ctx context.Context, ip string, port int) (duck.RunStatistics, error)

	// Track calls with atomic counts (thread-safe for concurrent tests)
	pingCalledCount               int32
	startServerCalledCount        int32
	stopServerCalledCount         int32
	getStateCalledCount           int32
	fetchRunStatisticsCalledCount int32

	// Track last call parameters (protected by mutex for concurrent access)
	mu       sync.Mutex
	LastIP   string
	LastPort int

	// For timeout testing - simulated delay before returning
	SimulatedDelay time.Duration
}

// Thread-safe getter methods for call counters
func (m *MockRPCClient) PingCalledCount() int               { return int(atomic.LoadInt32(&m.pingCalledCount)) }
func (m *MockRPCClient) StartServerCalledCount() int        { return int(atomic.LoadInt32(&m.startServerCalledCount)) }
func (m *MockRPCClient) StopServerCalledCount() int         { return int(atomic.LoadInt32(&m.stopServerCalledCount)) }
func (m *MockRPCClient) GetStateCalledCount() int           { return int(atomic.LoadInt32(&m.getStateCalledCount)) }
func (m *MockRPCClient) FetchRunStatisticsCalledCount() int { return int(atomic.LoadInt32(&m.fetchRunStatisticsCalledCount)) }

func (m *MockRPCClient) Ping(ctx context.Context, ip string, port int) (bool, error) {
	atomic.AddInt32(&m.pingCalledCount, 1)
	m.mu.Lock()
	m.LastIP = ip
	m.LastPort = port
	m.mu.Unlock()

	// Apply simulated delay if set, respecting context cancellation
	if m.SimulatedDelay > 0 {
		select {
		case <-time.After(m.SimulatedDelay):
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}

	if m.PingFunc != nil {
		return m.PingFunc(ctx, ip, port)
	}
	return true, nil
}

func (m *MockRPCClient) StartServer(ctx context.Context, ip string, port int) error {
	atomic.AddInt32(&m.startServerCalledCount, 1)
	m.mu.Lock()
	m.LastIP = ip
	m.LastPort = port
	m.mu.Unlock()

	// Apply simulated delay if set, respecting context cancellation
	if m.SimulatedDelay > 0 {
		select {
		case <-time.After(m.SimulatedDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if m.StartServerFunc != nil {
		return m.StartServerFunc(ctx, ip, port)
	}
	return nil
}

func (m *MockRPCClient) StopServer(ctx context.Context, ip string, port int) error {
	atomic.AddInt32(&m.stopServerCalledCount, 1)
	m.mu.Lock()
	m.LastIP = ip
	m.LastPort = port
	m.mu.Unlock()

	// Apply simulated delay if set, respecting context cancellation
	if m.SimulatedDelay > 0 {
		select {
		case <-time.After(m.SimulatedDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if m.StopServerFunc != nil {
		return m.StopServerFunc(ctx, ip, port)
	}
	return nil
}

func (m *MockRPCClient) GetState(ctx context.Context, ip string, port int) error {
	atomic.AddInt32(&m.getStateCalledCount, 1)
	m.mu.Lock()
	m.LastIP = ip
	m.LastPort = port
	m.mu.Unlock()

	// Apply simulated delay if set, respecting context cancellation
	if m.SimulatedDelay > 0 {
		select {
		case <-time.After(m.SimulatedDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if m.GetStateFunc != nil {
		return m.GetStateFunc(ctx, ip, port)
	}
	return nil
}

func (m *MockRPCClient) FetchRunStatistics(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
	atomic.AddInt32(&m.fetchRunStatisticsCalledCount, 1)
	m.mu.Lock()
	m.LastIP = ip
	m.LastPort = port
	m.mu.Unlock()

	// Apply simulated delay if set, respecting context cancellation
	if m.SimulatedDelay > 0 {
		select {
		case <-time.After(m.SimulatedDelay):
		case <-ctx.Done():
			return duck.RunStatistics{}, ctx.Err()
		}
	}

	if m.FetchRunStatisticsFunc != nil {
		return m.FetchRunStatisticsFunc(ctx, ip, port)
	}
	return duck.RunStatistics{}, nil
}

// ===== updateGDCStatistics Tests =====

func TestUpdateGDCStatistics_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{
		FetchRunStatisticsFunc: func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
			assert.Equal(t, "192.168.1.100", ip)
			assert.Equal(t, 6001, port)
			return duck.RunStatistics{
				Events: 1000,
				Bytes:  5000,
				Errors: 5,
			}, nil
		},
	}

	gdc := duck.GDCConfiguration{
		ID:       1,
		Name:     "gdc1",
		IP:       "192.168.1.100",
		GRPCPort: 6001,
	}
	run := 123

	// Mock database calls
	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.MatchedBy(func(arg database.InsertGDCEventsParams) bool {
		return arg.Run == int32(run) && arg.GdcID.Int32 == int32(gdc.ID) && arg.Events.Int64 == 1000
	})).Return(nil)

	mockQuerier.On("InsertGDCBytes", mock.Anything, mock.MatchedBy(func(arg database.InsertGDCBytesParams) bool {
		return arg.Run == int32(run) && arg.GdcID.Int32 == int32(gdc.ID) && arg.Bytes.Int64 == 5000
	})).Return(nil)

	mockQuerier.On("InsertGDCErrorCount", mock.Anything, mock.MatchedBy(func(arg database.InsertGDCErrorCountParams) bool {
		return arg.Run == int32(run) && arg.GdcID.Int32 == int32(gdc.ID) && arg.Errors.Int64 == 5
	})).Return(nil)

	// Initialize logger
	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	updateGDCStatistics(gdc, mockQuerier, run, mockRPC)

	// Assert
	assert.Greater(t, mockRPC.FetchRunStatisticsCalledCount(), 0, "FetchRunStatistics should be called")
	mockQuerier.AssertExpectations(t)
}

func TestUpdateGDCStatistics_RPCError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{
		FetchRunStatisticsFunc: func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
			return duck.RunStatistics{}, errors.New("RPC connection failed")
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

	// Execute - should handle error gracefully
	updateGDCStatistics(gdc, mockQuerier, run, mockRPC)

	// Assert - RPC was called but database operations weren't
	assert.Greater(t, mockRPC.FetchRunStatisticsCalledCount(), 0, "FetchRunStatistics should be called")
}

func TestUpdateGDCStatistics_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{
		FetchRunStatisticsFunc: func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
			return duck.RunStatistics{
				Events: 1000,
				Bytes:  5000,
				Errors: 5,
			}, nil
		},
	}

	gdc := duck.GDCConfiguration{
		ID:       1,
		Name:     "gdc1",
		IP:       "192.168.1.100",
		GRPCPort: 6001,
	}
	run := 123

	// Mock all database calls - first one returns error, others can too
	mockQuerier.On("InsertGDCEvents", mock.Anything, mock.Anything).
		Return(errors.New("database error"))
	mockQuerier.On("InsertGDCBytes", mock.Anything, mock.Anything).
		Return(errors.New("database error"))
	mockQuerier.On("InsertGDCErrorCount", mock.Anything, mock.Anything).
		Return(errors.New("database error"))

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute - should handle error gracefully
	updateGDCStatistics(gdc, mockQuerier, run, mockRPC)

	// Assert
	assert.Greater(t, mockRPC.FetchRunStatisticsCalledCount(), 0, "FetchRunStatistics should be called")
	mockQuerier.AssertExpectations(t)
}

// ===== updateLDCStatistics Tests =====

func TestUpdateLDCStatistics_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{
		FetchRunStatisticsFunc: func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
			return duck.RunStatistics{
				Events: 2000,
				Bytes:  10000,
				Errors: 10,
			}, nil
		},
	}

	ldc := duck.LDCConfiguration{
		ID:       1,
		Name:     "ldc1",
		IP:       "192.168.1.101",
		GRPCPort: 7001,
	}
	run := 123

	// Mock database calls
	mockQuerier.On("InsertLDCEvents", mock.Anything, mock.MatchedBy(func(arg database.InsertLDCEventsParams) bool {
		return arg.Run == int32(run) && arg.LdcID.Int32 == int32(ldc.ID) && arg.Events.Int64 == 2000
	})).Return(nil)

	mockQuerier.On("InsertLDCBytes", mock.Anything, mock.MatchedBy(func(arg database.InsertLDCBytesParams) bool {
		return arg.Run == int32(run) && arg.LdcID.Int32 == int32(ldc.ID) && arg.Bytes.Int64 == 10000
	})).Return(nil)

	mockQuerier.On("InsertLDCErrorCount", mock.Anything, mock.MatchedBy(func(arg database.InsertLDCErrorCountParams) bool {
		return arg.Run == int32(run) && arg.LdcID.Int32 == int32(ldc.ID) && arg.Errors.Int64 == 10
	})).Return(nil)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	updateLDCStatistics(ldc, mockQuerier, run, mockRPC)

	// Assert
	assert.Greater(t, mockRPC.FetchRunStatisticsCalledCount(), 0, "FetchRunStatistics should be called")
	mockQuerier.AssertExpectations(t)
}

func TestUpdateLDCStatistics_RPCError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)
	mockRPC := &MockRPCClient{
		FetchRunStatisticsFunc: func(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
			return duck.RunStatistics{}, errors.New("RPC connection failed")
		},
	}

	ldc := duck.LDCConfiguration{
		ID:       1,
		Name:     "ldc1",
		IP:       "192.168.1.101",
		GRPCPort: 7001,
	}
	run := 123

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute - should handle error gracefully
	updateLDCStatistics(ldc, mockQuerier, run, mockRPC)

	// Assert
	assert.Greater(t, mockRPC.FetchRunStatisticsCalledCount(), 0, "FetchRunStatistics should be called")
}

// ===== pingDevices Tests =====

func TestPingDevices_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		PingFunc: func(ctx context.Context, ip string, port int) (bool, error) {
		assert.Equal(t, "192.168.1.101", ip)
		assert.Equal(t, 7001, port)
		return true, nil
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

	// Execute
	pingDevices(ldc, ldc.GRPCPort, errorChan, mockRPC)

	// Assert
	err := <-errorChan
	assert.NoError(t, err)
	assert.Greater(t, mockRPC.PingCalledCount(), 0, "Ping should be called")
}

func TestPingDevices_NotEnabled(t *testing.T) {
	mockRPC := &MockRPCClient{}

	ldc := duck.LDCConfiguration{
		ID:       1,
		Name:     "ldc1",
		IP:       "192.168.1.101",
		GRPCPort: 7001,
		Enabled:  false, // Not enabled
	}
	errorChan := make(chan error, 1)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	pingDevices(ldc, ldc.GRPCPort, errorChan, mockRPC)

	// Assert - Ping should not be called for disabled device
	assert.Equal(t, 0, mockRPC.PingCalledCount(), "Ping should not be called")
}

func TestPingDevices_RPCError(t *testing.T) {
	mockRPC := &MockRPCClient{
		PingFunc: func(ctx context.Context, ip string, port int) (bool, error) {
		return false, errors.New("ping failed")
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

	// Execute
	pingDevices(ldc, ldc.GRPCPort, errorChan, mockRPC)

	// Assert
	err := <-errorChan
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
}

func TestPingDevices_Unsuccessful(t *testing.T) {
	mockRPC := &MockRPCClient{
		PingFunc: func(ctx context.Context, ip string, port int) (bool, error) {
		return false, nil // Ping returns false but no error
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

	// Execute
	pingDevices(ldc, ldc.GRPCPort, errorChan, mockRPC)

	// Assert
	err := <-errorChan
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ping unsuccessful")
}

// ===== startServer Tests =====

func TestStartServer_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		StartServerFunc: func(ctx context.Context, ip string, port int) error {
			assert.Equal(t, "192.168.1.100", ip)
			assert.Equal(t, 6001, port)
			return nil
		},
	}

	errorChan := make(chan error, 1)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	startServer("192.168.1.100", "gdc1", 6001, errorChan, mockRPC)

	// Assert
	err := <-errorChan
	assert.NoError(t, err)
	assert.Greater(t, mockRPC.StartServerCalledCount(), 0, "StartServer should be called")
}

func TestStartServer_Error(t *testing.T) {
	mockRPC := &MockRPCClient{
		StartServerFunc: func(ctx context.Context, ip string, port int) error {
			return errors.New("start failed")
		},
	}

	errorChan := make(chan error, 1)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	startServer("192.168.1.100", "gdc1", 6001, errorChan, mockRPC)

	// Assert
	err := <-errorChan
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start failed")
}

// ===== stopServer Tests =====

func TestStopServer_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		StopServerFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	errorChan := make(chan error, 1)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	stopServer("192.168.1.100", "gdc1", 6001, errorChan, mockRPC)

	// Assert
	err := <-errorChan
	assert.NoError(t, err)
	assert.Greater(t, mockRPC.StopServerCalledCount(), 0, "StopServer should be called")
}

func TestStopServer_Error(t *testing.T) {
	mockRPC := &MockRPCClient{
		StopServerFunc: func(ctx context.Context, ip string, port int) error {
			return errors.New("stop failed")
		},
	}

	errorChan := make(chan error, 1)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	stopServer("192.168.1.100", "gdc1", 6001, errorChan, mockRPC)

	// Assert
	err := <-errorChan
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stop failed")
}

// ===== getServerState Tests =====

func TestGetServerState_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		GetStateFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	errorChan := make(chan error, 1)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	getServerState("192.168.1.100", "gdc1", 6001, errorChan, mockRPC)

	// Assert
	err := <-errorChan
	assert.NoError(t, err)
	assert.Greater(t, mockRPC.GetStateCalledCount(), 0, "GetState should be called")
}

func TestGetServerState_Error(t *testing.T) {
	mockRPC := &MockRPCClient{
		GetStateFunc: func(ctx context.Context, ip string, port int) error {
			return errors.New("get state failed")
		},
	}

	errorChan := make(chan error, 1)

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	getServerState("192.168.1.100", "gdc1", 6001, errorChan, mockRPC)

	// Assert
	err := <-errorChan
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get state failed")
}

// ===== sendPingToAllLDCs Tests =====

func TestSendPingToAllLDCs_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		PingFunc: func(ctx context.Context, ip string, port int) (bool, error) {
			return true, nil
		},
	}

	ldcs := []duck.LDCConfiguration{
		{ID: 1, Name: "ldc1", IP: "192.168.1.101", GRPCPort: 7001, Enabled: true},
		{ID: 2, Name: "ldc2", IP: "192.168.1.102", GRPCPort: 7002, Enabled: true},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := sendPingToAllLDCs(ldcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Greater(t, mockRPC.PingCalledCount(), 0, "Ping should be called")
}

func TestSendPingToAllLDCs_PartialFailure(t *testing.T) {
	mockRPC := &MockRPCClient{
		PingFunc: func(ctx context.Context, ip string, port int) (bool, error) {
			// First LDC succeeds, second fails
			if ip == "192.168.1.101" {
				return true, nil
			}
			return false, errors.New("ping failed")
		},
	}

	ldcs := []duck.LDCConfiguration{
		{ID: 1, Name: "ldc1", IP: "192.168.1.101", GRPCPort: 7001, Enabled: true},
		{ID: 2, Name: "ldc2", IP: "192.168.1.102", GRPCPort: 7002, Enabled: true},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := sendPingToAllLDCs(ldcs, mockRPC)

	// Assert - should return false on partial failure
	assert.False(t, success)
}

func TestSendPingToAllLDCs_AllDisabled(t *testing.T) {
	mockRPC := &MockRPCClient{}

	ldcs := []duck.LDCConfiguration{
		{ID: 1, Name: "ldc1", IP: "192.168.1.101", GRPCPort: 7001, Enabled: false},
		{ID: 2, Name: "ldc2", IP: "192.168.1.102", GRPCPort: 7002, Enabled: false},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := sendPingToAllLDCs(ldcs, mockRPC)

	// Assert - should succeed (nothing to ping)
	assert.True(t, success)
	assert.Equal(t, 0, mockRPC.PingCalledCount(), "Ping should not be called")
}

func TestSendPingToAllLDCs_EmptyList(t *testing.T) {
	mockRPC := &MockRPCClient{}
	ldcs := []duck.LDCConfiguration{}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := sendPingToAllLDCs(ldcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Equal(t, 0, mockRPC.PingCalledCount(), "Ping should not be called")
}

// ===== startGDCs Tests =====

func TestStartGDCs_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		StartServerFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	gdcs := []duck.GDCConfiguration{
		{ID: 1, Name: "gdc1", IP: "192.168.1.100", GRPCPort: 6001},
		{ID: 2, Name: "gdc2", IP: "192.168.1.101", GRPCPort: 6002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := startGDCs(gdcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Greater(t, mockRPC.StartServerCalledCount(), 0, "StartServer should be called")
}

func TestStartGDCs_PartialFailure(t *testing.T) {
	mockRPC := &MockRPCClient{
		StartServerFunc: func(ctx context.Context, ip string, port int) error {
			if ip == "192.168.1.100" {
				return nil
			}
			return errors.New("start failed")
		},
	}

	gdcs := []duck.GDCConfiguration{
		{ID: 1, Name: "gdc1", IP: "192.168.1.100", GRPCPort: 6001},
		{ID: 2, Name: "gdc2", IP: "192.168.1.101", GRPCPort: 6002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := startGDCs(gdcs, mockRPC)

	// Assert
	assert.False(t, success)
}

func TestStartGDCs_EmptyList(t *testing.T) {
	mockRPC := &MockRPCClient{}
	gdcs := []duck.GDCConfiguration{}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := startGDCs(gdcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Equal(t, 0, mockRPC.StartServerCalledCount(), "StartServer should not be called")
}

// ===== startLDCs Tests =====

func TestStartLDCs_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		StartServerFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	ldcs := []duck.LDCConfiguration{
		{ID: 1, Name: "ldc1", IP: "192.168.1.101", GRPCPort: 7001},
		{ID: 2, Name: "ldc2", IP: "192.168.1.102", GRPCPort: 7002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := startLDCs(ldcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Greater(t, mockRPC.StartServerCalledCount(), 0, "StartServer should be called")
}

func TestStartLDCs_PartialFailure(t *testing.T) {
	mockRPC := &MockRPCClient{
		StartServerFunc: func(ctx context.Context, ip string, port int) error {
			if ip == "192.168.1.101" {
				return nil
			}
			return errors.New("start failed")
		},
	}

	ldcs := []duck.LDCConfiguration{
		{ID: 1, Name: "ldc1", IP: "192.168.1.101", GRPCPort: 7001},
		{ID: 2, Name: "ldc2", IP: "192.168.1.102", GRPCPort: 7002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := startLDCs(ldcs, mockRPC)

	// Assert - should return false on partial failure
	assert.False(t, success)
}

func TestStartLDCs_EmptyList(t *testing.T) {
	mockRPC := &MockRPCClient{}
	ldcs := []duck.LDCConfiguration{}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := startLDCs(ldcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Equal(t, 0, mockRPC.StartServerCalledCount(), "StartServer should not be called")
}

// ===== stopGDCs Tests =====

func TestStopGDCs_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		StopServerFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	gdcs := []duck.GDCConfiguration{
		{ID: 1, Name: "gdc1", IP: "192.168.1.100", GRPCPort: 6001},
		{ID: 2, Name: "gdc2", IP: "192.168.1.101", GRPCPort: 6002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := stopGDCs(gdcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Greater(t, mockRPC.StopServerCalledCount(), 0, "StopServer should be called")
}

func TestStopGDCs_PartialFailure(t *testing.T) {
	mockRPC := &MockRPCClient{
		StopServerFunc: func(ctx context.Context, ip string, port int) error {
			if ip == "192.168.1.100" {
				return nil
			}
			return errors.New("stop failed")
		},
	}

	gdcs := []duck.GDCConfiguration{
		{ID: 1, Name: "gdc1", IP: "192.168.1.100", GRPCPort: 6001},
		{ID: 2, Name: "gdc2", IP: "192.168.1.101", GRPCPort: 6002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := stopGDCs(gdcs, mockRPC)

	// Assert
	assert.False(t, success)
}

func TestStopGDCs_EmptyList(t *testing.T) {
	mockRPC := &MockRPCClient{}
	gdcs := []duck.GDCConfiguration{}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := stopGDCs(gdcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Equal(t, 0, mockRPC.StopServerCalledCount(), "StopServer should not be called")
}

// ===== stopLDCs Tests =====

func TestStopLDCs_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		StopServerFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	ldcs := []duck.LDCConfiguration{
		{ID: 1, Name: "ldc1", IP: "192.168.1.101", GRPCPort: 7001},
		{ID: 2, Name: "ldc2", IP: "192.168.1.102", GRPCPort: 7002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := stopLDCs(ldcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Greater(t, mockRPC.StopServerCalledCount(), 0, "StopServer should be called")
}

func TestStopLDCs_PartialFailure(t *testing.T) {
	mockRPC := &MockRPCClient{
		StopServerFunc: func(ctx context.Context, ip string, port int) error {
			if ip == "192.168.1.101" {
				return nil
			}
			return errors.New("stop failed")
		},
	}

	ldcs := []duck.LDCConfiguration{
		{ID: 1, Name: "ldc1", IP: "192.168.1.101", GRPCPort: 7001},
		{ID: 2, Name: "ldc2", IP: "192.168.1.102", GRPCPort: 7002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := stopLDCs(ldcs, mockRPC)

	// Assert
	assert.False(t, success)
}

func TestStopLDCs_EmptyList(t *testing.T) {
	mockRPC := &MockRPCClient{}
	ldcs := []duck.LDCConfiguration{}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := stopLDCs(ldcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Equal(t, 0, mockRPC.StopServerCalledCount(), "StopServer should not be called")
}

// ===== getLDCStates Tests =====

func TestGetLDCStates_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		GetStateFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	ldcs := []duck.LDCConfiguration{
		{ID: 1, Name: "ldc1", IP: "192.168.1.101", GRPCPort: 7001},
		{ID: 2, Name: "ldc2", IP: "192.168.1.102", GRPCPort: 7002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := getLDCStates(ldcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Greater(t, mockRPC.GetStateCalledCount(), 0, "GetState should be called")
}

func TestGetLDCStates_PartialFailure(t *testing.T) {
	mockRPC := &MockRPCClient{
		GetStateFunc: func(ctx context.Context, ip string, port int) error {
			if ip == "192.168.1.101" {
				return nil
			}
			return errors.New("get state failed")
		},
	}

	ldcs := []duck.LDCConfiguration{
		{ID: 1, Name: "ldc1", IP: "192.168.1.101", GRPCPort: 7001},
		{ID: 2, Name: "ldc2", IP: "192.168.1.102", GRPCPort: 7002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := getLDCStates(ldcs, mockRPC)

	// Assert
	assert.False(t, success)
}

func TestGetLDCStates_EmptyList(t *testing.T) {
	mockRPC := &MockRPCClient{}
	ldcs := []duck.LDCConfiguration{}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := getLDCStates(ldcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Equal(t, 0, mockRPC.GetStateCalledCount(), "GetState should not be called")
}

// ===== getGDCStates Tests =====

func TestGetGDCStates_Success(t *testing.T) {
	mockRPC := &MockRPCClient{
		GetStateFunc: func(ctx context.Context, ip string, port int) error {
			return nil
		},
	}

	gdcs := []duck.GDCConfiguration{
		{ID: 1, Name: "gdc1", IP: "192.168.1.100", GRPCPort: 6001},
		{ID: 2, Name: "gdc2", IP: "192.168.1.101", GRPCPort: 6002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := getGDCStates(gdcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Greater(t, mockRPC.GetStateCalledCount(), 0, "GetState should be called")
}

func TestGetGDCStates_PartialFailure(t *testing.T) {
	mockRPC := &MockRPCClient{
		GetStateFunc: func(ctx context.Context, ip string, port int) error {
			if ip == "192.168.1.100" {
				return nil
			}
			return errors.New("get state failed")
		},
	}

	gdcs := []duck.GDCConfiguration{
		{ID: 1, Name: "gdc1", IP: "192.168.1.100", GRPCPort: 6001},
		{ID: 2, Name: "gdc2", IP: "192.168.1.101", GRPCPort: 6002},
	}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := getGDCStates(gdcs, mockRPC)

	// Assert
	assert.False(t, success)
}

func TestGetGDCStates_EmptyList(t *testing.T) {
	mockRPC := &MockRPCClient{}
	gdcs := []duck.GDCConfiguration{}

	logger = duck.NewDuckLogger("test", nil, 0)

	// Execute
	success := getGDCStates(gdcs, mockRPC)

	// Assert
	assert.True(t, success)
	assert.Equal(t, 0, mockRPC.GetStateCalledCount(), "GetState should not be called")
}
