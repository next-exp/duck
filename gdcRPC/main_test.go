//go:build nohdf5

package main

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	duck "github.com/jmbenlloch/next_duck/pkg"
	pb "github.com/jmbenlloch/next_duck/rpc/control"
)

// TestServer_StartRun_StateTransition tests INITIALIZED -> RUNNING state transition
func TestServer_StartRun_StateTransition(t *testing.T) {
	s := createTestServer()
	require.Equal(t, duck.INITIALIZED, s.getState())

	// Call actual StartRun method
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.StartRun(context.Background(), req)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, "Started", resp.Msg.Message)
	assert.Equal(t, duck.RUNNING, s.getState())
}

// TestServer_StopRun_StateTransition tests RUNNING -> STOPPING state transition
func TestServer_StopRun_StateTransition(t *testing.T) {
	s := createTestServer()

	// Start first
	startReq := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), startReq)
	require.NoError(t, err)
	require.Equal(t, duck.RUNNING, s.getState())

	// Stop
	stopReq := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.StopRun(context.Background(), stopReq)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, "Stopped", resp.Msg.Message)
	// State should be STOPPING immediately after StopRun (gdcServer sets INITIALIZED asynchronously)
	assert.Equal(t, duck.STOPPING, s.getState())
}

// TestServer_GetState_ReturnsCurrentState tests that GetState returns the correct state
func TestServer_GetState_ReturnsCurrentState(t *testing.T) {
	s := createTestServer()

	// Test INITIALIZED state
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.GetState(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "INITIALIZED", resp.Msg.Message)

	// Start and test RUNNING state
	_, err = s.StartRun(context.Background(), req)
	require.NoError(t, err)

	resp, err = s.GetState(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "RUNNING", resp.Msg.Message)
}

// TestServer_GetRunStatistics_ReturnsCounters tests that GetRunStatistics returns correct metrics
func TestServer_GetRunStatistics_ReturnsCounters(t *testing.T) {
	s := createTestServer()
	s.metrics.eventCounter.Set(100)
	s.metrics.sizeCounter.Set(50000)
	s.metrics.evtErrorCounter.Set(2)

	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.GetRunStatistics(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(100), resp.Msg.Events)
	assert.Equal(t, int64(50000), resp.Msg.Bytes)
	assert.Equal(t, int64(2), resp.Msg.Errors)
}

// TestServer_PingDevices_AlwaysTrue tests that PingDevices returns true for GDC
func TestServer_PingDevices_AlwaysTrue(t *testing.T) {
	s := createTestServer()

	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.PingDevices(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.Msg.Success)
}

// TestState_String_Representation tests state string representations
func TestState_String_Representation(t *testing.T) {
	tests := []struct {
		state    duck.StateType
		expected string
	}{
		{duck.INITIALIZED, "INITIALIZED"},
		{duck.RUNNING, "RUNNING"},
		{duck.STOPPING, "STOPPING"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// The String() method is provided by the duck package
			result := tt.state.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestServer_StartRun_AlreadyRunning tests error handling when already running
func TestServer_StartRun_AlreadyRunning(t *testing.T) {
	s := createTestServer()

	// Start first time
	req := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, duck.RUNNING, s.getState())

	// Try to start again
	resp, err := s.StartRun(context.Background(), req)

	// Should return error message (not nil error - this is RPC convention)
	require.NoError(t, err)
	assert.Equal(t, "Cannot start before stop", resp.Msg.Message)
	assert.Equal(t, duck.RUNNING, s.getState()) // State unchanged
}

// TestServer_StopRun_NotStarted tests error handling when stopping without starting
func TestServer_StopRun_NotStarted(t *testing.T) {
	s := createTestServer()
	require.Equal(t, duck.INITIALIZED, s.getState())

	// Try to stop without starting
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.StopRun(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Can not stop before start", resp.Msg.Message)
	assert.Equal(t, duck.INITIALIZED, s.getState()) // State unchanged
}

// TestServer_MultipleStartStopCycles tests multiple start/stop cycles
func TestServer_MultipleStartStopCycles(t *testing.T) {
	s := createTestServer()
	req := connect.NewRequest(&pb.DuckRequest{})

	// Cycle 1
	require.Equal(t, duck.INITIALIZED, s.getState())

	resp, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Started", resp.Msg.Message)
	assert.Equal(t, duck.RUNNING, s.getState())

	resp, err = s.StopRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Stopped", resp.Msg.Message)
	assert.Equal(t, duck.STOPPING, s.getState())

	// Manually reset state for next cycle (simulating gdcServer)
	s.setState(duck.INITIALIZED)

	// Cycle 2
	resp, err = s.StartRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Started", resp.Msg.Message)
	assert.Equal(t, duck.RUNNING, s.getState())

	resp, err = s.StopRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Stopped", resp.Msg.Message)
	assert.Equal(t, duck.STOPPING, s.getState())
}

// TestServer_StateTransitions_InvalidTransitions tests invalid state transitions
func TestServer_StateTransitions_InvalidTransitions(t *testing.T) {
	t.Run("Cannot start when STOPPING", func(t *testing.T) {
		s := createTestServer()
		s.setState(duck.STOPPING)

		req := connect.NewRequest(&pb.DuckRequest{})
		resp, err := s.StartRun(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Cannot start before stop", resp.Msg.Message)
	})

	t.Run("Cannot stop when INITIALIZED", func(t *testing.T) {
		s := createTestServer()
		require.Equal(t, duck.INITIALIZED, s.getState())

		req := connect.NewRequest(&pb.DuckRequest{})
		resp, err := s.StopRun(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, "Can not stop before start", resp.Msg.Message)
	})
}

// TestServer_Structure tests server structure
func TestServer_Structure(t *testing.T) {
	// Setup
	testCtx := context.Background()
	cancel := func() {}

	s := &server{
		ctx:            testCtx,
		cancelCtx:      cancel,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}

	// Assert
	assert.NotNil(t, s.ctx)
	assert.NotNil(t, s.cancelCtx)
	assert.Equal(t, "test_config.yml", s.configFilename)
	assert.Equal(t, "test_server", s.serverName)
	assert.NotNil(t, s.logger)
	assert.NotNil(t, s.metrics)
}

// TestGetEventCounter_ZeroInitialValue tests that event counter starts at zero
func TestGetEventCounter_ZeroInitialValue(t *testing.T) {
	// Setup - create server and reset counter
	s := createTestServer()
	s.metrics.eventCounter.Set(0)

	// Act
	count := s.metrics.GetEventCounter()

	// Assert
	assert.Equal(t, int64(0), count)
}

// TestGetBytesCounter_ZeroInitialValue tests that byte counter starts at zero
func TestGetBytesCounter_ZeroInitialValue(t *testing.T) {
	// Setup - create server and reset counter
	s := createTestServer()
	s.metrics.sizeCounter.Set(0)

	// Act
	count := s.metrics.GetBytesCounter()

	// Assert
	assert.Equal(t, int64(0), count)
}

// TestGetEvtErrorCounter_ZeroInitialValue tests that error counter starts at zero
func TestGetEvtErrorCounter_ZeroInitialValue(t *testing.T) {
	// Setup - create server and reset counter
	s := createTestServer()
	s.metrics.evtErrorCounter.Set(0)

	// Act
	count := s.metrics.GetEvtErrorCounter()

	// Assert
	assert.Equal(t, int64(0), count)
}

// TestStateConstants_AreUnique tests that state constants have different values
func TestStateConstants_AreUnique(t *testing.T) {
	states := []duck.StateType{
		duck.INITIALIZED,
		duck.RUNNING,
		duck.STOPPING,
	}

	// Verify all states are unique
	uniqueStates := make(map[duck.StateType]bool)
	for _, state := range states {
		uniqueStates[state] = true
	}

	assert.Equal(t, len(states), len(uniqueStates), "All state constants should have unique values")
}

// TestServer_ThreadSafeStateAccess tests that state access is thread-safe
func TestServer_ThreadSafeStateAccess(t *testing.T) {
	s := createTestServer()

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

// Helper function to create a test server
func createTestServer() *server {
	ctx, cancel := context.WithCancel(context.Background())
	return &server{
		ctx:            ctx,
		cancelCtx:      cancel,
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
		runActiveCond:  sync.NewCond(&sync.Mutex{}),
	}
}

// ========== CONCURRENCY TESTS ==========
// These tests expose race conditions and thread-safety issues

// TestServer_StartRun_ConcurrentAccess tests that concurrent StartRun calls are handled correctly
// This test exposes a potential race condition where multiple goroutines could pass the INITIALIZED check
// before any of them sets the state to RUNNING
func TestServer_StartRun_ConcurrentAccess(t *testing.T) {
	s := createTestServer()
	require.Equal(t, duck.INITIALIZED, s.getState())

	// Set initial metrics to non-zero values to test reset
	s.metrics.eventCounter.Set(100)
	s.metrics.sizeCounter.Set(50000)
	s.metrics.evtErrorCounter.Set(5)

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

	// Final state should be RUNNING
	assert.Equal(t, duck.RUNNING, s.getState())

	// Metrics should be reset to zero (side effect of StartRun)
	assert.Equal(t, int64(0), s.metrics.GetEventCounter())
	assert.Equal(t, int64(0), s.metrics.GetBytesCounter())
	assert.Equal(t, int64(0), s.metrics.GetEvtErrorCounter())
}

// TestServer_StopRun_ConcurrentAccess tests that concurrent StopRun calls are handled correctly
func TestServer_StopRun_ConcurrentAccess(t *testing.T) {
	s := createTestServer()

	// Start first
	startReq := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), startReq)
	require.NoError(t, err)
	require.Equal(t, duck.RUNNING, s.getState())

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

	// Final state should be STOPPING (gdcServer sets INITIALIZED asynchronously)
	assert.Equal(t, duck.STOPPING, s.getState())
}

// TestServer_StartStop_RaceCondition tests rapid start/stop cycles
func TestServer_StartStop_RaceCondition(t *testing.T) {
	numCycles := 5
	for i := 0; i < numCycles; i++ {
		// Create fresh server for each cycle
		s := createTestServer()

		// Start
		startReq := connect.NewRequest(&pb.DuckRequest{})
		startResp, err := s.StartRun(context.Background(), startReq)
		require.NoError(t, err)
		assert.Equal(t, "Started", startResp.Msg.Message)
		assert.Equal(t, duck.RUNNING, s.getState())

		// Stop
		stopReq := connect.NewRequest(&pb.DuckRequest{})
		stopResp, err := s.StopRun(context.Background(), stopReq)
		require.NoError(t, err)
		assert.Equal(t, "Stopped", stopResp.Msg.Message)
		// State is STOPPING after StopRun (gdcServer sets INITIALIZED asynchronously)
		assert.Equal(t, duck.STOPPING, s.getState())
	}
}

// ========== METRICS VERIFICATION TESTS ==========

// TestServer_StartRun_ResetsMetrics verifies that StartRun resets all metrics to zero
func TestServer_StartRun_ResetsMetrics(t *testing.T) {
	s := createTestServer()

	// Set metrics to non-zero values
	s.metrics.eventCounter.Set(100)
	s.metrics.sizeCounter.Set(50000)
	s.metrics.evtErrorCounter.Set(5)
	s.metrics.incompleteEventsCounter.Set(10)
	s.metrics.filesOpenedCounter.Set(3)
	s.metrics.subRunCounter.Set(2)
	s.metrics.evtsWaitingDecodeCounter.Set(7)
	s.metrics.evtDecoderErrorCounter.Set(1)

	// Verify metrics are set
	assert.Equal(t, int64(100), s.metrics.GetEventCounter())
	assert.Equal(t, int64(50000), s.metrics.GetBytesCounter())
	assert.Equal(t, int64(5), s.metrics.GetEvtErrorCounter())

	// Start the run (should reset metrics)
	req := connect.NewRequest(&pb.DuckRequest{})
	resp, err := s.StartRun(context.Background(), req)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, "Started", resp.Msg.Message)

	// All metrics should be reset to zero
	assert.Equal(t, int64(0), s.metrics.GetEventCounter(), "Event counter should be reset")
	assert.Equal(t, int64(0), s.metrics.GetBytesCounter(), "Byte counter should be reset")
	assert.Equal(t, int64(0), s.metrics.GetEvtErrorCounter(), "Error counter should be reset")
}

// TestServer_StartRun_MetricsResetAcrossCycles verifies metrics reset across multiple cycles
func TestServer_StartRun_MetricsResetAcrossCycles(t *testing.T) {
	s := createTestServer()
	req := connect.NewRequest(&pb.DuckRequest{})

	// First cycle
	s.metrics.eventCounter.Set(100)
	s.metrics.sizeCounter.Set(50000)

	_, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, int64(0), s.metrics.GetEventCounter())

	// Stop and verify
	_, err = s.StopRun(context.Background(), req)
	require.NoError(t, err)

	// Second cycle
	s.metrics.eventCounter.Set(200)
	s.metrics.sizeCounter.Set(100000)

	_, err = s.StartRun(context.Background(), req)
	require.NoError(t, err)

	// Metrics should be reset again
	assert.Equal(t, int64(0), s.metrics.GetEventCounter(), "Event counter should be reset in second cycle")
	assert.Equal(t, int64(0), s.metrics.GetBytesCounter(), "Byte counter should be reset in second cycle")
}

// ========== CONTEXT PROPAGATION TESTS ==========

// TestServer_StartRun_ContextSetup verifies that StartRun properly sets up context values
func TestServer_StartRun_ContextSetup(t *testing.T) {
	s := createTestServer()
	require.Nil(t, s.ctx.Value("cancelCtx"), "cancelCtx should not be set initially")
	require.Nil(t, s.ctx.Value("server"), "server should not be set initially")

	// Start the run
	req := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)

	// Verify context values are set
	assert.NotNil(t, s.ctx.Value("cancelCtx"), "cancelCtx should be set in context")
	assert.NotNil(t, s.ctx.Value("server"), "server should be set in context")

	// Verify the server value is the correct instance
	serverValue := s.ctx.Value("server")
	assert.Same(t, s, serverValue, "server context value should be the same instance")
}

// TestServer_StartRun_ContextCancellation verifies that the context can be cancelled
func TestServer_StartRun_ContextCancellation(t *testing.T) {
	s := createTestServer()

	// Start the run
	req := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)

	// Get the cancel function from context
	cancelFunc, ok := s.ctx.Value("cancelCtx").(context.CancelFunc)
	require.True(t, ok, "cancelCtx should be a CancelFunc")

	// Cancel the context
	cancelFunc()

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
	s := createTestServer()

	// First start
	req := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)

	firstCtx := s.ctx
	_, ok := s.ctx.Value("cancelCtx").(context.CancelFunc)
	require.True(t, ok, "cancelCtx should be set after first StartRun")

	// Stop
	_, err = s.StopRun(context.Background(), req)
	require.NoError(t, err)

	// Second start
	_, err = s.StartRun(context.Background(), req)
	require.NoError(t, err)

	secondCtx := s.ctx
	_, ok = s.ctx.Value("cancelCtx").(context.CancelFunc)
	require.True(t, ok, "cancelCtx should be set after second StartRun")

	// Verify contexts are different
	assert.NotSame(t, firstCtx, secondCtx, "Context should be recreated on each StartRun")
	// Note: CancelFunc is a function type, so we can't use NotSame
	// Instead, we verify they behave differently by checking first is cancelled and second is not
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
}

// TestServer_DoubleCancellation tests calling StopRun twice rapidly
func TestServer_DoubleCancellation(t *testing.T) {
	s := createTestServer()

	// Start a run
	req := connect.NewRequest(&pb.DuckRequest{})
	_, err := s.StartRun(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, duck.RUNNING, s.getState())

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

	// Final state should be STOPPING (gdcServer sets INITIALIZED asynchronously)
	assert.Equal(t, duck.STOPPING, s.getState())
}
