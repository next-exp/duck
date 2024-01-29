//go:build integration
// +build integration

package e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSingleGDCEnvironment(t *testing.T) {
	dbConfig := testhelpers.GetSharedDBConfig()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("UPDATE gdcs SET enabled=false WHERE name='test_gdc2'")
	require.NoError(t, err)
	_, err = db.Exec("UPDATE ldcs SET enabled=false WHERE name='test_ldc2'")
	require.NoError(t, err)
}

func TestE2E_StateSynchronization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/e2e_test.yml"

	gdc := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc.Stop()

	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	setupSingleGDCEnvironment(t)

	ldcState, err := ldc.GetState()
	require.NoError(t, err)
	assert.Equal(t, "INITIALIZED", ldcState)

	gdcState, err := gdc.GetState()
	require.NoError(t, err)
	assert.Equal(t, "INITIALIZED", gdcState)

	err = gdc.StartRun()
	require.NoError(t, err)
	require.NoError(t, gdc.WaitForState("RUNNING", testhelpers.StateChangeTimeout))

	err = ldc.StartRun()
	require.NoError(t, err)
	require.NoError(t, ldc.WaitForState("RUNNING", testhelpers.StateChangeTimeout))

	err = ldc.StopRun()
	require.NoError(t, err)

	err = gdc.StopRun()
	require.NoError(t, err)

	require.NoError(t, ldc.WaitForState("INITIALIZED", testhelpers.StateChangeTimeout))
	require.NoError(t, gdc.WaitForState("INITIALIZED", testhelpers.StateChangeTimeout))

	t.Log("State synchronization test passed")
}

func TestE2E_APIControl(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/e2e_test.yml"

	gdc := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc.Stop()

	ldc := testhelpers.StartTestLDC(t, configPath)
	defer ldc.Stop()

	setupSingleGDCEnvironment(t)

	api := testhelpers.StartTestAPI(t, configPath)
	defer api.Stop()

	ctx := context.Background()
	err := api.StartRun(ctx)
	require.NoError(t, err, "Failed to start run via API")

	require.NoError(t, gdc.WaitForState("RUNNING", testhelpers.StateChangeTimeout), "GDC should be RUNNING")
	require.NoError(t, ldc.WaitForState("RUNNING", testhelpers.StateChangeTimeout), "LDC should be RUNNING")

	runNumber, err := api.GetRunNumber(ctx)
	require.NoError(t, err, "Failed to get run number")
	t.Logf("Run number: %d", runNumber)

	sender, err := testhelpers.CreateUDPSender("127.0.0.1", 16006)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	ldcInitialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get LDC initial statistics")

	numEvents := 3
	for eventID := 1; eventID <= numEvents; eventID++ {
		packets := testhelpers.GenerateTestEvent(eventID, testhelpers.DefaultTestPacketCount)
		err = sender.SendEvent(packets)
		require.NoError(t, err, "Failed to send event %d", eventID)
	}

	require.NoError(t, ldc.WaitForProcessedEvents(ldcInitialStats.Events+int64(numEvents), testhelpers.EventProcessingTimeout),
		"LDC did not process all events in time")

	time.Sleep(3 * time.Second)

	ldcFinalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get LDC final statistics")
	gdcFinalStats, err := gdc.GetStatistics()
	require.NoError(t, err, "Failed to get GDC final statistics")

	t.Logf("Final Stats - LDC: Events=%d, Bytes=%d | GDC: Events=%d, Bytes=%d",
		ldcFinalStats.Events, ldcFinalStats.Bytes, gdcFinalStats.Events, gdcFinalStats.Bytes)

	err = api.StopRun(ctx)
	require.NoError(t, err, "Failed to stop run via API")

	t.Log("Verifying run metadata in database...")
	runStats := testhelpers.GetRunStatistics(t, int(runNumber))
	testhelpers.VerifyRunTimestamps(t, runStats.Run)

	t.Logf("Run %d statistics:", runNumber)
	for ldcID, stats := range runStats.LDCStats {
		t.Logf("  LDC%d: Events=%d, Bytes=%d, Errors=%d", ldcID, stats.Events, stats.Bytes, stats.Errors)
	}
	for gdcID, stats := range runStats.GDCStats {
		t.Logf("  GDC%d: Events=%d, Bytes=%d, Errors=%d", gdcID, stats.Events, stats.Bytes, stats.Errors)
	}

	require.Contains(t, runStats.LDCStats, 1, "LDC1 stats should be present")
	ldc1Stats := runStats.LDCStats[1]
	assert.Equal(t, int64(numEvents), ldc1Stats.Events, "LDC1 events should match")
	assert.Equal(t, ldcFinalStats.Bytes, ldc1Stats.Bytes, "LDC1 bytes should match")
	assert.Equal(t, int64(0), ldc1Stats.Errors, "LDC1 should have no errors")

	require.Contains(t, runStats.GDCStats, 1, "GDC1 stats should be present")
	gdc1Stats := runStats.GDCStats[1]
	assert.Equal(t, int64(numEvents), gdc1Stats.Events, "GDC1 events should match")
	assert.Equal(t, gdcFinalStats.Bytes, gdc1Stats.Bytes, "GDC1 bytes should match")
	assert.Equal(t, int64(0), gdc1Stats.Errors, "GDC1 should have no errors")

	t.Log("Successfully completed API control test with database verification")
}
