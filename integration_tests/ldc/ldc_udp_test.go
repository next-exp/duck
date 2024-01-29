//go:build integration
// +build integration

package ldc_test

import (
	"testing"
	"time"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLDC_ReceivesUDPPackets tests that an LDC can receive and process UDP packets
func TestLDC_ReceivesUDPPackets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start tracking GDC listeners for all enabled GDCs
	_, gdcCleanup := testhelpers.StartTrackingGDCListenersForAllEnabled(t)
	defer gdcCleanup()

	// Start LDC with test configuration (uses shared MySQL)
	configPath := "integration_tests/fixtures/configs/ldc_test.yml"
	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	// Verify LDC is in INITIALIZED state
	state, err := ldc.GetState()
	require.NoError(t, err, "Failed to get LDC state")
	assert.Equal(t, "INITIALIZED", state, "LDC should be in INITIALIZED state")

	// Start the run
	err = ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	// Get initial statistics
	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get initial statistics")
	t.Logf("Initial stats - Events: %d, Bytes: %d, Errors: %d",
		initialStats.Events, initialStats.Bytes, initialStats.Errors)

	// Create UDP sender targeting the first equipment in LDC configuration
	// From ldc_test.yml: Equipment 1 is on host_port 16006
	sender, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	// Generate and send test event
	// Event ID 1, 10 packets
	packets := testhelpers.GenerateTestEvent(testhelpers.DefaultTestEventID, testhelpers.DefaultTestPacketCount)
	t.Logf("Sending event with %d packets", len(packets))

	err = sender.SendEvent(packets)
	require.NoError(t, err, "Failed to send event")

	// Wait for LDC to process the event
	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+1, testhelpers.EventProcessingTimeout), "LDC did not process event in time")

	// Get statistics after sending event
	finalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get final statistics")
	t.Logf("Final stats - Events: %d, Bytes: %d, Errors: %d",
		finalStats.Events, finalStats.Bytes, finalStats.Errors)

	// Verify exact events and bytes processed
	// Expect 1 event with 10 data packets (1400 bytes each)
	expectedEvents := int64(1)
	expectedBytesPerEvent := testhelpers.CalculateExpectedEventSize(testhelpers.DefaultTestPacketCount)
	eventsDelta := finalStats.Events - initialStats.Events
	bytesDelta := finalStats.Bytes - initialStats.Bytes
	assert.Equal(t, expectedEvents, eventsDelta)
	assert.Equal(t, int64(expectedBytesPerEvent), bytesDelta)

	// Stop the run
	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")

	// Wait for LDC to transition back to INITIALIZED
	// Note: This may take up to 5 seconds due to UDP read deadline in listenEquipment
	require.NoError(t, ldc.WaitForState("INITIALIZED", 10*time.Second), "LDC did not return to INITIALIZED state")

	state, err = ldc.GetState()
	require.NoError(t, err, "Failed to get LDC state after stop")
	assert.Equal(t, "INITIALIZED", state, "LDC should be back in INITIALIZED state after stop")
}

// TestLDC_ReceivesMultipleEvents tests that an LDC can receive and process multiple sequential events
func TestLDC_ReceivesMultipleEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start tracking GDC listeners for all enabled GDCs
	_, gdcCleanup := testhelpers.StartTrackingGDCListenersForAllEnabled(t)
	defer gdcCleanup()

	// Start LDC with test configuration
	configPath := "integration_tests/fixtures/configs/ldc_test.yml"
	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	// Start the run
	err := ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	// Create UDP sender
	sender, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	// Get initial statistics
	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get initial statistics")

	// Send multiple events
	numEvents := 5
	for i := 1; i <= numEvents; i++ {
		packets := testhelpers.GenerateTestEvent(i, testhelpers.DefaultTestPacketCount)
		err = sender.SendEvent(packets)
		require.NoError(t, err, "Failed to send event %d", i)

		// Small delay between events
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all events to be processed
	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+int64(numEvents), 10*time.Second), "LDC did not process all events in time")

	// Get final statistics
	finalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get final statistics")

	t.Logf("Sent %d events, Stats - Events: %d, Bytes: %d, Errors: %d",
		numEvents, finalStats.Events, finalStats.Bytes, finalStats.Errors)

	// Verify exact events and bytes processed
	expectedEvents := int64(numEvents)
	expectedBytesPerEvent := testhelpers.CalculateExpectedEventSize(testhelpers.DefaultTestPacketCount)
	eventsDelta := finalStats.Events - initialStats.Events
	bytesDelta := finalStats.Bytes - initialStats.Bytes
	assert.Equal(t, expectedEvents, eventsDelta)
	assert.Equal(t, int64(expectedBytesPerEvent)*expectedEvents, bytesDelta)

	// Stop the run
	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}

// TestLDC_HandlesPacketDelay tests that LDC can handle delayed UDP packets
func TestLDC_HandlesPacketDelay(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start tracking GDC listeners for all enabled GDCs
	_, gdcCleanup := testhelpers.StartTrackingGDCListenersForAllEnabled(t)
	defer gdcCleanup()

	// Start LDC
	configPath := "integration_tests/fixtures/configs/ldc_test.yml"
	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	// Start run
	err := ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	// Create UDP sender
	sender, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	// Get initial stats
	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err)

	// Generate event packets
	packets := testhelpers.GenerateTestEvent(testhelpers.DefaultTestEventID, testhelpers.DefaultTestPacketCount)

	// Send packets with longer delays
	for i, packet := range packets {
		err := sender.Send(packet)
		require.NoError(t, err, "Failed to send packet %d", i)

		// Longer delay between packets (50ms instead of 10ms)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for processing
	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+1, testhelpers.EventProcessingTimeout), "LDC did not process event in time")

	// Get statistics
	stats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get statistics")

	t.Logf("Stats with delayed packets - Events: %d, Bytes: %d, Errors: %d",
		stats.Events, stats.Bytes, stats.Errors)

	// Verify exact events and bytes processed
	expectedEvents := int64(1)
	expectedBytesPerEvent := testhelpers.CalculateExpectedEventSize(testhelpers.DefaultTestPacketCount)
	assert.Equal(t, expectedEvents, stats.Events-initialStats.Events)
	assert.Equal(t, int64(expectedBytesPerEvent), stats.Bytes-initialStats.Bytes)

	// Stop run
	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}

// TestLDC_StateTransitions tests that LDC properly transitions between states
func TestLDC_StateTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start tracking GDC listeners for all enabled GDCs
	_, gdcCleanup := testhelpers.StartTrackingGDCListenersForAllEnabled(t)
	defer gdcCleanup()

	// Start LDC
	configPath := "integration_tests/fixtures/configs/ldc_test.yml"
	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	// Test initial state
	state, err := ldc.GetState()
	require.NoError(t, err, "Failed to get initial state")
	assert.Equal(t, "INITIALIZED", state, "LDC should start in INITIALIZED state")

	// Test transition to RUNNING
	err = ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	state, err = ldc.GetState()
	require.NoError(t, err, "Failed to get state after start")
	assert.Equal(t, "RUNNING", state, "LDC should transition to RUNNING")

	// Test transition back to INITIALIZED
	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")

	require.NoError(t, ldc.WaitForState("INITIALIZED", 5*time.Second), "LDC did not transition back to INITIALIZED")

	state, err = ldc.GetState()
	require.NoError(t, err, "Failed to get state after stop")
	assert.Equal(t, "INITIALIZED", state, "LDC should transition back to INITIALIZED")

	// Test that we can start again
	err = ldc.StartRun()
	require.NoError(t, err, "Failed to restart run")

	state, err = ldc.GetState()
	require.NoError(t, err, "Failed to get state after restart")
	assert.Equal(t, "RUNNING", state, "LDC should be RUNNING after restart")

	// Final stop
	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}

// TestLDC_SocketsReadyWhenRunning verifies that when LDC reports RUNNING state,
// all equipment sockets are already bound and ready to receive data.
// This tests the readyChan mechanism that ensures state=RUNNING only after
// all sockets are open.
func TestLDC_SocketsReadyWhenRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start tracking GDC listeners for all enabled GDCs
	_, gdcCleanup := testhelpers.StartTrackingGDCListenersForAllEnabled(t)
	defer gdcCleanup()

	// Start LDC
	configPath := "integration_tests/fixtures/configs/ldc_test.yml"
	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	// Start run and wait for RUNNING state
	err := ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	state, err := ldc.GetState()
	require.NoError(t, err, "Failed to get state")
	require.Equal(t, "RUNNING", state, "LDC should be in RUNNING state")

	// Immediately send a packet (no delay between state check and send)
	// If sockets weren't ready when RUNNING was set, this packet would be lost
	sender, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get initial statistics")

	// Send a single event immediately after state is RUNNING
	packets := testhelpers.GenerateTestEvent(1, testhelpers.DefaultTestPacketCount)
	err = sender.SendEvent(packets)
	require.NoError(t, err, "Failed to send event immediately after RUNNING")

	// Wait for the event to be processed
	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+1, 5*time.Second),
		"Event should be processed - sockets must be ready when state is RUNNING")

	// Verify the event was received (no packet loss)
	finalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get final statistics")
	assert.Equal(t, int64(1), finalStats.Events-initialStats.Events,
		"Event sent immediately after RUNNING should be received (no packet loss)")

	// Cleanup
	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}
