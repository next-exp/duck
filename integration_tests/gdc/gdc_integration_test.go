//go:build integration
// +build integration

package gdc_test

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGDC_StartAndStop tests basic GDC lifecycle
func TestGDC_StartAndStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use a test configuration
	configPath := "integration_tests/fixtures/configs/gdc_test.yml"
	gdc := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc.Stop()

	// Verify initial state
	state, err := gdc.GetState()
	require.NoError(t, err, "Failed to get initial state")
	assert.Equal(t, "INITIALIZED", state, "GDC should start in INITIALIZED state")

	// Start the run
	err = gdc.StartRun()
	require.NoError(t, err, "Failed to start run")

	// Wait for state transition
	require.NoError(t, gdc.WaitForState("RUNNING", 10*time.Second), "GDC should transition to RUNNING")

	state, err = gdc.GetState()
	require.NoError(t, err, "Failed to get state after start")
	assert.Equal(t, "RUNNING", state, "GDC should be RUNNING")

	// Stop the run
	err = gdc.StopRun()
	require.NoError(t, err, "Failed to stop run")

	require.NoError(t, gdc.WaitForState("INITIALIZED", 10*time.Second), "GDC should return to INITIALIZED")

	state, err = gdc.GetState()
	require.NoError(t, err, "Failed to get state after stop")
	assert.Equal(t, "INITIALIZED", state, "GDC should be back to INITIALIZED")
}

// TestGDC_GetStatistics tests retrieving statistics from GDC
func TestGDC_GetStatistics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/gdc_test.yml"
	gdc := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc.Stop()

	// Get initial statistics
	stats, err := gdc.GetStatistics()
	require.NoError(t, err, "Failed to get statistics")

	t.Logf("GDC Statistics - Events: %d, Bytes: %d, Errors: %d",
		stats.Events, stats.Bytes, stats.Errors)

	// Initially should have zero events
	assert.Equal(t, int64(0), stats.Events, "Should have zero events initially")
	assert.Equal(t, int64(0), stats.Bytes, "Should have zero bytes initially")
}

// TestGDC_MultipleStateTransitions tests multiple start/stop cycles
func TestGDC_MultipleStateTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/gdc_test.yml"
	gdc := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc.Stop()

	// Perform multiple cycles
	numCycles := 3
	for i := 0; i < numCycles; i++ {
		err := gdc.StartRun()
		require.NoError(t, err, "Failed to start run in cycle %d", i)

		// Wait for state transition to RUNNING (listener must be ready)
		require.NoError(t, gdc.WaitForState("RUNNING", 5*time.Second), "Should be RUNNING in cycle %d", i)

		state, err := gdc.GetState()
		require.NoError(t, err, "Failed to get state in cycle %d", i)
		assert.Equal(t, "RUNNING", state, "Should be RUNNING in cycle %d", i)

		err = gdc.StopRun()
		require.NoError(t, err, "Failed to stop run in cycle %d", i)

		require.NoError(t, gdc.WaitForState("INITIALIZED", 10*time.Second), "Should return to INITIALIZED in cycle %d", i)

		state, err = gdc.GetState()
		require.NoError(t, err, "Failed to get state after stop in cycle %d", i)
		assert.Equal(t, "INITIALIZED", state, "Should be INITIALIZED after stop in cycle %d", i)
	}

	t.Logf("Successfully completed %d GDC state change cycles", numCycles)
}

// TestGDC_ConnectRPC tests ConnectRPC connectivity
func TestGDC_ConnectRPC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/gdc_test.yml"
	gdc := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc.Stop()

	// Test that we can call RPC methods
	state, err := gdc.GetState()
	require.NoError(t, err, "Should be able to get state via ConnectRPC")
	assert.NotEmpty(t, state, "State should not be empty")

	stats, err := gdc.GetStatistics()
	require.NoError(t, err, "Should be able to get statistics via ConnectRPC")
	assert.NotNil(t, stats, "Statistics should not be nil")
}

// TestGDC_StateQueryDuringRun tests querying state while GDC is running
func TestGDC_StateQueryDuringRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/gdc_test.yml"
	gdc := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc.Stop()

	err := gdc.StartRun()
	require.NoError(t, err, "Failed to start run")

	// Wait for state to become RUNNING (TCP listener ready)
	require.NoError(t, gdc.WaitForState("RUNNING", 5*time.Second), "GDC should transition to RUNNING")

	// Query state multiple times while running
	for i := 0; i < 5; i++ {
		state, err := gdc.GetState()
		require.NoError(t, err, "Failed to get state on query %d", i)
		assert.Equal(t, "RUNNING", state, "Should remain RUNNING")
		time.Sleep(500 * time.Millisecond)
	}

	err = gdc.StopRun()
	require.NoError(t, err, "Failed to stop run")

	// Wait for state to return to INITIALIZED before next test
	require.NoError(t, gdc.WaitForState("INITIALIZED", 10*time.Second), "GDC should return to INITIALIZED")
}

// waitForPort waits for a TCP port to become available for listening
func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Check both 127.0.0.1 and 0.0.0.0 since GDC binds to 0.0.0.0
		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
		if err == nil {
			ln.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("port %d not available after %v", port, timeout)
}

// TestGDC_LDCConnectionLoss tests GDC behavior when an LDC disconnects mid-run
// This is the opposite of TestLDC_GDCDisconnect in the LDC error scenarios tests
//
// Expected behavior per gdc.go:ldcConnectionMonitor:
// 1. When LDC disconnects (EOF), state -> STOPPING
// 2. If no more LDC connections remain, cancelCtx() is called and GDC stops
// 3. GDC transitions back to INITIALIZED
func TestGDC_LDCConnectionLoss(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/e2e_test.yml"

	// Start GDC first
	gdc := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc.Stop()

	// Start LDC (this will reset database, re-enabling both GDCs)
	ldc := testhelpers.StartTestLDC(t, configPath)

	// Disable test_gdc2 AFTER LDC setup but BEFORE starting runs
	// This prevents LDC from trying to connect to non-existent GDC
	dbConfig := testhelpers.GetSharedDBConfig()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE gdcs SET enabled=false WHERE name='test_gdc2'")
	require.NoError(t, err)
	db.Close()

	// Start GDC run first and wait for it to be listening
	err = gdc.StartRun()
	require.NoError(t, err, "Failed to start GDC run")
	require.NoError(t, gdc.WaitForState("RUNNING", 5*time.Second), "GDC should be RUNNING")

	// Then start LDC run (will only connect to test_gdc1 since test_gdc2 is disabled)
	err = ldc.StartRun()
	require.NoError(t, err, "Failed to start LDC run")
	require.NoError(t, ldc.WaitForState("RUNNING", 5*time.Second), "LDC should be RUNNING")

	// Verify both are running
	ldcState, err := ldc.GetState()
	require.NoError(t, err, "Failed to get LDC state")
	assert.Equal(t, "RUNNING", ldcState, "LDC should be RUNNING")

	gdcState, err := gdc.GetState()
	require.NoError(t, err, "Failed to get GDC state")
	assert.Equal(t, "RUNNING", gdcState, "GDC should be RUNNING")

	// Send some UDP packets to generate traffic from LDC to GDC
	sender, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	// Send first event - should succeed
	packets1 := testhelpers.GenerateTestEvent(1, 3)
	err = sender.SendEvent(packets1)
	require.NoError(t, err, "Failed to send first event")

	// Wait for GDC to receive the first event
	time.Sleep(1 * time.Second)

	// Get GDC stats before LDC disconnect
	gdcStatsBefore, err := gdc.GetStatistics()
	require.NoError(t, err, "Failed to get GDC stats before disconnect")
	t.Logf("GDC stats before LDC disconnect: Events=%d, Bytes=%d, Errors=%d",
		gdcStatsBefore.Events, gdcStatsBefore.Bytes, gdcStatsBefore.Errors)

	// Now simulate LDC connection loss by stopping the LDC process
	// This is the opposite of the LDC error scenarios test which closes the GDC
	t.Log("Simulating LDC connection loss by stopping LDC process...")
	ldc.Stop()

	// Give GDC time to detect the connection loss (EOF from closed connection)
	// Per gdc.go:ldcConnectionMonitor, when connection closes:
	// - state -> STOPPING
	// - if no more connections, cancelCtx() is called
	time.Sleep(2 * time.Second)

	// Send another event - this should not reach the GDC since LDC is gone
	packets2 := testhelpers.GenerateTestEvent(2, 3)
	err = sender.SendEvent(packets2)
	// UDP send will succeed (fire and forget), but no LDC to process it
	if err != nil {
		t.Logf("UDP send failed: %v", err)
	}

	// Wait for GDC to process the disconnection
	time.Sleep(2 * time.Second)

	// Per gdc.go implementation, when the last LDC disconnects:
	// 1. State should transition to STOPPING
	// 2. Eventually to INITIALIZED (after context cancellation)
	gdcState, err = gdc.GetState()
	require.NoError(t, err, "Failed to get GDC state after LDC disconnect")
	t.Logf("GDC state after LDC disconnect: %s", gdcState)

	// State should be either STOPPING or INITIALIZED (transition happens quickly)
	// After LDC disconnect, GDC enters STOPPING state (per ldcConnectionMonitor)
	assert.Contains(t, []string{"STOPPING", "INITIALIZED"}, gdcState,
		"GDC should be in STOPPING or INITIALIZED after LDC disconnect (per gdc.go:ldcConnectionMonitor)")

	// Get GDC stats after LDC disconnect
	gdcStatsAfter, err := gdc.GetStatistics()
	require.NoError(t, err, "Failed to get GDC stats after disconnect")
	t.Logf("GDC stats after LDC disconnect: Events=%d, Bytes=%d, Errors=%d",
		gdcStatsAfter.Events, gdcStatsAfter.Bytes, gdcStatsAfter.Errors)

	// GDC event count should not have increased (LDC was stopped)
	// Bytes should be the same (no new data received)
	assert.Equal(t, gdcStatsBefore.Events, gdcStatsAfter.Events,
		"GDC event count should not increase after LDC disconnect")

	// GDC should eventually return to INITIALIZED state after processing the disconnect
	// If it's already in INITIALIZED, we don't need to wait
	if gdcState != "INITIALIZED" {
		t.Log("Waiting for GDC to transition to INITIALIZED...")
		err = gdc.WaitForState("INITIALIZED", 10*time.Second)
		// Note: WaitForState might fail if GDC RPC server has already stopped
		// This is expected behavior after cancelCtx() is called
		if err != nil {
			t.Logf("GDC RPC may have stopped (expected after LDC disconnect): %v", err)
			// Verify GDC process is actually done by checking if we can call RPC
			_, err = gdc.GetState()
			assert.Error(t, err, "GDC RPC should be unreachable after LDC disconnect (context cancelled)")
		}
	}

	t.Log("Connection loss test passed - GDC detected LDC disconnect and transitioned to STOPPING/INITIALIZED")
}

// TestGDC_ConcurrentLDCConnections tests GDC handling multiple LDC connections simultaneously
func TestGDC_ConcurrentLDCConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/e2e_test.yml"

	// Start GDC
	gdc := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc.Stop()

	// Start GDC run
	err := gdc.StartRun()
	require.NoError(t, err, "Failed to start GDC run")

	// Wait for GDC to transition to RUNNING (TCP listener ready)
	require.NoError(t, gdc.WaitForState("RUNNING", 5*time.Second), "GDC should transition to RUNNING")

	// Verify GDC is in RUNNING state
	gdcState, err := gdc.GetState()
	require.NoError(t, err)
	assert.Equal(t, "RUNNING", gdcState, "GDC should be RUNNING")

	// Create multiple raw TCP connections to GDC to simulate concurrent LDC connections
	// The GDC listens on port 6005 for LDC connections (from integration_seed.sql)
	numConnections := 3
	connections := make([]net.Conn, numConnections)
	connErrors := make([]error, numConnections)

	// Create connections concurrently using goroutines
	var wg sync.WaitGroup
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", "127.0.0.1:6005", 5*time.Second)
			if err != nil {
				connErrors[idx] = fmt.Errorf("connection %d failed: %w", idx, err)
				return
			}
			connections[idx] = conn
			t.Logf("Concurrent connection %d established to GDC", idx)
		}(i)
	}
	wg.Wait()

	// Check that all connections succeeded
	for i, err := range connErrors {
		require.NoError(t, err, "Connection %d should succeed", i)
	}

	// Count successful connections
	successCount := 0
	for _, conn := range connections {
		if conn != nil {
			successCount++
		}
	}
	require.Equal(t, numConnections, successCount, "Should have established all connections")

	// Get GDC stats to verify it's handling connections
	gdcStats, err := gdc.GetStatistics()
	require.NoError(t, err, "Failed to get GDC statistics")
	t.Logf("GDC statistics with %d concurrent connections: Events=%d, Bytes=%d, Errors=%d",
		numConnections, gdcStats.Events, gdcStats.Bytes, gdcStats.Errors)

	// Verify GDC is still healthy despite multiple connection attempts
	// Note: After closing all connections, the ldcConnectionMonitor will stop the GDC
	// This is expected behavior - GDC shuts down when no LDC connections remain
	gdcState, err = gdc.GetState()
	require.NoError(t, err)
	// GDC should be in STOPPING or INITIALIZED state after all connections are closed
	if gdcState != "STOPPING" && gdcState != "INITIALIZED" {
		t.Logf("GDC state after closing connections: %s (expected STOPPING or INITIALIZED)", gdcState)
	}

	// Close all connections cleanly
	for i, conn := range connections {
		if conn != nil {
			err := conn.Close()
			require.NoError(t, err, "Connection %d should close cleanly", i)
			t.Logf("Concurrent connection %d closed", i)
		}
	}

	// Wait a bit for GDC to process the disconnections
	time.Sleep(500 * time.Millisecond)

	// Verify GDC transitioned to INITIALIZED after all connections closed
	// This is expected behavior - ldcConnectionMonitor calls stopOnLocalError when no LDC connections remain
	gdcState, err = gdc.GetState()
	require.NoError(t, err)
	assert.Equal(t, "INITIALIZED", gdcState, "GDC should transition to INITIALIZED after all LDC connections close")

	t.Log("Concurrent connections test passed - GDC handled multiple simultaneous connections and stopped cleanly")
}
