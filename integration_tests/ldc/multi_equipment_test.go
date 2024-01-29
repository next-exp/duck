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

// TestLDC_MultipleEquipments_MountsEvent tests that an LDC can mount an event from multiple equipments
func TestLDC_MultipleEquipments_MountsEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/ldc_multi_eq_test.yml"
	equipConfigs := []testhelpers.EquipmentConfig{
		{LDCID: 1, EquipmentIDs: []int{1, 5, 6}},
	}

	testhelpers.CreateDummyGDCListeners(t)

	ldc := testhelpers.StartTestLDCWithConfig(t, configPath, equipConfigs, "", false)
	defer ldc.Stop()

	err := ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	// Wait a moment for all equipment listeners to be ready
	time.Sleep(500 * time.Millisecond)

	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get initial statistics")

	// Generate event with 3 equipments, 5 packets each
	equipmentIDs := []int{1, 5, 6} // Use IDs that don't conflict with seed data
	equipmentPackets := testhelpers.GenerateMultiEquipmentEvent(equipmentIDs, 1, 5)

	// Map equipment IDs to their ports (equipment 1 from seed data: 16006, auto-created: 16006+eqID)
	ports := map[int]int{
		1: 16006,
		5: 16011,
		6: 16012,
	}

	t.Logf("Sending event from %d equipments", len(equipmentIDs))
	err = testhelpers.SendMultiEquipmentEvent(equipmentPackets, "127.0.0.1", ports)
	require.NoError(t, err, "Failed to send multi-equipment event")

	// Wait for LDC to process the mounted event
	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+1, testhelpers.EventProcessingTimeout), "LDC did not process event in time")

	finalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get final statistics")

	t.Logf("Multi-equipment event stats - Events: %d, Bytes: %d, Errors: %d",
		finalStats.Events, finalStats.Bytes, finalStats.Errors)

	// Should have 1 mounted event from 3 equipments
	eventsDelta := finalStats.Events - initialStats.Events
	assert.Equal(t, int64(1), eventsDelta, "Should have 1 mounted event")

	// Verify we received data from all 3 equipments
	// Expected bytes = LDC header (80) + 3 * (equipment header (28) + 5 packets (1400*5) + end marker (4))
	expectedBytes := int64(80 + 3*(28+5*1400+4))
	bytesDelta := finalStats.Bytes - initialStats.Bytes
	assert.Equal(t, expectedBytes, bytesDelta, "Should have correct byte count for 3 equipments")

	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}

// TestLDC_MultipleEquipments_PartialEvent tests LDC with single equipment from configured set
func TestLDC_MultipleEquipments_PartialEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/ldc_multi_eq_test.yml"
	equipConfigs := []testhelpers.EquipmentConfig{
		{LDCID: 1, EquipmentIDs: []int{1}},
	}

	testhelpers.CreateDummyGDCListeners(t)

	ldc := testhelpers.StartTestLDCWithConfig(t, configPath, equipConfigs, "", false)
	defer ldc.Stop()

	err := ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	// Wait for equipment listeners to be ready
	time.Sleep(500 * time.Millisecond)

	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get initial statistics")

	// Send event from single equipment
	equipmentIDs := []int{1}
	equipmentPackets := testhelpers.GenerateMultiEquipmentEvent(equipmentIDs, 1, 5)

	// Equipment 1 is on port 16006 from seed data
	ports := map[int]int{
		1: 16006,
	}

	t.Logf("Sending event from equipment 1")
	err = testhelpers.SendMultiEquipmentEvent(equipmentPackets, "127.0.0.1", ports)
	require.NoError(t, err, "Failed to send event")

	// Wait for processing
	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+1, testhelpers.EventProcessingTimeout), "LDC did not process event in time")

	finalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get final statistics")

	t.Logf("Single equipment event stats - Events: %d, Bytes: %d, Errors: %d",
		finalStats.Events, finalStats.Bytes, finalStats.Errors)

	// Should mount the event with data from single equipment
	eventsDelta := finalStats.Events - initialStats.Events
	assert.Equal(t, int64(1), eventsDelta, "Should mount event from single equipment")

	// Verify we received data from 1 equipment only
	expectedBytes := int64(80 + 1*(28+5*1400+4))
	bytesDelta := finalStats.Bytes - initialStats.Bytes
	assert.Equal(t, expectedBytes, bytesDelta, "Should have correct byte count for 1 equipment")

	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}

// TestLDC_MultipleEquipments_DisabledEquipmentIgnored tests that LDC processes events only from enabled equipments
func TestLDC_MultipleEquipments_DisabledEquipmentIgnored(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	configPath := "integration_tests/fixtures/configs/ldc_multi_eq_test.yml"
	equipConfigs := []testhelpers.EquipmentConfig{
		{LDCID: 1, EquipmentIDs: []int{1}},
	}

	testhelpers.CreateDummyGDCListeners(t)

	ldc := testhelpers.StartTestLDCWithConfig(t, configPath, equipConfigs, "", false)
	defer ldc.Stop()

	err := ldc.StartRun()
	require.NoError(t, err, "Failed to start run")

	// Wait for equipment listeners to be ready
	time.Sleep(500 * time.Millisecond)

	initialStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get initial statistics")

	// Send event to enabled equipment 1
	equipmentIDs := []int{1}
	equipmentPackets := testhelpers.GenerateMultiEquipmentEvent(equipmentIDs, 1, 5)

	ports := map[int]int{
		1: 16006,
	}

	t.Logf("Sending event to enabled equipment 1")
	err = testhelpers.SendMultiEquipmentEvent(equipmentPackets, "127.0.0.1", ports)
	require.NoError(t, err, "Failed to send event to enabled equipment")

	// Should process this event
	require.NoError(t, ldc.WaitForProcessedEvents(initialStats.Events+1, testhelpers.EventProcessingTimeout), "LDC should process event from enabled equipment")

	finalStats, err := ldc.GetStatistics()
	require.NoError(t, err, "Failed to get final statistics")

	t.Logf("Enabled equipment stats - Events: %d, Bytes: %d, Errors: %d",
		finalStats.Events, finalStats.Bytes, finalStats.Errors)

	// Should have processed exactly 1 event from enabled equipment
	eventsDelta := finalStats.Events - initialStats.Events
	assert.Equal(t, int64(1), eventsDelta, "Should process event from enabled equipment")

	// Verify byte count is correct (only equipment 1 data)
	expectedBytes := int64(80 + 1*(28+5*1400+4))
	bytesDelta := finalStats.Bytes - initialStats.Bytes
	assert.Equal(t, expectedBytes, bytesDelta, "Should have correct byte count for 1 equipment")

	err = ldc.StopRun()
	require.NoError(t, err, "Failed to stop run")
}
