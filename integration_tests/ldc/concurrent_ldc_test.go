//go:build integration
// +build integration

package ldc_test

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLDC_MultipleLDCs_SameGDC tests that multiple LDCs can send to the same GDC simultaneously
func TestLDC_MultipleLDCs_SameGDC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database once at the beginning to ensure clean state
	testhelpers.ResetDatabase(t)

	// Disable GDC2 so that both LDCs only use GDC1 (port 6005)
	// This prevents round-robin distribution which would split events across GDCs
	dbConfig := testhelpers.GetSharedDBConfig()
	connStr := fmt.Sprintf("root:testpass@(%s:%s)/duck_test?parseTime=true", dbConfig.Hostname, dbConfig.Port)
	db, dbErr := sql.Open("mysql", connStr)
	require.NoError(t, dbErr, "Failed to connect to database")
	defer db.Close()
	_, dbErr = db.Exec("UPDATE gdcs SET enabled=false WHERE id=2")
	require.NoError(t, dbErr, "Failed to disable GDC2")

	// Start tracking GDC listener (GDC 1 on port 6005 from integration_seed.sql)
	// Both LDCs will connect to this same GDC
	gdc := testhelpers.StartTrackingGDCListener(t, 6005)
	defer gdc.Close()

	// Configure and start LDC1 (first LDC, so don't skip DB reset - it's already done above)
	ldc1ConfigPath := "integration_tests/fixtures/configs/ldc_test.yml"
	equipConfigs1 := []testhelpers.EquipmentConfig{
		{LDCID: 1, EquipmentIDs: []int{1}}, // Only enable equipment 1 for LDC1
	}
	ldc1 := testhelpers.StartTestLDCWithConfig(t, ldc1ConfigPath, equipConfigs1, "test_ldc1", true)
	defer ldc1.Stop()

	// Configure and start LDC2 (skip DB reset to preserve LDC1's equipment configuration)
	ldc2ConfigPath := "integration_tests/fixtures/configs/ldc_test.yml"
	equipConfigs2 := []testhelpers.EquipmentConfig{
		{LDCID: 2, EquipmentIDs: []int{5}}, // Only enable equipment 5 for LDC2
	}
	ldc2 := testhelpers.StartTestLDCWithConfig(t, ldc2ConfigPath, equipConfigs2, "test_ldc2", true)
	defer ldc2.Stop()

	// Start both LDCs
	var err error
	err = ldc1.StartRun()
	require.NoError(t, err, "Failed to start LDC1 run")

	err = ldc2.StartRun()
	require.NoError(t, err, "Failed to start LDC2 run")

	// Give LDCs time to establish connections to GDC
	time.Sleep(500 * time.Millisecond)

	// Get initial statistics
	initialStats1, err := ldc1.GetStatistics()
	require.NoError(t, err, "Failed to get LDC1 initial statistics")

	initialStats2, err := ldc2.GetStatistics()
	require.NoError(t, err, "Failed to get LDC2 initial statistics")

	// Create UDP senders for both LDCs
	sender1, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create LDC1 UDP sender")
	defer sender1.Close()

	sender2, err := testhelpers.CreateUDPSender("127.0.0.1", 16011) // Equipment 5 uses port 16011
	require.NoError(t, err, "Failed to create LDC2 UDP sender")
	defer sender2.Close()

	// Send events from both LDCs concurrently
	numEventsPerLDC := 10
	var wg sync.WaitGroup

	t.Log("Sending events concurrently from LDC1 and LDC2")

	// LDC1 sends events
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= numEventsPerLDC; i++ {
			packets := testhelpers.GenerateTestEvent(i, testhelpers.DefaultTestPacketCount)
			err := sender1.SendEvent(packets)
			require.NoError(t, err, "LDC1 failed to send event %d", i)
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// LDC2 sends events
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i <= numEventsPerLDC; i++ {
			packets := testhelpers.GenerateTestEvent(i, testhelpers.DefaultTestPacketCount)
			err := sender2.SendEvent(packets)
			require.NoError(t, err, "LDC2 failed to send event %d", i)
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Wait for all events to be sent
	wg.Wait()

	// Wait for both LDCs to process their events
	require.NoError(t, ldc1.WaitForProcessedEvents(initialStats1.Events+int64(numEventsPerLDC), 30*time.Second), "LDC1 did not process all events")
	require.NoError(t, ldc2.WaitForProcessedEvents(initialStats2.Events+int64(numEventsPerLDC), 30*time.Second), "LDC2 did not process all events")

	// Give GDC time to receive all events
	time.Sleep(500 * time.Millisecond)

	// Get final statistics
	finalStats1, err := ldc1.GetStatistics()
	require.NoError(t, err, "Failed to get LDC1 final statistics")

	finalStats2, err := ldc2.GetStatistics()
	require.NoError(t, err, "Failed to get LDC2 final statistics")

	t.Logf("LDC1 stats - Events: %d, Bytes: %d, Errors: %d",
		finalStats1.Events-initialStats1.Events,
		finalStats1.Bytes-initialStats1.Bytes,
		finalStats1.Errors)

	t.Logf("LDC2 stats - Events: %d, Bytes: %d, Errors: %d",
		finalStats2.Events-initialStats2.Events,
		finalStats2.Bytes-initialStats2.Bytes,
		finalStats2.Errors)

	// Verify GDC received events from both LDCs
	gdcEvents := gdc.GetReceivedEvents()
	t.Logf("GDC received events: %v", gdcEvents)

	// Should have received all events from both LDCs (events 1-10 from LDC1, 101-110 from LDC2)
	totalExpectedEvents := numEventsPerLDC * 2
	assert.Equal(t, totalExpectedEvents, len(gdcEvents), "GDC should receive all events from both LDCs")

	// Verify both LDCs processed their events
	events1Delta := finalStats1.Events - initialStats1.Events
	events2Delta := finalStats2.Events - initialStats2.Events

	assert.Equal(t, int64(numEventsPerLDC), events1Delta, "LDC1 should process all its events")
	assert.Equal(t, int64(numEventsPerLDC), events2Delta, "LDC2 should process all its events")

	// Stop both LDCs
	err = ldc1.StopRun()
	require.NoError(t, err, "Failed to stop LDC1 run")

	err = ldc2.StopRun()
	require.NoError(t, err, "Failed to stop LDC2 run")
}

// TestLDC_ConcurrentRPC_Calls tests that LDC can handle multiple concurrent RPC calls
func TestLDC_ConcurrentRPC_Calls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset database first to ensure clean state
	testhelpers.ResetDatabase(t)

	// Start tracking GDC listeners for all enabled GDCs (after reset)
	_, gdcCleanup := testhelpers.StartTrackingGDCListenersForAllEnabled(t)
	defer gdcCleanup()

	configPath := "integration_tests/fixtures/configs/ldc_test.yml"
	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	err := ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	// Send some events first
	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get initial statistics")

	sender, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	// Send a few events to have some data
	for i := 1; i <= 5; i++ {
		packets := testhelpers.GenerateTestEvent(i, testhelpers.DefaultTestPacketCount)
		err = sender.SendEvent(packets)
		require.NoError(t, err, "Failed to send event %d", i)
		time.Sleep(50 * time.Millisecond)
	}

	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+5, testhelpers.EventProcessingTimeout), "LDC did not process initial events")

	// Now make multiple concurrent RPC calls
	numConcurrentCalls := 10
	var wg sync.WaitGroup
	errors := make(chan error, numConcurrentCalls)

	t.Logf("Making %d concurrent RPC calls to LDC", numConcurrentCalls)

	// Make various RPC calls concurrently
	for i := 0; i < numConcurrentCalls; i++ {
		wg.Add(1)
		go func(callNum int) {
			defer wg.Done()

			// Rotate through different RPC methods
			switch callNum % 4 {
			case 0:
				// GetState
				_, err := ldc.GetState()
				if err != nil {
					errors <- fmt.Errorf("call %d (GetState) failed: %w", callNum, err)
					return
				}
			case 1:
				// GetStatistics
				_, err := ldc.GetStatistics()
				if err != nil {
					errors <- fmt.Errorf("call %d (GetStatistics) failed: %w", callNum, err)
					return
				}
			case 2:
				// PingDevices
				_, err := ldc.PingDevices()
				if err != nil {
					errors <- fmt.Errorf("call %d (PingDevices) failed: %w", callNum, err)
					return
				}
			case 3:
				// GetState again
				_, err := ldc.GetState()
				if err != nil {
					errors <- fmt.Errorf("call %d (GetState) failed: %w", callNum, err)
					return
				}
			}
		}(i)
	}

	// Wait for all calls to complete
	wg.Wait()
	close(errors)

	// Check for errors
	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		t.Logf("Concurrent RPC call errors: %d errors occurred", len(allErrors))
		for _, e := range allErrors {
			t.Logf("  - %v", e)
		}
	}

	// All concurrent read-only RPC calls should succeed
	successfulCalls := numConcurrentCalls - len(allErrors)
	t.Logf("Successful concurrent RPC calls: %d/%d", successfulCalls, numConcurrentCalls)

	assert.Equal(t, numConcurrentCalls, successfulCalls, "All concurrent RPC calls should succeed")

	// Verify LDC is still functional after concurrent calls
	finalState, err := ldc.GetState()
	require.NoError(t, err, "LDC should still be responsive after concurrent calls")
	// LDC should be in a valid state (either RUNNING or INITIALIZED)
	assert.Contains(t, []string{"RUNNING", "INITIALIZED"}, finalState, "LDC should be in a valid state after concurrent calls")

	finalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Should be able to get statistics after concurrent calls")

	t.Logf("Final stats after concurrent calls - Events: %d, Bytes: %d, Errors: %d",
		finalStats.Events, finalStats.Bytes, finalStats.Errors)

	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}

// TestLDC_ConcurrentStartStop tests concurrent start/stop operations
func TestLDC_ConcurrentStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start tracking GDC listeners for all enabled GDCs
	_, gdcCleanup := testhelpers.StartTrackingGDCListenersForAllEnabled(t)
	defer gdcCleanup()

	configPath := "integration_tests/fixtures/configs/ldc_test.yml"
	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	// Test rapid start/stop cycles (sequential but rapid)
	numCycles := 5
	for i := 0; i < numCycles; i++ {
		// Start
		err := ldc.StartRun()
		require.NoError(t, err, "Failed to start run in cycle %d", i)

		// Verify it's running
		state, err := ldc.GetState()
		require.NoError(t, err, "Failed to get state in cycle %d", i)
		assert.Equal(t, "RUNNING", state, "Should be RUNNING in cycle %d", i)

		// Small delay to simulate operation
		time.Sleep(50 * time.Millisecond)

		// Stop
		err = ldc.StopRun()
		require.NoError(t, err, "Failed to stop run in cycle %d", i)

		// Verify it's back to initialized
		require.NoError(t, ldc.WaitForState("INITIALIZED", testhelpers.StateChangeTimeout), "Should return to INITIALIZED in cycle %d", i)
	}

	t.Logf("Successfully completed %d rapid start/stop cycles", numCycles)

	// Try a clean start/stop
	err := ldc.StartRun()
	require.NoError(t, err, "Should be able to start after rapid cycles")

	err = ldc.StopRun()
	require.NoError(t, err, "Should be able to stop after rapid cycles")
}
