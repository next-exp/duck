//go:build integration
// +build integration

package e2e_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sendValidEventsFromAllEquipments(t *testing.T, equipmentPorts map[int]int, equipmentIDs []int, startEventID, numEvents int) {
	for eventID := startEventID; eventID < startEventID+numEvents; eventID++ {
		packets := generateMultiEquipmentPacketsWithPattern(equipmentIDs, eventID, testhelpers.DefaultTestPacketCount)
		for _, eqID := range equipmentIDs {
			sender, err := testhelpers.CreateUDPSender("127.0.0.1", equipmentPorts[eqID])
			require.NoError(t, err, "Failed to create UDP sender for equipment %d", eqID)
			err = sender.SendEvent(packets[eqID])
			sender.Close()
			require.NoError(t, err, "Failed to send event %d from equipment %d", eventID, eqID)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Logf("Sent %d valid events from equipments %v", numEvents, equipmentIDs)
}

func sendEventWithSequenceError(t *testing.T, equipmentPort, equipmentID, eventID int) {
	packets := testhelpers.GenerateEventWithSequenceSkip(equipmentID, eventID, testhelpers.DefaultTestPacketCount, 1)

	sender, err := testhelpers.CreateUDPSender("127.0.0.1", equipmentPort)
	require.NoError(t, err, "Failed to create UDP sender")
	defer sender.Close()

	err = sender.SendEvent(packets)
	if err != nil {
		t.Logf("SendEvent returned error (expected if LDC stopped): %v", err)
	} else {
		t.Logf("Sent event %d with sequence error from equipment %d on port %d", eventID, equipmentID, equipmentPort)
	}
}

func TestE2E_ErrorPropagation_Centrifuge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/e2e_test.yml"

	centrifugePool, centrifugeResource, centrifugeConfig := testhelpers.SetupCentrifugeForSystemTest(t)
	defer testhelpers.CleanupDocker(centrifugePool, centrifugeResource)

	t.Logf("Centrifuge started at %s:%d", centrifugeConfig.Host, centrifugeConfig.Port)

	tracker := testhelpers.NewCentrifugeSystemTracker(t, centrifugeConfig)
	defer tracker.Close()

	tempDir := t.TempDir()
	gdc1OutputDir := filepath.Join(tempDir, "gdc1_output")
	gdc2OutputDir := filepath.Join(tempDir, "gdc2_output")
	err := os.MkdirAll(gdc1OutputDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(gdc2OutputDir, 0755)
	require.NoError(t, err)

	setupMultiComponentEnvironment(t, gdc1OutputDir, gdc2OutputDir)

	gdc1 := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc1.Stop()

	gdc2 := testhelpers.StartTestGDC(t, configPath, "test_gdc2")
	defer gdc2.Stop()

	ldc1 := testhelpers.StartTestLDCWithCentrifuge(t, configPath, []testhelpers.EquipmentConfig{
		{LDCID: 1, EquipmentIDs: []int{1, 2}},
	}, "", true, centrifugeConfig)
	defer ldc1.Stop()

	ldc2 := testhelpers.StartTestLDCWithCentrifuge(t, configPath, []testhelpers.EquipmentConfig{
		{LDCID: 2, EquipmentIDs: []int{3, 4}},
	}, "test_ldc2", true, centrifugeConfig)
	defer ldc2.Stop()

	api := testhelpers.StartTestAPIWithCentrifuge(t, configPath, centrifugeConfig)
	defer api.Stop()

	ctx := context.Background()
	err = api.StartRun(ctx)
	require.NoError(t, err, "Failed to start run via API")

	runNumber, err := api.GetRunNumber(ctx)
	require.NoError(t, err, "Failed to get run number")
	t.Logf("Run number: %d", runNumber)

	require.NoError(t, gdc1.WaitForState("RUNNING", 10*time.Second), "GDC1 should be RUNNING")
	require.NoError(t, gdc2.WaitForState("RUNNING", 10*time.Second), "GDC2 should be RUNNING")
	require.NoError(t, ldc1.WaitForState("RUNNING", 10*time.Second), "LDC1 should be RUNNING")
	require.NoError(t, ldc2.WaitForState("RUNNING", 10*time.Second), "LDC2 should be RUNNING")

	equipmentPorts := map[int]int{
		1: 16006,
		2: 16007,
		3: 16008,
		4: 16009,
	}
	allEquipmentIDs := []int{1, 2, 3, 4}

	ldc1InitialStats, err := ldc1.GetStatistics()
	require.NoError(t, err)
	ldc2InitialStats, err := ldc2.GetStatistics()
	require.NoError(t, err)

	sendValidEventsFromAllEquipments(t, equipmentPorts, allEquipmentIDs, 1, 2)

	time.Sleep(200 * time.Millisecond)

	t.Log("Sending event with sequence error from Equipment 1...")
	sendEventWithSequenceError(t, equipmentPorts[1], 1, 3)

	require.NoError(t, ldc1.WaitForProcessedEvents(ldc1InitialStats.Events+2, 20*time.Second),
		"LDC1 did not process valid events")
	require.NoError(t, ldc2.WaitForProcessedEvents(ldc2InitialStats.Events+2, 20*time.Second),
		"LDC2 did not process valid events")

	t.Log("Valid events processed successfully")

	t.Log("Waiting for error message via Centrifuge...")
	errMsg, err := tracker.WaitForStopProcessesMessage(30 * time.Second)
	require.NoError(t, err, "Did not receive error message via Centrifuge")

	t.Logf("Received error message: type=%s, host=%s, stop_processes=%v, value=%.200s",
		errMsg.Type, errMsg.Host, errMsg.StopProcesses, errMsg.Value)

	assert.Equal(t, duck.MessageError, errMsg.Type, "Message type should be error")
	assert.True(t, errMsg.StopProcesses, "Error message should have StopProcesses=true")
	assert.Contains(t, errMsg.Value, "Packet mismatch", "Error should indicate sequence mismatch")

	t.Log("Waiting for all processes to stop via API forceStop...")

	time.Sleep(5 * time.Second)

	assert.NoError(t, ldc1.WaitForState("INITIALIZED", 15*time.Second), "LDC1 should return to INITIALIZED")
	assert.NoError(t, ldc2.WaitForState("INITIALIZED", 15*time.Second), "LDC2 should return to INITIALIZED")
	assert.NoError(t, gdc1.WaitForState("INITIALIZED", 15*time.Second), "GDC1 should return to INITIALIZED")
	assert.NoError(t, gdc2.WaitForState("INITIALIZED", 15*time.Second), "GDC2 should return to INITIALIZED")

	t.Log("All processes stopped successfully after error propagation")

	t.Log("Verifying run metadata in database (forceStop should set timestamps but NOT store statistics)...")
	testhelpers.VerifyRunTimestampsSetWithoutStats(t, int(runNumber))

	t.Log("Successfully completed Centrifuge error propagation test")
}
