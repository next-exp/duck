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

// TestLDC_MultipleGDCs_RoundRobinDistribution tests that LDC distributes events across multiple GDCs using round-robin
func TestLDC_MultipleGDCs_RoundRobinDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/ldc_multi_gdc_test.yml"

	// Start tracking GDC listeners (ports from integration_seed.sql: GDC 1=6005, GDC 2=6006)
	gdc1 := testhelpers.StartTrackingGDCListener(t, 6005)
	defer gdc1.Close()

	gdc2 := testhelpers.StartTrackingGDCListener(t, 6006)
	defer gdc2.Close()

	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	err := ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	// Give LDC time to establish connections to GDCs
	time.Sleep(500 * time.Millisecond)

	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get initial statistics")

	// Send multiple events to test round-robin distribution
	sender, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	numEvents := 10
	for i := 1; i <= numEvents; i++ {
		packets := testhelpers.GenerateTestEvent(i, testhelpers.DefaultTestPacketCount)
		err = sender.SendEvent(packets)
		require.NoError(t, err, "Failed to send event %d", i)
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all events to be processed
	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+int64(numEvents), 15*time.Second), "LDC did not process all events")

	// Give GDCs time to receive all data
	time.Sleep(500 * time.Millisecond)

	finalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get final statistics")

	t.Logf("Round-robin test stats - Events: %d, Bytes: %d, Errors: %d",
		finalStats.Events, finalStats.Bytes, finalStats.Errors)

	// Verify events were processed
	eventsDelta := finalStats.Events - initialStats.Events
	assert.Equal(t, int64(numEvents), eventsDelta, "Should process all events")

	// Verify round-robin distribution
	gdc1Events := gdc1.GetReceivedEvents()
	gdc2Events := gdc2.GetReceivedEvents()

	t.Logf("GDC1 received events: %v", gdc1Events)
	t.Logf("GDC2 received events: %v", gdc2Events)

	// Expected: GDC1 gets odd events (1,3,5,7,9), GDC2 gets even events (2,4,6,8,10)
	expectedGDC1 := []int{1, 3, 5, 7, 9}
	expectedGDC2 := []int{2, 4, 6, 8, 10}

	assert.Equal(t, expectedGDC1, gdc1Events, "GDC1 should receive odd-numbered events")
	assert.Equal(t, expectedGDC2, gdc2Events, "GDC2 should receive even-numbered events")

	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}
