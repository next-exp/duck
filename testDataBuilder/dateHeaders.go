package main

type EventSizeType uint32

type EventMagicType uint32

const EVENT_MAGIC_NUMBER EventMagicType = 0xDA1E5AFE
const EVENT_MAGIC_NUMBER_SWAPPED EventMagicType = 0xFE5A1EDA

type EventHeadSizeType uint32

/* ---------- Unique version identifier ---------- */
const EVENT_MAJOR_VERSION_NUMBER = 3
const EVENT_MINOR_VERSION_NUMBER = 14
const EVENT_CURRENT_VERSION = ((EVENT_MAJOR_VERSION_NUMBER << 16) & 0xffff0000) | (EVENT_MINOR_VERSION_NUMBER & 0x0000ffff)

type EventVersionType uint32

/* ---------- Event type ---------- */
type EventTypeType uint32

const (
	START_OF_RUN EventTypeType = iota + 1
	END_OF_RUN
	START_OF_RUN_FILES
	END_OF_RUN_FILES
	START_OF_BURST
	END_OF_BURST
	PHYSICS_EVENT
	CALIBRATION_EVENT
	EVENT_FORMAT_ERROR
	START_OF_DATA
	END_OF_DATA
	SYSTEM_SOFTWARE_TRIGGER_EVENT
	DETECTOR_SOFTWARE_TRIGGER_EVENT
	SYNC_EVENT
)

type EventRunNbType uint32

/* ---------- The eventId field ---------- */
type EventIdType [2]uint32

/* ---------- Trigger pattern (and relative masks) ---------- */
type EventTriggerPatternType [4]uint32

/* ---------- Detectors cluster (and relative masks) ---------- */
type EventDetectorPatternType uint32

/* ---------- Type  ---------- */
type EventTypeAttributeType [3]uint32

const SUPER_EVENT = 0x00000010
const ORIGINAL_EVENT = 0x00000200

/* ---------- LDC and GDC identifier ---------- */
type EventHostIdType uint32
type EventLdcIdType EventHostIdType
type EventGdcIdType EventHostIdType

const LDC_VOID EventLdcIdType = 0xffffffff
const GDC_VOID EventGdcIdType = 0xffffffff

/*
---------- Timestamps ----------

	The timestamp is split into seconds and microseconds.
*/
type EventTimestampSecType uint32

/* Microseconds: range [0..999999] */
type EventTimestampUsecType uint32

/* ---------- The event header structure (with + without data) ---------- */
type EventHeaderStruct struct {
	EventSize            EventSizeType
	EventMagic           EventMagicType
	EventHeadSize        EventHeadSizeType
	EventVersion         EventVersionType
	EventType            EventTypeType
	EventRunNb           EventRunNbType
	EventId              EventIdType
	EventTriggerPattern  EventTriggerPatternType
	EventDetectorPattern EventDetectorPatternType
	EventTypeAttribute   EventTypeAttributeType
	EventLdcId           EventLdcIdType
	EventGdcId           EventGdcIdType
	EventTimestampSec    EventTimestampSecType
	EventTimestampUsec   EventTimestampUsecType
}

type EquipmentSizeType uint32
type EquipmentTypeType uint32
type EquipmentIdType uint32
type EquipmentTypeAttributeType EventTypeAttributeType
type EquipmentBasicElementSizeType uint32

type EquipmentHeaderStruct struct {
	EquipmentSize             EquipmentSizeType
	EquipmentType             EquipmentTypeType
	EquipmentId               EquipmentIdType
	EquipmentTypeAttribute    EquipmentTypeAttributeType
	EquipmentBasicElementSize EquipmentBasicElementSizeType
}

func EventIdGetNbInRun(id EventIdType) uint32 {
	return id[0]
}
