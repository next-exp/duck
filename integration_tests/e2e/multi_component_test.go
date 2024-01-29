//go:build integration
// +build integration

package e2e_test

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMultiComponentEnvironment(t *testing.T, gdc1OutputDir, gdc2OutputDir string) {
	dbConfig := testhelpers.GetSharedDBConfig()
	db, err := duck.ParseDatabaseConfiguration(dbConfig)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("UPDATE gdcs SET enabled=true, datapath=?", gdc1OutputDir)
	require.NoError(t, err)
	_, err = db.Exec("UPDATE gdcs SET enabled=true, datapath=? WHERE id=2", gdc2OutputDir)
	require.NoError(t, err)

	_, err = db.Exec("UPDATE ldcs SET enabled=true")
	require.NoError(t, err)

	_, err = db.Exec("UPDATE equipments SET enabled=true")
	require.NoError(t, err)
}

func generateMultiEquipmentPacketsWithPattern(equipmentIDs []int, eventID int, numPackets int) map[int][][]byte {
	result := make(map[int][][]byte)

	for _, eqID := range equipmentIDs {
		packets := make([][]byte, 0, numPackets+1)

		for i := 0; i < numPackets; i++ {
			packet := make([]byte, testhelpers.UDPDataPacketSize)

			binary.BigEndian.PutUint32(packet[0:4], uint32(i))
			binary.LittleEndian.PutUint32(packet[4:8], uint32(eqID))
			binary.LittleEndian.PutUint32(packet[8:12], uint32(eventID))

			for j := 12; j < testhelpers.UDPDataPacketSize; j++ {
				packet[j] = byte((eqID + i + j) % 256)
			}
			packets = append(packets, packet)
		}

		endMarker := make([]byte, 4)
		binary.LittleEndian.PutUint32(endMarker, 0xfafafafa)
		packets = append(packets, endMarker)

		result[eqID] = packets
	}

	return result
}

func sendEventsFromAllEquipments(t *testing.T, equipmentPorts map[int]int, equipmentIDs []int, numEvents int) {
	for eventID := 1; eventID <= numEvents; eventID++ {
		packets := generateMultiEquipmentPacketsWithPattern(equipmentIDs, eventID, testhelpers.DefaultTestPacketCount)

		err := testhelpers.SendMultiEquipmentEvent(packets, "127.0.0.1", equipmentPorts)
		require.NoError(t, err, "Failed to send event %d from all equipments", eventID)

		t.Logf("Sent event %d from equipments %v", eventID, equipmentIDs)
		time.Sleep(50 * time.Millisecond)
	}
}

func TestE2E_MultiComponent_DataIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/e2e_test.yml"

	tempDir := t.TempDir()
	gdc1OutputDir := filepath.Join(tempDir, "gdc1_output")
	gdc2OutputDir := filepath.Join(tempDir, "gdc2_output")
	err := os.MkdirAll(gdc1OutputDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(gdc2OutputDir, 0755)
	require.NoError(t, err)

	t.Logf("GDC1 output directory: %s", gdc1OutputDir)
	t.Logf("GDC2 output directory: %s", gdc2OutputDir)

	setupMultiComponentEnvironment(t, gdc1OutputDir, gdc2OutputDir)

	gdc1 := testhelpers.StartTestGDC(t, configPath, "test_gdc1")
	defer gdc1.Stop()

	gdc2 := testhelpers.StartTestGDC(t, configPath, "test_gdc2")
	defer gdc2.Stop()

	ldc1 := testhelpers.StartTestLDCWithConfig(t, configPath, []testhelpers.EquipmentConfig{
		{LDCID: 1, EquipmentIDs: []int{1, 2}},
	}, "", true)
	defer ldc1.Stop()

	ldc2 := testhelpers.StartTestLDCWithConfig(t, configPath, []testhelpers.EquipmentConfig{
		{LDCID: 2, EquipmentIDs: []int{3, 4}},
	}, "test_ldc2", true)
	defer ldc2.Stop()

	api := testhelpers.StartTestAPI(t, configPath)
	defer api.Stop()

	ctx := context.Background()
	err = api.StartRun(ctx)
	require.NoError(t, err, "Failed to start run via API")

	runNumber, err := api.GetRunNumber(ctx)
	require.NoError(t, err, "Failed to get run number")
	t.Logf("Run number: %d", runNumber)

	require.NoError(t, gdc1.WaitForState("RUNNING", 15*time.Second), "GDC1 should be RUNNING")
	require.NoError(t, gdc2.WaitForState("RUNNING", 15*time.Second), "GDC2 should be RUNNING")
	require.NoError(t, ldc1.WaitForState("RUNNING", 15*time.Second), "LDC1 should be RUNNING")
	require.NoError(t, ldc2.WaitForState("RUNNING", 15*time.Second), "LDC2 should be RUNNING")

	ldc1InitialStats, err := ldc1.GetStatistics()
	require.NoError(t, err)
	ldc2InitialStats, err := ldc2.GetStatistics()
	require.NoError(t, err)

	equipmentPorts := map[int]int{
		1: 16006,
		2: 16007,
		3: 16008,
		4: 16009,
	}
	allEquipmentIDs := []int{1, 2, 3, 4}

	numEvents := 6
	sendEventsFromAllEquipments(t, equipmentPorts, allEquipmentIDs, numEvents)

	require.NoError(t, ldc1.WaitForProcessedEvents(ldc1InitialStats.Events+int64(numEvents), 20*time.Second),
		"LDC1 did not process all events in time")
	require.NoError(t, ldc2.WaitForProcessedEvents(ldc2InitialStats.Events+int64(numEvents), 20*time.Second),
		"LDC2 did not process all events in time")

	time.Sleep(3 * time.Second)

	ldc1FinalStats, err := ldc1.GetStatistics()
	require.NoError(t, err)
	ldc2FinalStats, err := ldc2.GetStatistics()
	require.NoError(t, err)
	gdc1FinalStats, err := gdc1.GetStatistics()
	require.NoError(t, err)
	gdc2FinalStats, err := gdc2.GetStatistics()
	require.NoError(t, err)

	t.Logf("Final Stats:")
	t.Logf("  LDC1: Events=%d, Bytes=%d, Errors=%d", ldc1FinalStats.Events, ldc1FinalStats.Bytes, ldc1FinalStats.Errors)
	t.Logf("  LDC2: Events=%d, Bytes=%d, Errors=%d", ldc2FinalStats.Events, ldc2FinalStats.Bytes, ldc2FinalStats.Errors)
	t.Logf("  GDC1: Events=%d, Bytes=%d, Errors=%d", gdc1FinalStats.Events, gdc1FinalStats.Bytes, gdc1FinalStats.Errors)
	t.Logf("  GDC2: Events=%d, Bytes=%d, Errors=%d", gdc2FinalStats.Events, gdc2FinalStats.Bytes, gdc2FinalStats.Errors)

	err = api.StopRun(ctx)
	require.NoError(t, err, "Failed to stop run via API")

	t.Log("Verifying run metadata in database...")
	runStats := testhelpers.GetRunStatistics(t, int(runNumber))
	testhelpers.VerifyRunTimestamps(t, runStats.Run)

	t.Logf("Run %d statistics from database:", runNumber)
	for ldcID, stats := range runStats.LDCStats {
		t.Logf("  LDC%d: Events=%d, Bytes=%d, Errors=%d", ldcID, stats.Events, stats.Bytes, stats.Errors)
	}
	for gdcID, stats := range runStats.GDCStats {
		t.Logf("  GDC%d: Events=%d, Bytes=%d, Errors=%d", gdcID, stats.Events, stats.Bytes, stats.Errors)
	}

	require.Contains(t, runStats.LDCStats, 1, "LDC1 stats should be present")
	assert.Equal(t, int64(numEvents), runStats.LDCStats[1].Events, "LDC1 events should match")
	assert.Equal(t, ldc1FinalStats.Bytes, runStats.LDCStats[1].Bytes, "LDC1 bytes should match")
	assert.Equal(t, int64(0), runStats.LDCStats[1].Errors, "LDC1 should have no errors")

	require.Contains(t, runStats.LDCStats, 2, "LDC2 stats should be present")
	assert.Equal(t, int64(numEvents), runStats.LDCStats[2].Events, "LDC2 events should match")
	assert.Equal(t, ldc2FinalStats.Bytes, runStats.LDCStats[2].Bytes, "LDC2 bytes should match")
	assert.Equal(t, int64(0), runStats.LDCStats[2].Errors, "LDC2 should have no errors")

	require.Contains(t, runStats.GDCStats, 1, "GDC1 stats should be present")
	assert.Equal(t, int64(numEvents/2), runStats.GDCStats[1].Events, "GDC1 events should match (round-robin)")
	assert.Equal(t, gdc1FinalStats.Bytes, runStats.GDCStats[1].Bytes, "GDC1 bytes should match")
	assert.Equal(t, int64(0), runStats.GDCStats[1].Errors, "GDC1 should have no errors")

	require.Contains(t, runStats.GDCStats, 2, "GDC2 stats should be present")
	assert.Equal(t, int64(numEvents/2), runStats.GDCStats[2].Events, "GDC2 events should match (round-robin)")
	assert.Equal(t, gdc2FinalStats.Bytes, runStats.GDCStats[2].Bytes, "GDC2 bytes should match")
	assert.Equal(t, int64(0), runStats.GDCStats[2].Errors, "GDC2 should have no errors")

	require.NoError(t, ldc1.WaitForState("INITIALIZED", testhelpers.StateChangeTimeout))
	require.NoError(t, ldc2.WaitForState("INITIALIZED", testhelpers.StateChangeTimeout))
	require.NoError(t, gdc1.WaitForState("INITIALIZED", testhelpers.StateChangeTimeout))
	require.NoError(t, gdc2.WaitForState("INITIALIZED", testhelpers.StateChangeTimeout))

	gdc1Files, err := testhelpers.FindBinaryFiles(gdc1OutputDir)
	require.NoError(t, err)
	gdc2Files, err := testhelpers.FindBinaryFiles(gdc2OutputDir)
	require.NoError(t, err)

	t.Logf("GDC1 binary files: %v", gdc1Files)
	t.Logf("GDC2 binary files: %v", gdc2Files)

	require.NotEmpty(t, gdc1Files, "GDC1 should have created binary files")
	require.NotEmpty(t, gdc2Files, "GDC2 should have created binary files")

	gdc1Events, err := testhelpers.ReadBinaryEventsFromFile(gdc1Files[0])
	require.NoError(t, err, "Failed to read GDC1 binary file")
	gdc2Events, err := testhelpers.ReadBinaryEventsFromFile(gdc2Files[0])
	require.NoError(t, err, "Failed to read GDC2 binary file")

	t.Logf("GDC1 events: %d", len(gdc1Events))
	t.Logf("GDC2 events: %d", len(gdc2Events))

	assert.Equal(t, numEvents/2, len(gdc1Events), "GDC1 should have half the events (round-robin)")
	assert.Equal(t, numEvents/2, len(gdc2Events), "GDC2 should have half the events (round-robin)")

	gdc1EventIDs := testhelpers.ExtractEventIDs(gdc1Events)
	gdc2EventIDs := testhelpers.ExtractEventIDs(gdc2Events)
	t.Logf("GDC1 event IDs: %v", gdc1EventIDs)
	t.Logf("GDC2 event IDs: %v", gdc2EventIDs)

	expectedGDC1IDs := []uint32{1, 3, 5}
	expectedGDC2IDs := []uint32{2, 4, 6}
	assert.ElementsMatch(t, expectedGDC1IDs, gdc1EventIDs, "GDC1 should have odd-numbered events")
	assert.ElementsMatch(t, expectedGDC2IDs, gdc2EventIDs, "GDC2 should have even-numbered events")

	expectedEquipIDs := []uint32{1, 2, 3, 4}
	expectedLDCIDs := []uint32{1, 2}

	for _, event := range gdc1Events {
		err := testhelpers.VerifyEventHasAllLDCs(event, expectedLDCIDs)
		assert.NoError(t, err, "Event %d should have data from all LDCs", event.EventID)

		err = testhelpers.VerifyEventHasAllEquipments(event, expectedEquipIDs)
		assert.NoError(t, err, "Event %d should have data from all equipments", event.EventID)

		for _, ldc := range event.LDCData {
			for _, equip := range ldc.EquipmentData {
				err := testhelpers.VerifyEquipmentDataPattern(equip, event.EventID)
				assert.NoError(t, err, "Event %d LDC %d Equipment %d data pattern verification failed",
					event.EventID, ldc.LDCID, equip.EquipmentID)
			}
		}
	}

	for _, event := range gdc2Events {
		err := testhelpers.VerifyEventHasAllLDCs(event, expectedLDCIDs)
		assert.NoError(t, err, "Event %d should have data from all LDCs", event.EventID)

		err = testhelpers.VerifyEventHasAllEquipments(event, expectedEquipIDs)
		assert.NoError(t, err, "Event %d should have data from all equipments", event.EventID)

		for _, ldc := range event.LDCData {
			for _, equip := range ldc.EquipmentData {
				err := testhelpers.VerifyEquipmentDataPattern(equip, event.EventID)
				assert.NoError(t, err, "Event %d LDC %d Equipment %d data pattern verification failed",
					event.EventID, ldc.LDCID, equip.EquipmentID)
			}
		}
	}

	totalEvents := len(gdc1Events) + len(gdc2Events)
	assert.Equal(t, numEvents, totalEvents, "Total events across both GDCs should equal sent events")

	t.Log("Successfully completed multi-component data integrity test")
}
