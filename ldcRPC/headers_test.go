package main

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jmbenlloch/next_duck/ldcRPC/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

func TestAddEquipmentHeader_CreatesCorrectHeader(t *testing.T) {
	// Setup
	logger := testhelpers.SetupTest()
	equipment := testhelpers.NewTestEquipment(5, 22)
	subevent := make([]byte, 100)
	for i := range subevent {
		subevent[i] = byte(i)
	}

	// Act
	result := addEquipmentHeader(logger, subevent, equipment)

	// Assert
	// Header size is 7*4 = 28 bytes
	expectedSize := len(subevent) + 28
	assert.Equal(t, expectedSize, len(result))

	// Check equipment size field (first 4 bytes)
	equipmentSize := binary.LittleEndian.Uint32(result[0:4])
	assert.Equal(t, uint32(expectedSize), equipmentSize)

	// Check equipment type (bytes 4-8)
	equipmentType := binary.LittleEndian.Uint32(result[4:8])
	assert.Equal(t, uint32(22), equipmentType)

	// Check equipment ID (bytes 8-12)
	equipmentID := binary.LittleEndian.Uint32(result[8:12])
	assert.Equal(t, uint32(5), equipmentID)

	// Check basic element size (bytes 24-28)
	basicElementSize := binary.LittleEndian.Uint32(result[24:28])
	assert.Equal(t, uint32(4), basicElementSize)
}

func TestAddEquipmentHeader_EmptySubevent(t *testing.T) {
	// Setup
	logger := testhelpers.SetupTest()
	equipment := testhelpers.NewTestEquipment(1, 10)
	subevent := []byte{}

	// Act
	result := addEquipmentHeader(logger, subevent, equipment)

	// Assert
	// Should have just the header (28 bytes)
	assert.Equal(t, 28, len(result))

	equipmentSize := binary.LittleEndian.Uint32(result[0:4])
	assert.Equal(t, uint32(28), equipmentSize)
}

func TestAddLDCHeader_CreatesCorrectHeader(t *testing.T) {
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(3, "testldc", 2)
	runNumber := 123
	eventID := 42

	// Create equipment data
	equipmentsData := make(map[int][]byte)
	equipmentsData[1] = []byte{0x01, 0x02, 0x03, 0x04}
	equipmentsData[2] = []byte{0x05, 0x06, 0x07, 0x08}

	// Act
	result := addLDCHeader(equipmentsData, eventID, ldcConfig, runNumber)

	// Assert
	// Header is 80 bytes (20*4), plus 8 bytes of data
	expectedSize := 80 + 8
	assert.Equal(t, expectedSize, len(result))

	// Check event size (first 4 bytes)
	eventSize := binary.LittleEndian.Uint32(result[0:4])
	assert.Equal(t, uint32(expectedSize), eventSize)

	// Check magic number (bytes 4-8)
	magicNumber := binary.LittleEndian.Uint32(result[4:8])
	assert.Equal(t, uint32(duck.EVENT_MAGIC_NUMBER), magicNumber)

	// Check header size (bytes 8-12)
	headerSize := binary.LittleEndian.Uint32(result[8:12])
	assert.Equal(t, uint32(80), headerSize)

	// Check event version (bytes 12-16)
	version := binary.LittleEndian.Uint32(result[12:16])
	assert.Equal(t, uint32(duck.EVENT_CURRENT_VERSION), version)

	// Check event type (bytes 16-20)
	eventType := binary.LittleEndian.Uint32(result[16:20])
	assert.Equal(t, uint32(duck.PHYSICS_EVENT), eventType)

	// Check run number (bytes 20-24)
	runNb := binary.LittleEndian.Uint32(result[20:24])
	assert.Equal(t, uint32(123), runNb)

	// Check event ID (bytes 24-28)
	evtID := binary.LittleEndian.Uint32(result[24:28])
	assert.Equal(t, uint32(42), evtID)

	// Check LDC ID (bytes 64-68)
	ldcID := binary.LittleEndian.Uint32(result[64:68])
	assert.Equal(t, uint32(3), ldcID)

	// Check GDC ID (bytes 68-72) should be GDC_VOID
	gdcID := binary.LittleEndian.Uint32(result[68:72])
	assert.Equal(t, uint32(duck.GDC_VOID), gdcID)

	// Check that timestamp fields are set (non-zero)
	timestampSec := binary.LittleEndian.Uint32(result[72:76])
	assert.Greater(t, timestampSec, uint32(0))

	// Check type attribute (bytes 60-64) should be ORIGINAL_EVENT
	typeAttr := binary.LittleEndian.Uint32(result[60:64])
	assert.Equal(t, uint32(duck.ORIGINAL_EVENT), typeAttr)
}

func TestAddLDCHeader_MultipleEquipments(t *testing.T) {
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 3)
	runNumber := 100
	eventID := 1

	// Create equipment data with different sizes
	equipmentsData := make(map[int][]byte)
	equipmentsData[1] = make([]byte, 10)
	equipmentsData[2] = make([]byte, 20)
	equipmentsData[3] = make([]byte, 15)

	totalDataSize := 10 + 20 + 15

	// Act
	result := addLDCHeader(equipmentsData, eventID, ldcConfig, runNumber)

	// Assert
	expectedSize := 80 + totalDataSize
	assert.Equal(t, expectedSize, len(result))

	// Check event size matches
	eventSize := binary.LittleEndian.Uint32(result[0:4])
	assert.Equal(t, uint32(expectedSize), eventSize)
}

func TestAddLDCHeader_EmptyEquipmentData(t *testing.T) {
	// Setup
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 1)
	runNumber := 50
	eventID := 10

	equipmentsData := make(map[int][]byte)

	// Act
	result := addLDCHeader(equipmentsData, eventID, ldcConfig, runNumber)

	// Assert
	// Should have just the header (80 bytes)
	assert.Equal(t, 80, len(result))

	eventSize := binary.LittleEndian.Uint32(result[0:4])
	assert.Equal(t, uint32(80), eventSize)
}

func TestAddEquipmentHeader_DataEndiannessFlip(t *testing.T) {
	// Setup
	logger := testhelpers.SetupTest()
	equipment := testhelpers.NewTestEquipment(1, 22)

	// Create subevent with known pattern (4 bytes)
	subevent := []byte{0x01, 0x02, 0x03, 0x04}

	// Act
	result := addEquipmentHeader(logger, subevent, equipment)

	// Assert: Data should start at byte 28 and be flipped to big endian
	// Original: 0x01020304 (little endian) -> Should become big endian in output
	dataStart := result[28:32]

	// Original data as little endian uint32
	originalWord := binary.LittleEndian.Uint32(subevent)

	// Data in result should be big endian
	resultWord := binary.BigEndian.Uint32(dataStart)

	assert.Equal(t, originalWord, resultWord)
}

func TestAddLDCHeader_LargeEventID_BoundaryValue(t *testing.T) {
	// Setup - test with MaxUint32 event ID
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 1)
	runNumber := 100
	eventID := int(^uint32(0)) // MaxUint32 = 4294967295

	equipmentsData := make(map[int][]byte)
	equipmentsData[1] = []byte{0x01, 0x02, 0x03, 0x04}

	// Act
	result := addLDCHeader(equipmentsData, eventID, ldcConfig, runNumber)

	// Assert - verify event ID is stored correctly
	storedEventID := binary.LittleEndian.Uint32(result[24:28])
	assert.Equal(t, uint32(eventID), storedEventID)

	// Also test with boundary-1 value
	eventID2 := int(^uint32(0) - 1)
	result2 := addLDCHeader(equipmentsData, eventID2, ldcConfig, runNumber)
	storedEventID2 := binary.LittleEndian.Uint32(result2[24:28])
	assert.Equal(t, uint32(eventID2), storedEventID2)
}

func TestAddEquipmentHeader_LargeData_HandlesCorrectly(t *testing.T) {
	// Setup - create MB-sized equipment data
	logger := testhelpers.SetupTest()
	equipment := testhelpers.NewTestEquipment(1, 22)

	// Create 1MB subevent data
	subeventSize := 1024 * 1024 // 1 MB
	subevent := make([]byte, subeventSize)
	for i := range subevent {
		subevent[i] = byte(i % 256)
	}

	// Act
	result := addEquipmentHeader(logger, subevent, equipment)

	// Assert
	// Header size is 7*4 = 28 bytes
	expectedSize := subeventSize + 28
	assert.Equal(t, expectedSize, len(result))

	// Check equipment size field (first 4 bytes)
	equipmentSize := binary.LittleEndian.Uint32(result[0:4])
	assert.Equal(t, uint32(expectedSize), equipmentSize)

	// Verify data integrity - check first and last few bytes after header
	// Data starts at byte 28, and each 4-byte word is endian-flipped
	// First word: bytes 0,1,2,3 -> should be flipped to big endian
	if subeventSize >= 4 {
		originalFirstWord := binary.LittleEndian.Uint32(subevent[0:4])
		resultFirstWord := binary.BigEndian.Uint32(result[28:32])
		assert.Equal(t, originalFirstWord, resultFirstWord)
	}
}

func TestAddLDCHeader_LargeRunNumber_Boundary(t *testing.T) {
	// Setup - test with large run number
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 1)
	runNumber := int(^uint32(0)) // MaxUint32
	eventID := 1

	equipmentsData := make(map[int][]byte)
	equipmentsData[1] = []byte{0x01, 0x02, 0x03, 0x04}

	// Act
	result := addLDCHeader(equipmentsData, eventID, ldcConfig, runNumber)

	// Assert - verify run number is stored correctly
	storedRunNumber := binary.LittleEndian.Uint32(result[20:24])
	assert.Equal(t, uint32(runNumber), storedRunNumber)
}

func TestAddEquipmentHeader_AllEquipmentTypes(t *testing.T) {
	// Test various equipment types
	logger := testhelpers.SetupTest()
	equipmentTypes := []int{0, 1, 22, 100, 255, 65535}
	subevent := []byte{0x01, 0x02, 0x03, 0x04}

	for _, eqType := range equipmentTypes {
		t.Run("Type_"+string(rune('0'+eqType%10)), func(t *testing.T) {
			equipment := testhelpers.NewTestEquipment(1, eqType)

			// Act
			result := addEquipmentHeader(logger, subevent, equipment)

			// Assert - check equipment type
			storedType := binary.LittleEndian.Uint32(result[4:8])
			assert.Equal(t, uint32(eqType), storedType)
		})
	}
}

// TestAddLDCHeader_TableDriven_BoundaryValues tests boundary values for LDC header fields
func TestAddLDCHeader_TableDriven_BoundaryValues(t *testing.T) {
	tests := []struct {
		name       string
		eventID    int
		runNumber  int
		ldcID      int
		dataSize   int
		expectPass bool
	}{
		{
			name:       "Zero values",
			eventID:    0,
			runNumber:  0,
			ldcID:      0,
			dataSize:   4,
			expectPass: true,
		},
		{
			name:       "Typical values",
			eventID:    12345,
			runNumber:  100,
			ldcID:      1,
			dataSize:   100,
			expectPass: true,
		},
		{
			name:       "Max uint32 eventID",
			eventID:    int(^uint32(0)),
			runNumber:  100,
			ldcID:      1,
			dataSize:   4,
			expectPass: true,
		},
		{
			name:       "Max uint32 runNumber",
			eventID:    1,
			runNumber:  int(^uint32(0)),
			ldcID:      1,
			dataSize:   4,
			expectPass: true,
		},
		{
			name:       "Large LDC ID",
			eventID:    1,
			runNumber:  1,
			ldcID:      255,
			dataSize:   4,
			expectPass: true,
		},
		{
			name:       "Empty equipment data",
			eventID:    1,
			runNumber:  1,
			ldcID:      1,
			dataSize:   0,
			expectPass: true,
		},
		{
			name:       "Single byte data",
			eventID:    1,
			runNumber:  1,
			ldcID:      1,
			dataSize:   1,
			expectPass: true,
		},
		{
			name:       "Large data",
			eventID:    1,
			runNumber:  1,
			ldcID:      1,
			dataSize:   10000,
			expectPass: true,
		},
		{
			name:       "Event ID near max",
			eventID:    int(^uint32(0)) - 1,
			runNumber:  1,
			ldcID:      1,
			dataSize:   4,
			expectPass: true,
		},
		{
			name:       "Run number near max",
			eventID:    1,
			runNumber:  int(^uint32(0)) - 1,
			ldcID:      1,
			dataSize:   4,
			expectPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create LDC configuration with specified ID
			ldcConfig := duck.LDCConfiguration{
				ID:   tt.ldcID,
				Name: "testldc",
			}

			// Create equipment data
			equipmentsData := make(map[int][]byte)
			if tt.dataSize > 0 {
				data := make([]byte, tt.dataSize)
				for i := range data {
					data[i] = byte(i % 256)
				}
				equipmentsData[1] = data
			}

			// Act
			result := addLDCHeader(equipmentsData, tt.eventID, ldcConfig, tt.runNumber)

			// Assert
			if tt.expectPass {
				// Check event size
				eventSize := binary.LittleEndian.Uint32(result[0:4])
				expectedSize := uint32(80 + tt.dataSize)
				assert.Equal(t, expectedSize, eventSize, "Event size mismatch")

				// Check event ID
				storedEventID := binary.LittleEndian.Uint32(result[24:28])
				assert.Equal(t, uint32(tt.eventID), storedEventID, "Event ID mismatch")

				// Check run number
				storedRunNumber := binary.LittleEndian.Uint32(result[20:24])
				assert.Equal(t, uint32(tt.runNumber), storedRunNumber, "Run number mismatch")

				// Check LDC ID
				storedLDCID := binary.LittleEndian.Uint32(result[64:68])
				assert.Equal(t, uint32(tt.ldcID), storedLDCID, "LDC ID mismatch")

				// Check magic number is always correct
				magicNumber := binary.LittleEndian.Uint32(result[4:8])
				assert.Equal(t, uint32(duck.EVENT_MAGIC_NUMBER), magicNumber, "Magic number mismatch")
			}
		})
	}
}

// TestAddEquipmentHeader_TableDriven_BoundaryValues tests boundary values for equipment header fields
func TestAddEquipmentHeader_TableDriven_BoundaryValues(t *testing.T) {
	tests := []struct {
		name          string
		equipmentID   int
		equipmentType int
		dataSize      int
		expectPass    bool
	}{
		{
			name:          "Zero values",
			equipmentID:   0,
			equipmentType: 0,
			dataSize:      4,
			expectPass:    true,
		},
		{
			name:          "Typical values",
			equipmentID:   5,
			equipmentType: 22,
			dataSize:      100,
			expectPass:    true,
		},
		{
			name:          "Max uint16 equipment ID",
			equipmentID:   65535,
			equipmentType: 22,
			dataSize:      4,
			expectPass:    true,
		},
		{
			name:          "Max uint16 equipment type",
			equipmentID:   1,
			equipmentType: 65535,
			dataSize:      4,
			expectPass:    true,
		},
		{
			name:          "Empty subevent",
			equipmentID:   1,
			equipmentType: 22,
			dataSize:      0,
			expectPass:    true,
		},
		{
			name:          "Single byte subevent",
			equipmentID:   1,
			equipmentType: 22,
			dataSize:      1,
			expectPass:    true,
		},
		{
			name:          "Large subevent (1MB)",
			equipmentID:   1,
			equipmentType: 22,
			dataSize:      1024 * 1024,
			expectPass:    true,
		},
		{
			name:          "Word-aligned data (4 bytes)",
			equipmentID:   1,
			equipmentType: 22,
			dataSize:      4,
			expectPass:    true,
		},
		{
			name:          "Non-word-aligned data (5 bytes)",
			equipmentID:   1,
			equipmentType: 22,
			dataSize:      5,
			expectPass:    true,
		},
		{
			name:          "Equipment ID 1",
			equipmentID:   1,
			equipmentType: 1,
			dataSize:      4,
			expectPass:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testhelpers.SetupTest()
			// Create equipment with specified values
			equipment := duck.Equipment{
				ID:   tt.equipmentID,
				Type: tt.equipmentType,
			}

			// Create subevent data
			subevent := make([]byte, tt.dataSize)
			for i := range subevent {
				subevent[i] = byte(i % 256)
			}

			// Act
			result := addEquipmentHeader(logger, subevent, equipment)

			// Assert
			if tt.expectPass {
				// Check equipment size
				equipmentSize := binary.LittleEndian.Uint32(result[0:4])
				expectedSize := uint32(tt.dataSize + 28) // header is 7*4 = 28 bytes
				assert.Equal(t, expectedSize, equipmentSize, "Equipment size mismatch")

				// Check equipment type
				storedType := binary.LittleEndian.Uint32(result[4:8])
				assert.Equal(t, uint32(tt.equipmentType), storedType, "Equipment type mismatch")

				// Check equipment ID
				storedID := binary.LittleEndian.Uint32(result[8:12])
				assert.Equal(t, uint32(tt.equipmentID), storedID, "Equipment ID mismatch")

				// Check basic element size is always 4
				basicElementSize := binary.LittleEndian.Uint32(result[24:28])
				assert.Equal(t, uint32(4), basicElementSize, "Basic element size should be 4")

				// Verify total length
				assert.Equal(t, tt.dataSize+28, len(result), "Result length mismatch")
			}
		})
	}
}

// TestAddLDCHeader_MultipleEquipments_TableDriven tests with various numbers of equipments
func TestAddLDCHeader_MultipleEquipments_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		numEquipments  int
		dataSizePerEq  int
		expectedTotal  int
	}{
		{
			name:           "No equipments",
			numEquipments:  0,
			dataSizePerEq:  0,
			expectedTotal:  80, // Just header
		},
		{
			name:           "Single equipment",
			numEquipments:  1,
			dataSizePerEq:  10,
			expectedTotal:  80 + 10,
		},
		{
			name:           "Two equipments",
			numEquipments:  2,
			dataSizePerEq:  20,
			expectedTotal:  80 + 40, // 2 * 20
		},
		{
			name:           "Many equipments",
			numEquipments:  10,
			dataSizePerEq:  4,
			expectedTotal:  80 + 40, // 10 * 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", tt.numEquipments)
			runNumber := 100
			eventID := 1

			// Create equipment data
			equipmentsData := make(map[int][]byte)
			for i := 1; i <= tt.numEquipments; i++ {
				data := make([]byte, tt.dataSizePerEq)
				for j := range data {
					data[j] = byte((i + j) % 256)
				}
				equipmentsData[i] = data
			}

			// Act
			result := addLDCHeader(equipmentsData, eventID, ldcConfig, runNumber)

			// Assert
			assert.Equal(t, tt.expectedTotal, len(result), "Total size mismatch")

			// Check event size field matches actual size
			eventSize := binary.LittleEndian.Uint32(result[0:4])
			assert.Equal(t, uint32(tt.expectedTotal), eventSize, "Event size field mismatch")
		})
	}
}

// TestAddLDCHeader_DataIntegrity verifies that equipment data is preserved byte-by-byte
// Reference: gdcRPC/headers_test.go:189-248 (TestAddGDCHeader_DataIntegrity)
func TestAddLDCHeader_DataIntegrity(t *testing.T) {
	// Setup - verify that equipment data is preserved correctly
	ldcConfig := testhelpers.NewTestLDCConfiguration(1, "testldc", 2)
	runNumber := 100
	eventID := 42

	// Create equipment data
	equipmentsData := make(map[int][]byte)
	equipmentsData[1] = testhelpers.CreateEquipmentEventData(42, 1, 100)
	equipmentsData[2] = testhelpers.CreateEquipmentEventData(42, 2, 200)

	// Act
	result := addLDCHeader(equipmentsData, eventID, ldcConfig, runNumber)

	// Assert
	assert.NotNil(t, result)

	// Verify event size field
	eventSize := binary.LittleEndian.Uint32(result[0:4])
	expectedSize := len(equipmentsData[1]) + len(equipmentsData[2]) + 80
	assert.Equal(t, uint32(expectedSize), eventSize, "Event size should match header + all equipment data")

	// Verify total result size matches expected
	totalEquipmentSize := len(equipmentsData[1]) + len(equipmentsData[2])
	assert.Equal(t, 80+totalEquipmentSize, len(result), "Total size should be header + all equipment data")

	// The payload starts at byte 80
	dataSection := result[80:]
	allEquipmentData := [][]byte{equipmentsData[1], equipmentsData[2]}

	// Verify data integrity - each byte should be preserved
	for _, equipment := range allEquipmentData {
		found := false
		for i := 0; i <= len(dataSection)-len(equipment); i++ {
			match := true
			for j := 0; j < len(equipment); j++ {
				if dataSection[i+j] != equipment[j] {
					match = false
					break
				}
			}
			if match {
				found = true
				break
			}
		}
		assert.True(t, found, "Each byte of equipment data should be preserved correctly")
	}
}

func TestAddEquipmentHeader_NonWordAligned_PreservesAllBytes(t *testing.T) {
	logger := testhelpers.SetupTest()
	equipment := testhelpers.NewTestEquipment(1, 22)

	// Test with 5 bytes (1 word + 1 remainder byte)
	subevent := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	result := addEquipmentHeader(logger, subevent, equipment)

	// Verify total size
	assert.Equal(t, 28+5, len(result))

	// Verify first 4 bytes are endian-flipped
	originalWord := binary.LittleEndian.Uint32(subevent[0:4])
	resultWord := binary.BigEndian.Uint32(result[28:32])
	assert.Equal(t, originalWord, resultWord)

	// Verify 5th byte is preserved (not dropped!)
	assert.Equal(t, byte(0x05), result[32], "5th byte should be preserved")
}

func TestAddLDCHeader_AllFieldsVerified(t *testing.T) {
	ldcConfig := testhelpers.NewTestLDCConfiguration(5, "testldc", 2)
	runNumber := 12345
	eventID := 678

	equipmentsData := make(map[int][]byte)
	equipmentsData[1] = []byte{0x01, 0x02}

	result := addLDCHeader(equipmentsData, eventID, ldcConfig, runNumber)

	// Verify ALL 20 uint32 fields in the header (bytes 0-79)
	checks := []struct {
		offset    int
		expected  uint32
		fieldName string
	}{
		{0, 82, "EventSize"},                                                     // 80 + 2 bytes data
		{4, uint32(duck.EVENT_MAGIC_NUMBER), "Magic"},
		{8, 80, "HeaderSize"},
		{12, uint32(duck.EVENT_CURRENT_VERSION), "Version"},
		{16, uint32(duck.PHYSICS_EVENT), "EventType"},
		{20, 12345, "RunNumber"},
		{24, 678, "EventID"},
		{28, 0, "EventId[1]"},
		{32, 0, "TriggerPattern"},
		{36, 0, "DetectorPattern"},
		{40, 0, "EventTypeAttribute[0]"},
		{44, 0, "EventTypeAttribute[1]"},
		{48, 0, "Reserved0"},
		{52, 0, "Reserved1"},
		{56, 0, "Reserved2"},
		{60, uint32(duck.ORIGINAL_EVENT), "TypeAttribute"},
		{64, 5, "LDC_ID"},
		{68, uint32(duck.GDC_VOID), "GDC_ID"},
	}

	for _, check := range checks {
		value := binary.LittleEndian.Uint32(result[check.offset : check.offset+4])
		assert.Equal(t, check.expected, value, "%s mismatch at offset %d", check.fieldName, check.offset)
	}

	// Verify timestamp fields are set (non-zero)
	timestampSec := binary.LittleEndian.Uint32(result[72:76])
	assert.Greater(t, timestampSec, uint32(0), "TimestampSec should be set")

	timestampUSec := binary.LittleEndian.Uint32(result[76:80])
	assert.Greater(t, timestampUSec, uint32(0), "TimestampUSec should be set")
}
