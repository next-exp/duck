package duck

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestEventMagicNumber_Constants(t *testing.T) {
	assert.Equal(t, EventMagicType(0xDA1E5AFE), EVENT_MAGIC_NUMBER)
	assert.Equal(t, EventMagicType(0xFE5A1EDA), EVENT_MAGIC_NUMBER_SWAPPED)
}

func TestEventVersion_Constants(t *testing.T) {
	// Verify the version calculation
	assert.Equal(t, 3, EVENT_MAJOR_VERSION_NUMBER)
	assert.Equal(t, 14, EVENT_MINOR_VERSION_NUMBER)

	// Calculate expected version: ((3 << 16) & 0xffff0000) | (14 & 0x0000ffff)
	expectedVersion := ((3 << 16) & 0xffff0000) | (14 & 0x0000ffff)
	assert.Equal(t, expectedVersion, EVENT_CURRENT_VERSION)

	// Verify the version is 0x0003000E (196622 in decimal)
	assert.Equal(t, 0x0003000E, EVENT_CURRENT_VERSION)
}

func TestEventHeaderStruct_Size(t *testing.T) {
	// Verify struct size to detect accidental changes
	var header EventHeaderStruct
	expectedSize := uintptr(80) // 4+4+4+4+4+4+8+16+4+12+4+4+4+4 = 80 bytes

	actualSize := unsafe.Sizeof(header)
	assert.Equal(t, expectedSize, actualSize, "EventHeaderStruct size changed unexpectedly")
}

func TestEquipmentHeaderStruct_Size(t *testing.T) {
	var header EquipmentHeaderStruct
	expectedSize := uintptr(28) // 4+4+4+12+4 = 28 bytes

	actualSize := unsafe.Sizeof(header)
	assert.Equal(t, expectedSize, actualSize, "EquipmentHeaderStruct size changed unexpectedly")
}

func TestEventTypeType_AllConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant EventTypeType
		expected EventTypeType
	}{
		{"START_OF_RUN", START_OF_RUN, 1},
		{"END_OF_RUN", END_OF_RUN, 2},
		{"START_OF_RUN_FILES", START_OF_RUN_FILES, 3},
		{"END_OF_RUN_FILES", END_OF_RUN_FILES, 4},
		{"START_OF_BURST", START_OF_BURST, 5},
		{"END_OF_BURST", END_OF_BURST, 6},
		{"PHYSICS_EVENT", PHYSICS_EVENT, 7},
		{"CALIBRATION_EVENT", CALIBRATION_EVENT, 8},
		{"EVENT_FORMAT_ERROR", EVENT_FORMAT_ERROR, 9},
		{"START_OF_DATA", START_OF_DATA, 10},
		{"END_OF_DATA", END_OF_DATA, 11},
		{"SYSTEM_SOFTWARE_TRIGGER_EVENT", SYSTEM_SOFTWARE_TRIGGER_EVENT, 12},
		{"DETECTOR_SOFTWARE_TRIGGER_EVENT", DETECTOR_SOFTWARE_TRIGGER_EVENT, 13},
		{"SYNC_EVENT", SYNC_EVENT, 14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestEventTypeType_IotaStartsAtOne(t *testing.T) {
	// Verify iota + 1 means START_OF_RUN is 1, not 0
	assert.Equal(t, EventTypeType(1), START_OF_RUN)
	assert.NotEqual(t, EventTypeType(0), START_OF_RUN)
}

func TestEventTypeType_Consecutive(t *testing.T) {
	// Verify all event types are consecutive
	assert.Equal(t, START_OF_RUN+1, END_OF_RUN)
	assert.Equal(t, END_OF_RUN+1, START_OF_RUN_FILES)
	assert.Equal(t, START_OF_RUN_FILES+1, END_OF_RUN_FILES)
	assert.Equal(t, END_OF_RUN_FILES+1, START_OF_BURST)
	assert.Equal(t, START_OF_BURST+1, END_OF_BURST)
	assert.Equal(t, END_OF_BURST+1, PHYSICS_EVENT)
	assert.Equal(t, PHYSICS_EVENT+1, CALIBRATION_EVENT)
	assert.Equal(t, CALIBRATION_EVENT+1, EVENT_FORMAT_ERROR)
	assert.Equal(t, EVENT_FORMAT_ERROR+1, START_OF_DATA)
	assert.Equal(t, START_OF_DATA+1, END_OF_DATA)
	assert.Equal(t, END_OF_DATA+1, SYSTEM_SOFTWARE_TRIGGER_EVENT)
	assert.Equal(t, SYSTEM_SOFTWARE_TRIGGER_EVENT+1, DETECTOR_SOFTWARE_TRIGGER_EVENT)
	assert.Equal(t, DETECTOR_SOFTWARE_TRIGGER_EVENT+1, SYNC_EVENT)
}

func TestLDCGDCVoidConstants(t *testing.T) {
	assert.Equal(t, EventLdcIdType(0xffffffff), LDC_VOID)
	assert.Equal(t, EventGdcIdType(0xffffffff), GDC_VOID)
}

func TestEventAttributeConstants(t *testing.T) {
	assert.Equal(t, uint32(0x00000010), uint32(SUPER_EVENT))
	assert.Equal(t, uint32(0x00000200), uint32(ORIGINAL_EVENT))
}

func TestEventSizeType_TypeAlias(t *testing.T) {
	var size EventSizeType = 1024
	assert.Equal(t, uint32(1024), uint32(size))
}

func TestEventMagicType_TypeAlias(t *testing.T) {
	var magic EventMagicType = 0xDA1E5AFE
	assert.Equal(t, uint32(0xDA1E5AFE), uint32(magic))
}

func TestEventVersionType_TypeAlias(t *testing.T) {
	var version EventVersionType = 0x0003000E
	assert.Equal(t, uint32(0x0003000E), uint32(version))
}

func TestEventIdType_Size(t *testing.T) {
	var id EventIdType
	assert.Equal(t, 2, len(id))
	assert.Equal(t, uintptr(8), unsafe.Sizeof(id)) // 2 * 4 bytes = 8 bytes
}

func TestEventTriggerPatternType_Size(t *testing.T) {
	var pattern EventTriggerPatternType
	assert.Equal(t, 4, len(pattern))
	assert.Equal(t, uintptr(16), unsafe.Sizeof(pattern)) // 4 * 4 bytes = 16 bytes
}

func TestEventTypeAttributeType_Size(t *testing.T) {
	var attr EventTypeAttributeType
	assert.Equal(t, 3, len(attr))
	assert.Equal(t, uintptr(12), unsafe.Sizeof(attr)) // 3 * 4 bytes = 12 bytes
}

func TestEquipmentTypeAttributeType_IsAlias(t *testing.T) {
	// Verify EquipmentTypeAttributeType is same as EventTypeAttributeType
	var eqAttr EquipmentTypeAttributeType
	var evAttr EventTypeAttributeType

	assert.Equal(t, unsafe.Sizeof(evAttr), unsafe.Sizeof(eqAttr))
	assert.Equal(t, len(evAttr), len(eqAttr))
}

func TestEventHeaderStruct_FieldTypes(t *testing.T) {
	header := EventHeaderStruct{
		EventSize:            1024,
		EventMagic:           EVENT_MAGIC_NUMBER,
		EventHeadSize:        80,
		EventVersion:         EventVersionType(EVENT_CURRENT_VERSION),
		EventType:            PHYSICS_EVENT,
		EventRunNb:           12345,
		EventId:              EventIdType{1, 2},
		EventTriggerPattern:  EventTriggerPatternType{0, 0, 0, 0},
		EventDetectorPattern: 0xFF,
		EventTypeAttribute:   EventTypeAttributeType{SUPER_EVENT, 0, 0},
		EventLdcId:           1,
		EventGdcId:           2,
		EventTimestampSec:    1234567890,
		EventTimestampUsec:   123456,
	}

	assert.Equal(t, EventSizeType(1024), header.EventSize)
	assert.Equal(t, EVENT_MAGIC_NUMBER, header.EventMagic)
	assert.Equal(t, EventHeadSizeType(80), header.EventHeadSize)
	assert.Equal(t, PHYSICS_EVENT, header.EventType)
	assert.Equal(t, EventRunNbType(12345), header.EventRunNb)
}

func TestEquipmentHeaderStruct_FieldTypes(t *testing.T) {
	header := EquipmentHeaderStruct{
		EquipmentSize:             512,
		EquipmentType:             22,
		EquipmentId:               1,
		EquipmentTypeAttribute:    EquipmentTypeAttributeType{0, 0, 0},
		EquipmentBasicElementSize: 4,
	}

	assert.Equal(t, EquipmentSizeType(512), header.EquipmentSize)
	assert.Equal(t, EquipmentTypeType(22), header.EquipmentType)
	assert.Equal(t, EquipmentIdType(1), header.EquipmentId)
	assert.Equal(t, EquipmentBasicElementSizeType(4), header.EquipmentBasicElementSize)
}

func TestEventMagicNumber_ByteOrderDetection(t *testing.T) {
	// The magic number can be used to detect byte order
	// Regular: 0xDA1E5AFE
	// Swapped: 0xFE5A1EDA

	normalMagic := EVENT_MAGIC_NUMBER
	swappedMagic := EVENT_MAGIC_NUMBER_SWAPPED

	// Verify they are byte-reversed versions of each other
	normalBytes := []byte{
		byte(normalMagic >> 24),
		byte(normalMagic >> 16),
		byte(normalMagic >> 8),
		byte(normalMagic),
	}
	swappedBytes := []byte{
		byte(swappedMagic >> 24),
		byte(swappedMagic >> 16),
		byte(swappedMagic >> 8),
		byte(swappedMagic),
	}

	// Verify byte reversal
	assert.Equal(t, normalBytes[0], swappedBytes[3])
	assert.Equal(t, normalBytes[1], swappedBytes[2])
	assert.Equal(t, normalBytes[2], swappedBytes[1])
	assert.Equal(t, normalBytes[3], swappedBytes[0])
}
