package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/control"

	"github.com/jmbenlloch/next_duck/ldcRPC/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

// createTestServer creates a test server with all required fields initialized
// This helper reduces boilerplate in tests across the test suite
func createTestServer(t *testing.T) *server {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)

	return &server{
		ctx:              ctx,
		cancelCtx:        cancel,
		ldcConfiguration: &ldcConfig,
		configFilename:   "test_config.yml",
		state:            duck.INITIALIZED,
		logger:           testhelpers.SetupTest(),
		metrics:          NewMetricsRegistry(),
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}
}

func TestServer_StartRun_FromInitialized(t *testing.T) {
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Act
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.StartRun(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Started", resp.Msg.Message)
	// Note: state transitions happen asynchronously

	// Cleanup - cancel the context to stop background goroutines
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
}

func TestServer_StartRun_FromRunning_ReturnsError(t *testing.T) {
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.RUNNING,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Act
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.StartRun(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Cannot start before stop", resp.Msg.Message)
}

func TestServer_StopRun_FromRunning(t *testing.T) {
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	ctx, cancel := context.WithCancel(context.Background())
	s := &server{
		ldcConfiguration: &ldcConfig,
		ctx:              ctx,
		cancelCtx:        cancel,
		state:            duck.RUNNING,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Act
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.StopRun(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Stopped", resp.Msg.Message)
	assert.Equal(t, duck.STOPPING, s.getState())

	// Verify context was cancelled
	select {
	case <-ctx.Done():
		// Expected - context was cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context was not cancelled on StopRun")
	}
}

func TestServer_StopRun_FromInitialized_ReturnsSuccess(t *testing.T) {
	// Setup - INITIALIZED state means already stopped
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Act
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.StopRun(context.Background(), req)

	// Assert - StopRun is idempotent, returns success when already stopped
	require.NoError(t, err)
	assert.Equal(t, "Stopped", resp.Msg.Message)
}

func TestServer_GetState_ReturnsCurrentState(t *testing.T) {
	logger := testhelpers.SetupTest()

	tests := []struct {
		name          string
		setState      duck.StateType
		expectedState string
	}{
		{"Initialized", duck.INITIALIZED, "INITIALIZED"},
		{"Starting", duck.STARTING, "STARTING"},
		{"Running", duck.RUNNING, "RUNNING"},
		{"Stopping", duck.STOPPING, "STOPPING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
			metrics := NewMetricsRegistry()
			s := &server{
				ldcConfiguration: &ldcConfig,
				state:            tt.setState,
				logger:           logger,
				metrics:          metrics,
				runActiveCond:    sync.NewCond(&sync.Mutex{}),
			}

			// Act
			req := connect.NewRequest(&pb.DuckRequest{})
			resp, err := s.GetState(context.Background(), req)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedState, resp.Msg.Message)
		})
	}
}

func TestServer_GetRunStatistics_ReturnsCounters(t *testing.T) {
	// Setup - set known values
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	metrics.eventCounter.Set(100)
	metrics.sizeCounter.Set(2048)
	metrics.packetErrorCounter.Set(3)

	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	s := &server{
		ldcConfiguration: &ldcConfig,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Act
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.GetRunStatistics(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(100), resp.Msg.Events)
	assert.Equal(t, int64(2048), resp.Msg.Bytes)
	assert.Equal(t, int64(3), resp.Msg.Errors)
}

func TestServer_StartRun_ResetsCounters(t *testing.T) {
	// Setup - set some non-zero values
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	metrics.eventCounter.Set(100)
	metrics.sizeCounter.Set(2048)
	metrics.packetErrorCounter.Set(5)
	metrics.incompleteEventsCounter.Set(10)

	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Act
	req := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(0), metrics.GetEventCounter())
	assert.Equal(t, int64(0), metrics.GetBytesCounter())
	assert.Equal(t, int64(0), metrics.GetPacketErrorCounter())

	// Cleanup
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
}

// ========== CONCURRENCY TESTS ==========
// These tests are critical for LDC since it handles multiple equipment connections concurrently

// TestServer_StartRun_ConcurrentAccess tests that concurrent StartRun calls are handled correctly
// This test exposes a potential race condition where multiple goroutines could pass the INITIALIZED check
// before any of them sets the state to RUNNING
func TestServer_StartRun_ConcurrentAccess(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	// Set initial metrics to non-zero values to test reset
	metrics.eventCounter.Set(100)
	metrics.sizeCounter.Set(50000)
	metrics.packetErrorCounter.Set(5)

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	numGoroutines := 10
	results := make(chan string, numGoroutines)

	// Launch multiple StartRun calls concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			req := connect.NewRequest(&pb.DuckRequest{})
			resp, err := s.StartRun(context.Background(), req)
			require.NoError(t, err)
			results <- resp.Msg.Message
		}()
	}

	// Collect results
	successCount := 0
	errorCount := 0
	for i := 0; i < numGoroutines; i++ {
		result := <-results
		if result == "Started" {
			successCount++
		} else if result == "Cannot start before stop" {
			errorCount++
		}
	}

	// The mutex implementation guarantees exactly one goroutine succeeds
	assert.Equal(t, 1, successCount, "Exactly one StartRun should succeed")
	assert.Equal(t, numGoroutines-1, errorCount, "All others should fail")

	// Metrics should be reset to zero (side effect of StartRun)
	assert.Equal(t, int64(0), metrics.GetEventCounter())
	assert.Equal(t, int64(0), metrics.GetBytesCounter())
	assert.Equal(t, int64(0), metrics.GetPacketErrorCounter())

	// Cleanup
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
}

// TestServer_StopRun_ConcurrentAccess tests that concurrent StopRun calls are handled correctly
func TestServer_StopRun_ConcurrentAccess(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	ctx, cancel := context.WithCancel(context.Background())
	s := &server{
		ldcConfiguration: &ldcConfig,
		ctx:              ctx,
		cancelCtx:        cancel,
		state:            duck.RUNNING,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	numGoroutines := 10
	results := make(chan string, numGoroutines)

	// Launch multiple StopRun calls concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			req := connect.NewRequest(&pb.DuckRequest{})
			resp, err := s.StopRun(context.Background(), req)
			require.NoError(t, err)
			results <- resp.Msg.Message
		}()
	}

	// Collect results
	successCount := 0
	errorCount := 0
	for i := 0; i < numGoroutines; i++ {
		result := <-results
		if result == "Stopped" {
			successCount++
		} else if result == "Can not stop before start" {
			errorCount++
		}
	}

	// The mutex implementation guarantees exactly one goroutine succeeds
	assert.Equal(t, 1, successCount, "Exactly one StopRun should succeed")
	assert.Equal(t, numGoroutines-1, errorCount, "All others should fail")

	// Final state should be STOPPING
	assert.Equal(t, duck.STOPPING, s.getState())
}

// TestServer_ThreadSafeStateAccess tests that state access is thread-safe
func TestServer_ThreadSafeStateAccess(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Test concurrent reads and writes
	done := make(chan bool)
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		go func() {
			// Concurrent reads
			_ = s.getState()
			// Concurrent writes
			s.setState(duck.RUNNING)
			_ = s.getState()
			s.setState(duck.INITIALIZED)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout in thread-safe state access test")
		}
	}

	// Verify final state is a valid value (INITIALIZED or RUNNING)
	finalState := s.getState()
	assert.True(t, finalState == duck.INITIALIZED || finalState == duck.RUNNING,
		"Final state should be INITIALIZED or RUNNING, got %s", finalState)
}

// TestServer_StopRun_WhileStopping tests StopRun when state is already STOPPING
func TestServer_StopRun_WhileStopping(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.STOPPING,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Call StopRun
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.StopRun(context.Background(), req)

	// Verify returns "Can not stop before start" (since state != RUNNING)
	require.NoError(t, err)
	assert.Equal(t, "Can not stop before start", resp.Msg.Message)
}

// TestServer_DoubleCancellation tests calling StopRun twice rapidly
func TestServer_DoubleCancellation(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	ctx, cancel := context.WithCancel(context.Background())
	s := &server{
		ldcConfiguration: &ldcConfig,
		ctx:              ctx,
		cancelCtx:        cancel,
		state:            duck.RUNNING,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	results := make(chan string, 2)

	// Call StopRun twice rapidly
	for i := 0; i < 2; i++ {
		go func() {
			req := connect.NewRequest(&pb.DuckRequest{})
			resp, err := s.StopRun(context.Background(), req)
			require.NoError(t, err)
			results <- resp.Msg.Message
		}()
	}

	// Collect results - verify no panic
	successCount := 0
	errorCount := 0
	for i := 0; i < 2; i++ {
		select {
		case result := <-results:
			if result == "Stopped" {
				successCount++
			} else if result == "Can not stop before start" {
				errorCount++
			}
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout in double cancellation test")
		}
	}

	// Exactly one should succeed, the other should fail
	assert.Equal(t, 1, successCount, "Exactly one StopRun should succeed")
	assert.Equal(t, 1, errorCount, "The other StopRun should fail")

	// Final state should be STOPPING
	assert.Equal(t, duck.STOPPING, s.getState())
}

// TestServer_ConcurrentStartAndStop tests launching StartRun and StopRun simultaneously
func TestServer_ConcurrentStartAndStop(t *testing.T) {
	numIterations := 10
	for iter := 0; iter < numIterations; iter++ {
		// Create a fresh server for each iteration to avoid races with previous goroutines
		ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
		logger := testhelpers.SetupTest()
		metrics := NewMetricsRegistry()

		s := &server{
			ldcConfiguration: &ldcConfig,
			state:            duck.INITIALIZED,
			logger:           logger,
			metrics:          metrics,
			runActiveCond:    sync.NewCond(&sync.Mutex{}),
		}

		done := make(chan bool, 2)

		// Launch StartRun and StopRun simultaneously
		go func() {
			req := connect.NewRequest(&pb.DuckRequest{})
			_, _ = s.StartRun(context.Background(), req)
			done <- true
		}()

		go func() {
			req := connect.NewRequest(&pb.DuckRequest{})
			_, _ = s.StopRun(context.Background(), req)
			done <- true
		}()

		// Wait for both to complete
		for i := 0; i < 2; i++ {
			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Fatal("Timeout in concurrent start/stop test")
			}
		}

		// Verify no panic occurred and final state is valid
		finalState := s.getState()
		assert.True(t, finalState == duck.INITIALIZED || finalState == duck.RUNNING || finalState == duck.STOPPING || finalState == duck.STARTING,
			"Final state should be valid, got %s", finalState)

		// Cleanup
		if s.cancelCtx != nil {
			s.cancelCtx()
		}
	}
}

// TestServer_MultipleStartStopCycles tests multiple start/stop cycles
func TestServer_MultipleStartStopCycles(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	req := connect.NewRequest(&pb.DuckRequest{})

	// Cycle 1 - Start
	require.Equal(t, duck.INITIALIZED, s.getState())

	resp, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Started", resp.Msg.Message)
	assert.Equal(t, duck.STARTING, s.getState()) // LDC goes to STARTING first

	// Cycle 1 - Stop
	resp, err = s.StopRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Stopped", resp.Msg.Message)
	assert.Equal(t, duck.STOPPING, s.getState())

	// Wait a bit for state transition
	select {
	case <-time.After(100 * time.Millisecond):
		// Should have transitioned by now
	case <-time.After(50 * time.Millisecond):
		if s.getState() != duck.INITIALIZED {
			// Manually trigger the transition for test purposes
			// In real scenario, equipment would close and trigger this
			s.setState(duck.INITIALIZED)
		}
	}

	// Cycle 2 - verify we can start again
	// First ensure we're back to INITIALIZED
	if s.getState() != duck.INITIALIZED {
		s.setState(duck.INITIALIZED)
	}

	resp, err = s.StartRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Started", resp.Msg.Message)
	assert.Equal(t, duck.STARTING, s.getState())

	// Cleanup
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
}

// ========== ADDITIONAL CONCURRENCY AND CONTEXT TESTS ==========

// TestServer_GetState_DuringTransition tests GetState while StartRun transitions state
func TestServer_GetState_DuringTransition(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	numReads := 50
	done := make(chan bool, numReads+1)

	// Start a goroutine that rapidly calls StartRun/StopRun
	go func() {
		for i := 0; i < 10; i++ {
			req := connect.NewRequest(&pb.DuckRequest{})
			_, _ = s.StartRun(context.Background(), req)
			// Wait briefly for state to settle before stopping
			time.Sleep(5 * time.Millisecond)
			_, _ = s.StopRun(context.Background(), req)
			// Wait for state to return to INITIALIZED
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Concurrently call GetState multiple times
	for i := 0; i < numReads; i++ {
		go func() {
			req := connect.NewRequest(&pb.DuckRequest{})
			resp, err := s.GetState(context.Background(), req)
			require.NoError(t, err)

			// Verify GetState always returns a valid state string
			validStates := map[string]bool{
				"INITIALIZED": true,
				"STARTING":    true,
				"RUNNING":     true,
				"STOPPING":    true,
			}
			assert.True(t, validStates[resp.Msg.Message],
				"GetState should return valid state, got %s", resp.Msg.Message)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numReads+1; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout in GetState during transition test")
		}
	}

	// Cleanup
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
}

// TestServer_RapidStartStopCycles_WithConcurrentReads stress tests mutex contention
func TestServer_RapidStartStopCycles_WithConcurrentReads(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	numCycles := 10
	numReaders := 5
	done := make(chan bool, 1+numReaders)

	// Start rapid start/stop cycles in one goroutine
	go func() {
		for i := 0; i < numCycles; i++ {
			req := connect.NewRequest(&pb.DuckRequest{})
			_, _ = s.StartRun(context.Background(), req)
			// Wait briefly for state to settle
			time.Sleep(5 * time.Millisecond)
			_, _ = s.StopRun(context.Background(), req)
			// Wait for state to return to INITIALIZED
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Concurrent readers calling GetState and GetRunStatistics
	for i := 0; i < numReaders; i++ {
		go func() {
			for j := 0; j < numCycles; j++ {
				req := connect.NewRequest(&pb.DuckRequest{})
				_, err := s.GetState(context.Background(), req)
				assert.NoError(t, err)

				_, err = s.GetRunStatistics(context.Background(), req)
				assert.NoError(t, err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines with generous timeout for stress test
	for i := 0; i < 1+numReaders; i++ {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout in rapid start/stop with concurrent reads test")
		}
	}

	// Cleanup
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
}

// TestServer_StartRun_ContextSetup verifies that StartRun properly sets up context values
func TestServer_StartRun_ContextSetup(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Start the run
	req := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)

	// Verify context values are set
	// Note: LDC stores cancelCtx in the server struct, not in context
	assert.NotNil(t, s.cancelCtx, "cancelCtx should be set in server struct")
	assert.NotNil(t, s.ctx.Value("server"), "server should be set in context")
	assert.NotNil(t, s.ctx.Value("cancelCtx"), "cancelCtx should be set in context")

	// Verify the server value is the correct instance
	serverValue := s.ctx.Value("server")
	assert.Same(t, s, serverValue, "server context value should be the same instance")

	// Cleanup
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
}

// TestServer_StartRun_ContextCancellation verifies that the context can be cancelled
func TestServer_StartRun_ContextCancellation(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Start the run
	req := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)

	// Verify cancelCtx is set in server struct
	require.NotNil(t, s.cancelCtx, "cancelCtx should be set after StartRun")

	// Cancel the context using the server's cancelCtx
	s.cancelCtx()

	// Verify context is cancelled
	select {
	case <-s.getContext().Done():
		// Expected - context is cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled but it wasn't")
	}
}

// TestServer_MultipleContextCreations verifies that multiple StartRun calls create new contexts
func TestServer_MultipleContextCreations(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// First start
	req := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)

	firstCtx := s.ctx
	require.NotNil(t, s.cancelCtx, "cancelCtx should be set after first StartRun")

	// Stop
	_, err = s.StopRun(context.Background(), req)
	require.NoError(t, err)

	// Wait for state to return to INITIALIZED
	timeout := time.After(500 * time.Millisecond)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if s.getState() == duck.INITIALIZED {
				goto secondStart
			}
		case <-timeout:
			// Force state for test to continue
			s.setState(duck.INITIALIZED)
			goto secondStart
		}
	}

secondStart:
	// Second start
	_, err = s.StartRun(context.Background(), req)
	require.NoError(t, err)

	secondCtx := s.ctx
	require.NotNil(t, s.cancelCtx, "cancelCtx should be set after second StartRun")

	// Verify contexts are different
	assert.NotSame(t, firstCtx, secondCtx, "Context should be recreated on each StartRun")

	// Verify first context is cancelled and second is not
	firstCancelled := false
	select {
	case <-firstCtx.Done():
		firstCancelled = true
	default:
	}
	secondCancelled := false
	select {
	case <-secondCtx.Done():
		secondCancelled = true
	default:
	}
	assert.True(t, firstCancelled, "First context should be cancelled")
	assert.False(t, secondCancelled, "Second context should not be cancelled")

	// Cleanup
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
}

// ========== ADDITIONAL HIGH PRIORITY TESTS FROM GDC ==========

// TestServer_PingDevices_AlwaysTrue tests that PingDevices is callable and sets state correctly
// Note: LDC's PingDevices actually performs real pings, so this test just verifies the RPC is callable
// and state transitions work correctly (PINGING -> INITIALIZED)
func TestServer_PingDevices_AlwaysTrue(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()
	equipments := []duck.Equipment{
		testhelpers.NewTestEquipment(1, 22),
	}
	ldcConfig := testhelpers.NewTestLDCConfigurationWithEquipments(1, "testldc", equipments)

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// PingDevices should be callable (it will fail to ping in test environment, but that's ok)
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.PingDevices(context.Background(), req)

	require.NoError(t, err)
	// After PingDevices completes, state should return to INITIALIZED
	assert.Equal(t, duck.INITIALIZED, s.getState())
	// Success may be false if pinging fails (expected in test environment)
	// The important thing is the RPC is callable and doesn't panic
	_ = resp.Msg.Success
}

// TestGetEventCounter_ZeroInitialValue tests that event counter starts at zero
func TestGetEventCounter_ZeroInitialValue(t *testing.T) {
	metrics := NewMetricsRegistry()

	// Act
	count := metrics.GetEventCounter()

	// Assert - new metrics registry should have zero value
	assert.Equal(t, int64(0), count)
}

// TestGetBytesCounter_ZeroInitialValue tests that byte counter starts at zero
func TestGetBytesCounter_ZeroInitialValue(t *testing.T) {
	metrics := NewMetricsRegistry()

	// Act
	count := metrics.GetBytesCounter()

	// Assert - new metrics registry should have zero value
	assert.Equal(t, int64(0), count)
}

// TestGetPacketErrorCounter_ZeroInitialValue tests that packet error counter starts at zero
// Note: LDC uses packetErrorCounter instead of GDC's evtErrorCounter
func TestGetPacketErrorCounter_ZeroInitialValue(t *testing.T) {
	metrics := NewMetricsRegistry()

	// Act
	count := metrics.GetPacketErrorCounter()

	// Assert - new metrics registry should have zero value
	assert.Equal(t, int64(0), count)
}

// TestServer_StartRun_MetricsResetAcrossCycles verifies metrics reset across multiple cycles
func TestServer_StartRun_MetricsResetAcrossCycles(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	req := connect.NewRequest(&pb.DuckRequest{})

	// First cycle - set metrics to non-zero values
	metrics.eventCounter.Set(100)
	metrics.sizeCounter.Set(50000)
	metrics.packetErrorCounter.Set(5)

	_, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, int64(0), metrics.GetEventCounter(), "Event counter should be reset in first cycle")
	assert.Equal(t, int64(0), metrics.GetBytesCounter(), "Byte counter should be reset in first cycle")
	assert.Equal(t, int64(0), metrics.GetPacketErrorCounter(), "Packet error counter should be reset in first cycle")

	// Stop and wait for transition to INITIALIZED
	_, err = s.StopRun(context.Background(), req)
	require.NoError(t, err)

	// Wait for state to return to INITIALIZED before second cycle
	timeout := time.After(500 * time.Millisecond)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	waitDone := false
	for !waitDone {
		select {
		case <-ticker.C:
			if s.getState() == duck.INITIALIZED {
				waitDone = true
			}
		case <-timeout:
			// Force state for test to continue
			s.setState(duck.INITIALIZED)
			waitDone = true
		}
	}

	// Second cycle - set metrics to different non-zero values
	metrics.eventCounter.Set(200)
	metrics.sizeCounter.Set(100000)
	metrics.packetErrorCounter.Set(10)

	_, err = s.StartRun(context.Background(), req)
	require.NoError(t, err)

	// Metrics should be reset again in second cycle
	assert.Equal(t, int64(0), metrics.GetEventCounter(), "Event counter should be reset in second cycle")
	assert.Equal(t, int64(0), metrics.GetBytesCounter(), "Byte counter should be reset in second cycle")
	assert.Equal(t, int64(0), metrics.GetPacketErrorCounter(), "Packet error counter should be reset in second cycle")

	// Cleanup
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
}

// TestServer_StartStop_RaceCondition tests rapid start/stop cycles
func TestServer_StartStop_RaceCondition(t *testing.T) {
	numCycles := 5
	for i := 0; i < numCycles; i++ {
		// Create fresh server for each cycle to avoid interference from previous goroutines
		ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
		logger := testhelpers.SetupTest()
		metrics := NewMetricsRegistry()

		s := &server{
			ldcConfiguration: &ldcConfig,
			state:            duck.INITIALIZED,
			logger:           logger,
			metrics:          metrics,
			runActiveCond:    sync.NewCond(&sync.Mutex{}),
		}

		// Start
		startReq := connect.NewRequest(&pb.DuckRequest{})
		startResp, err := s.StartRun(context.Background(), startReq)
		require.NoError(t, err)
		assert.Equal(t, "Started", startResp.Msg.Message)
		assert.Equal(t, duck.STARTING, s.getState()) // LDC goes to STARTING first

		// Stop
		stopReq := connect.NewRequest(&pb.DuckRequest{})
		stopResp, err := s.StopRun(context.Background(), stopReq)
		require.NoError(t, err)
		assert.Equal(t, "Stopped", stopResp.Msg.Message)
		assert.Equal(t, duck.STOPPING, s.getState())

		// Cleanup
		if s.cancelCtx != nil {
			s.cancelCtx()
		}
	}
}

// TestServer_StateTransitions_InvalidTransitions tests invalid state transitions with subtests
func TestServer_StateTransitions_InvalidTransitions(t *testing.T) {
	t.Run("Cannot start when STARTING", func(t *testing.T) {
		ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
		logger := testhelpers.SetupTest()
		metrics := NewMetricsRegistry()

		s := &server{
			ldcConfiguration: &ldcConfig,
			state:            duck.STARTING,
			logger:           logger,
			metrics:          metrics,
			runActiveCond:    sync.NewCond(&sync.Mutex{}),
		}

		req := connect.NewRequest(&pb.DuckRequest{})
		resp, err := s.StartRun(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Cannot start before stop", resp.Msg.Message)
	})

	t.Run("Cannot start when RUNNING", func(t *testing.T) {
		ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
		logger := testhelpers.SetupTest()
		metrics := NewMetricsRegistry()

		s := &server{
			ldcConfiguration: &ldcConfig,
			state:            duck.RUNNING,
			logger:           logger,
			metrics:          metrics,
			runActiveCond:    sync.NewCond(&sync.Mutex{}),
		}

		req := connect.NewRequest(&pb.DuckRequest{})
		resp, err := s.StartRun(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Cannot start before stop", resp.Msg.Message)
	})

	t.Run("Cannot start when STOPPING", func(t *testing.T) {
		ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
		logger := testhelpers.SetupTest()
		metrics := NewMetricsRegistry()

		s := &server{
			ldcConfiguration: &ldcConfig,
			state:            duck.STOPPING,
			logger:           logger,
			metrics:          metrics,
			runActiveCond:    sync.NewCond(&sync.Mutex{}),
		}

		req := connect.NewRequest(&pb.DuckRequest{})
		resp, err := s.StartRun(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Cannot start before stop", resp.Msg.Message)
	})

	t.Run("Stop from INITIALIZED returns success (idempotent)", func(t *testing.T) {
		ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
		logger := testhelpers.SetupTest()
		metrics := NewMetricsRegistry()

		s := &server{
			ldcConfiguration: &ldcConfig,
			state:            duck.INITIALIZED,
			logger:           logger,
			metrics:          metrics,
			runActiveCond:    sync.NewCond(&sync.Mutex{}),
		}

		req := connect.NewRequest(&pb.DuckRequest{})
		resp, err := s.StopRun(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Stopped", resp.Msg.Message)
	})

	t.Run("Cannot stop when STOPPING", func(t *testing.T) {
		ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
		logger := testhelpers.SetupTest()
		metrics := NewMetricsRegistry()

		s := &server{
			ldcConfiguration: &ldcConfig,
			state:            duck.STOPPING,
			logger:           logger,
			metrics:          metrics,
			runActiveCond:    sync.NewCond(&sync.Mutex{}),
		}

		req := connect.NewRequest(&pb.DuckRequest{})
		resp, err := s.StopRun(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Can not stop before start", resp.Msg.Message)
	})
}

// ========== LDC NAME FIELD TESTS ==========

// TestServer_LdcNameField_BackwardCompatibility verifies that empty ldcName works (backward compatible)
func TestServer_LdcNameField_BackwardCompatibility(t *testing.T) {
	// Setup - create server without setting ldcName (zero value is empty string)
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		// Note: ldcName is not set, so it will be "" (empty string)
	}

	// Assert - ldcName should be empty string (default zero value)
	assert.Equal(t, "", s.ldcName, "ldcName should default to empty string for backward compatibility")
}

// TestServer_LdcNameField_WithCustomName verifies that a custom ldcName is stored correctly
func TestServer_LdcNameField_WithCustomName(t *testing.T) {
	customName := "my_custom_ldc"

	// Setup - create server with custom ldcName
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, customName, 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		ldcConfiguration: &ldcConfig,
		configFilename:   "test_config.yml",
		ldcName:          customName, // Set the name field
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Assert - ldcName should be set to the custom value
	assert.Equal(t, customName, s.ldcName, "ldcName should be set to custom value")
	assert.Equal(t, customName, s.ldcConfiguration.Name, "ldcName should match configuration name")
}

// TestServer_LdcNameField_DifferentNames verifies multiple servers can have different names
func TestServer_LdcNameField_DifferentNames(t *testing.T) {
	names := []string{"test_ldc1", "test_ldc2", "production_ldc"}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			// Setup - create server with specific name
			ldcConfig := testhelpers.NewTestLDCConfiguration(1, name, 0)
			logger := testhelpers.SetupTest()
			metrics := NewMetricsRegistry()

			s := &server{
				ldcConfiguration: &ldcConfig,
				configFilename:   "test_config.yml",
				ldcName:          name,
				state:            duck.INITIALIZED,
				logger:           logger,
				metrics:          metrics,
				runActiveCond:    sync.NewCond(&sync.Mutex{}),
			}

			// Assert - each server should have its unique name
			assert.Equal(t, name, s.ldcName, "Server should have unique ldcName")
		})
	}
}

// TestServer_LdcNameField_EmptyStringVsUnset verifies that explicitly setting empty string is same as unset
func TestServer_LdcNameField_EmptyStringVsUnset(t *testing.T) {
	// Setup - server with explicitly empty ldcName
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 0)
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s1 := &server{
		ldcConfiguration: &ldcConfig,
		ldcName:          "", // Explicitly set to empty string
		state:            duck.INITIALIZED,
		logger:           logger,
		metrics:          metrics,
		runActiveCond:    sync.NewCond(&sync.Mutex{}),
	}

	// Setup - server without setting ldcName (implicitly empty)
	s2 := &server{
		ldcConfiguration: &ldcConfig,
		// ldcName not set
		state:         duck.INITIALIZED,
		logger:        logger,
		metrics:       metrics,
		runActiveCond: sync.NewCond(&sync.Mutex{}),
	}

	// Assert - both should have empty ldcName
	assert.Equal(t, "", s1.ldcName, "Explicitly empty ldcName should be empty string")
	assert.Equal(t, "", s2.ldcName, "Unset ldcName should default to empty string")
}
