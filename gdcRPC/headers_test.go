package main

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jmbenlloch/next_duck/gdcRPC/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

func TestAddGDCHeader_CreatesCorrectHeader(t *testing.T) {
	// Setup
	ldcData := make(map[int][]byte)
	ldcData[1] = testhelpers.CreateLDCEventData(100, 1, 200)
	ldcData[2] = testhelpers.CreateLDCEventData(100, 2, 300)
	eventID := 100
	gdcID := 5
	runNumber := 123

	// Act
	result := addGDCHeader(ldcData, eventID, gdcID, runNumber)

	// Assert
	require.NotNil(t, result)
	require.GreaterOrEqual(t, len(result), 80, "GDC header should be at least 80 bytes")

	// Check event size (first 4 bytes)
	eventSize := binary.LittleEndian.Uint32(result[0:4])
	expectedSize := len(ldcData[1]) + len(ldcData[2]) + 80
	assert.Equal(t, uint32(expectedSize), eventSize, "Event size should include header + all LDC data")

	// Check magic number (bytes 4-8)
	magic := binary.LittleEndian.Uint32(result[4:8])
	assert.Equal(t, uint32(duck.EVENT_MAGIC_NUMBER), magic, "Magic number should match")

	// Check header size (bytes 8-12)
	headerSize := binary.LittleEndian.Uint32(result[8:12])
	assert.Equal(t, uint32(80), headerSize, "Header size should be 80 bytes")

	// Check version (bytes 12-16)
	version := binary.LittleEndian.Uint32(result[12:16])
	assert.Equal(t, uint32(duck.EVENT_CURRENT_VERSION), version, "Version should match")

	// Check event type (bytes 16-20)
	eventType := binary.LittleEndian.Uint32(result[16:20])
	assert.Equal(t, uint32(duck.PHYSICS_EVENT), eventType, "Event type should be PHYSICS_EVENT")

	// Check run number (bytes 20-24)
	runNb := binary.LittleEndian.Uint32(result[20:24])
	assert.Equal(t, uint32(runNumber), runNb, "Run number should match")

	// Check event ID (bytes 24-28)
	evtID := binary.LittleEndian.Uint32(result[24:28])
	assert.Equal(t, uint32(eventID), evtID, "Event ID should match")

	// Check type attribute (bytes 60-64) - should have ORIGINAL_EVENT | SUPER_EVENT
	typeAttr := binary.LittleEndian.Uint32(result[60:64])
	expectedAttr := uint32(duck.ORIGINAL_EVENT | duck.SUPER_EVENT)
	assert.Equal(t, expectedAttr, typeAttr, "Type attribute should be ORIGINAL_EVENT | SUPER_EVENT")

	// Check LDC ID (bytes 64-68) - should be LDC_VOID for GDC
	ldcID := binary.LittleEndian.Uint32(result[64:68])
	assert.Equal(t, uint32(duck.LDC_VOID), ldcID, "LDC ID should be LDC_VOID")

	// Check GDC ID (bytes 68-72)
	gdc := binary.LittleEndian.Uint32(result[68:72])
	assert.Equal(t, uint32(gdcID), gdc, "GDC ID should match")

	// Check that timestamp fields are present (bytes 72-80)
	timestampSec := binary.LittleEndian.Uint32(result[72:76])
	timestampUsec := binary.LittleEndian.Uint32(result[76:80])
	assert.Greater(t, timestampSec, uint32(0), "Timestamp seconds should be set")
	assert.GreaterOrEqual(t, timestampUsec, uint32(0), "Timestamp microseconds should be set")
	assert.Less(t, timestampUsec, uint32(1000000), "Timestamp microseconds should be < 1 million")

	// Verify all LDC data is present after the header (order doesn't matter)
	dataSection := result[80:]
	allLDCData := [][]byte{ldcData[1], ldcData[2]}

	// Check that each LDC's data appears somewhere in the result
	for _, ldc := range allLDCData {
		found := false
		// Search for this LDC data in the result
		for i := 0; i <= len(dataSection)-len(ldc); i++ {
			if string(dataSection[i:i+len(ldc)]) == string(ldc) {
				found = true
				break
			}
		}
		assert.True(t, found, "LDC data should be present in result")
	}

	// Verify total data section size matches sum of all LDC data
	assert.Equal(t, expectedSize-80, len(dataSection), "Data section size should match sum of LDC data")
}

func TestAddGDCHeader_SingleLDC(t *testing.T) {
	// Setup - test with single LDC
	ldcData := make(map[int][]byte)
	ldcData[1] = testhelpers.CreateLDCEventData(50, 1, 100)
	eventID := 50
	gdcID := 1
	runNumber := 456

	// Act
	result := addGDCHeader(ldcData, eventID, gdcID, runNumber)

	// Assert
	require.NotNil(t, result)
	eventSize := binary.LittleEndian.Uint32(result[0:4])
	expectedSize := len(ldcData[1]) + 80
	assert.Equal(t, uint32(expectedSize), eventSize, "Event size should include header + LDC data")

	// Verify run number and event ID
	runNb := binary.LittleEndian.Uint32(result[20:24])
	assert.Equal(t, uint32(runNumber), runNb)
	evtID := binary.LittleEndian.Uint32(result[24:28])
	assert.Equal(t, uint32(eventID), evtID)

	// Verify LDC data is concatenated after the header
	position := 80
	assert.Equal(t, ldcData[1], result[position:position+len(ldcData[1])], "LDC data should match")
}

func TestAddGDCHeader_MultipleLDCs(t *testing.T) {
	// Setup - test with multiple LDCs
	ldcData := make(map[int][]byte)
	ldcData[1] = testhelpers.CreateLDCEventData(200, 1, 150)
	ldcData[2] = testhelpers.CreateLDCEventData(200, 2, 200)
	ldcData[3] = testhelpers.CreateLDCEventData(200, 3, 250)
	eventID := 200
	gdcID := 3
	runNumber := 789

	// Act
	result := addGDCHeader(ldcData, eventID, gdcID, runNumber)

	// Assert
	require.NotNil(t, result)
	eventSize := binary.LittleEndian.Uint32(result[0:4])
	expectedSize := len(ldcData[1]) + len(ldcData[2]) + len(ldcData[3]) + 80
	assert.Equal(t, uint32(expectedSize), eventSize, "Event size should include header + all LDC data")

	// Verify total result size matches expected
	assert.Equal(t, expectedSize, len(result), "Result size should match expected size")

	// Verify all LDC data is present after the header (order doesn't matter)
	dataSection := result[80:]
	allLDCData := [][]byte{ldcData[1], ldcData[2], ldcData[3]}

	// Check that each LDC's data appears somewhere in the result
	for _, ldc := range allLDCData {
		found := false
		// Search for this LDC data in the result
		for i := 0; i <= len(dataSection)-len(ldc); i++ {
			if string(dataSection[i:i+len(ldc)]) == string(ldc) {
				found = true
				break
			}
		}
		assert.True(t, found, "LDC data should be present in result")
	}

	// Verify total data section size matches sum of all LDC data
	assert.Equal(t, expectedSize-80, len(dataSection), "Data section size should match sum of LDC data")
}

func TestAddGDCHeader_EmptyLDCData(t *testing.T) {
	// Setup - test with empty LDC data map
	ldcData := make(map[int][]byte)
	eventID := 1
	gdcID := 1
	runNumber := 100

	// Act
	result := addGDCHeader(ldcData, eventID, gdcID, runNumber)

	// Assert
	require.NotNil(t, result)
	// Should still have header
	assert.GreaterOrEqual(t, len(result), 80)
	eventSize := binary.LittleEndian.Uint32(result[0:4])
	assert.Equal(t, uint32(80), eventSize, "Event size should be just the header when no LDC data")
}

func TestAddGDCHeader_DataIntegrity(t *testing.T) {
	// Setup - verify that LDC data is preserved correctly
	ldcData := make(map[int][]byte)

	// Create proper LDC event data with headers
	ldcData[1] = testhelpers.CreateLDCEventData(42, 1, 100)
	ldcData[2] = testhelpers.CreateLDCEventData(42, 2, 200)

	// Act
	result := addGDCHeader(ldcData, 42, 1, 100)

	// Assert
	require.NotNil(t, result)

	// Verify event size field
	eventSize := binary.LittleEndian.Uint32(result[0:4])
	expectedSize := len(ldcData[1]) + len(ldcData[2]) + 80
	assert.Equal(t, uint32(expectedSize), eventSize, "Event size should match header + all LDC data")

	// Verify total result size matches expected
	totalLDCSize := len(ldcData[1]) + len(ldcData[2])
	assert.Equal(t, 80+totalLDCSize, len(result), "Total size should be header + all LDC data")

	// The payload starts at byte 80
	// Verify all LDC data is present after the header (order doesn't matter)
	dataSection := result[80:]
	allLDCData := [][]byte{ldcData[1], ldcData[2]}

	// Check that each LDC's data appears somewhere in the result
	for _, ldc := range allLDCData {
		found := false
		// Search for this LDC data in the result
		for i := 0; i <= len(dataSection)-len(ldc); i++ {
			if string(dataSection[i:i+len(ldc)]) == string(ldc) {
				found = true
				break
			}
		}
		assert.True(t, found, "LDC data should be present in result")
	}

	// Verify data integrity - each byte should be preserved
	for _, ldc := range allLDCData {
		found := false
		for i := 0; i <= len(dataSection)-len(ldc); i++ {
			match := true
			for j := 0; j < len(ldc); j++ {
				if dataSection[i+j] != ldc[j] {
					match = false
					break
				}
			}
			if match {
				found = true
				break
			}
		}
		assert.True(t, found, "Each byte of LDC data should be preserved correctly")
	}
}

func TestAddGDCHeader_EventIDZero(t *testing.T) {
	// Setup - test edge case with event ID = 0
	ldcData := make(map[int][]byte)
	ldcData[1] = testhelpers.CreateLDCEventData(0, 1, 50)
	eventID := 0
	gdcID := 2
	runNumber := 999

	// Act
	result := addGDCHeader(ldcData, eventID, gdcID, runNumber)

	// Assert
	require.NotNil(t, result)

	// Verify event size field
	eventSize := binary.LittleEndian.Uint32(result[0:4])
	expectedSize := len(ldcData[1]) + 80
	assert.Equal(t, uint32(expectedSize), eventSize, "Event size should match header + LDC data")

	// Verify event ID
	evtID := binary.LittleEndian.Uint32(result[24:28])
	assert.Equal(t, uint32(0), evtID, "Event ID 0 should be preserved")

	// Verify LDC data is concatenated after the header
	position := 80
	assert.Equal(t, ldcData[1], result[position:position+len(ldcData[1])], "LDC data should match")
}

func TestAddGDCHeader_HighEventID(t *testing.T) {
	// Setup - test with high event ID value
	ldcData := make(map[int][]byte)
	ldcData[1] = testhelpers.CreateLDCEventData(999999, 1, 100)
	eventID := 999999
	gdcID := 1
	runNumber := 100

	// Act
	result := addGDCHeader(ldcData, eventID, gdcID, runNumber)

	// Assert
	require.NotNil(t, result)

	// Verify event size field
	eventSize := binary.LittleEndian.Uint32(result[0:4])
	expectedSize := len(ldcData[1]) + 80
	assert.Equal(t, uint32(expectedSize), eventSize, "Event size should match header + LDC data")

	// Verify event ID
	evtID := binary.LittleEndian.Uint32(result[24:28])
	assert.Equal(t, uint32(eventID), evtID, "High event ID should be preserved")

	// Verify LDC data is concatenated after the header
	position := 80
	assert.Equal(t, ldcData[1], result[position:position+len(ldcData[1])], "LDC data should match")
}
