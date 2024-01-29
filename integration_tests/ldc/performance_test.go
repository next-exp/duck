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

// TestLDC_HighThroughput_1000Events stress tests LDC with 1000 events
func TestLDC_HighThroughput_1000Events(t *testing.T) {
	t.Skip("Skipping performance test")
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start tracking GDC listeners for all enabled GDCs
	_, gdcCleanup := testhelpers.StartTrackingGDCListenersForAllEnabled(t)
	defer gdcCleanup()

	configPath := "integration_tests/fixtures/configs/ldc_test.yml"
	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	err := ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get initial statistics")

	sender, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	numEvents := 1000
	packetsPerEvent := 5 // Smaller events for throughput test

	t.Logf("Sending %d events with %d packets each", numEvents, packetsPerEvent)

	startTime := time.Now()

	for i := 1; i <= numEvents; i++ {
		packets := testhelpers.GenerateTestEvent(i, packetsPerEvent)
		err = sender.SendEvent(packets)
		require.NoError(t, err, "Failed to send event %d", i)

		// Small delay to avoid overwhelming the system
		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	sendDuration := time.Since(startTime)
	t.Logf("Sent %d events in %v (%.2f events/sec)",
		numEvents, sendDuration, float64(numEvents)/sendDuration.Seconds())

	// Wait for all events to be processed with generous timeout
	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+int64(numEvents), 3*time.Minute), "LDC did not process all events")

	finalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get final statistics")

	eventsDelta := finalStats.Events - initialStats.Events
	totalDuration := time.Since(startTime)

	t.Logf("Performance test stats - Events: %d, Bytes: %d, Errors: %d",
		eventsDelta, finalStats.Bytes-initialStats.Bytes, finalStats.Errors)
	t.Logf("Total time: %v (%.2f events/sec)", totalDuration, float64(eventsDelta)/totalDuration.Seconds())

	assert.Equal(t, int64(numEvents), eventsDelta, "Should process all 1000 events")

	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}

// TestLDC_StressTest_BurstEvents tests handling of burst traffic
func TestLDC_StressTest_BurstEvents(t *testing.T) {
	t.Skip("Skipping performance test")
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Start tracking GDC listeners for all enabled GDCs
	_, gdcCleanup := testhelpers.StartTrackingGDCListenersForAllEnabled(t)
	defer gdcCleanup()

	configPath := "integration_tests/fixtures/configs/ldc_test.yml"
	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	err := ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get initial statistics")

	sender, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	// Send bursts of events
	numBursts := 10
	eventsPerBurst := 50

	t.Logf("Sending %d bursts of %d events each", numBursts, eventsPerBurst)

	totalEvents := 0
	for burst := 0; burst < numBursts; burst++ {
		t.Logf("Starting burst %d/%d", burst+1, numBursts)

		startTime := time.Now()
		for i := 1; i <= eventsPerBurst; i++ {
			eventID := totalEvents + i
			packets := testhelpers.GenerateTestEvent(eventID, 5)
			err = sender.SendEvent(packets)
			require.NoError(t, err, "Failed to send event %d in burst %d", eventID, burst)
		}
		totalEvents += eventsPerBurst

		burstDuration := time.Since(startTime)
		eventsPerSec := float64(eventsPerBurst) / burstDuration.Seconds()
		t.Logf("  Burst %d complete: %d events in %v (%.2f events/sec)",
			burst+1, eventsPerBurst, burstDuration, eventsPerSec)

		// Wait between bursts to allow LDC to catch up
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for all events to be processed
	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+int64(totalEvents), 5*time.Minute), "LDC did not process all burst events")

	finalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get final statistics")

	eventsDelta := finalStats.Events - initialStats.Events
	t.Logf("Burst test final stats - Events processed: %d/%d, Errors: %d",
		eventsDelta, totalEvents, finalStats.Errors)

	assert.Equal(t, int64(totalEvents), eventsDelta, "Should process all burst events")

	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}
