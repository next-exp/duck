package main

import (
	"encoding/binary"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"golang.org/x/exp/maps"
)

func addLDCHeader(equipmentsData map[int][]byte, eventID int, ldcConfiguration duck.LDCConfiguration, runNumber int) []byte {
	timestamp := time.Now().UnixMicro()
	timestampSec := timestamp / 1000000
	timestampUSec := timestamp % 1000000

	totalSizeEquipmentData := 0
	data := maps.Values(equipmentsData)
	for i := 0; i < len(data); i++ {
		totalSizeEquipmentData += len(data[i])
	}

	// Build LDC header
	ldcHeader := duck.EventHeaderStruct{
		EventSize:          duck.EventSizeType(totalSizeEquipmentData + 20*4),
		EventMagic:         duck.EVENT_MAGIC_NUMBER,
		EventHeadSize:      80,
		EventVersion:       duck.EVENT_CURRENT_VERSION,
		EventType:          duck.PHYSICS_EVENT,
		EventRunNb:         duck.EventRunNbType(runNumber),
		EventLdcId:         duck.EventLdcIdType(ldcConfiguration.ID),
		EventGdcId:         duck.GDC_VOID,
		EventTimestampSec:  duck.EventTimestampSecType(timestampSec),
		EventTimestampUsec: duck.EventTimestampUsecType(timestampUSec),
	}
	ldcHeader.EventId[0] = uint32(eventID)
	ldcHeader.EventTypeAttribute[2] = duck.ORIGINAL_EVENT

	ldcData := make([]byte, totalSizeEquipmentData+20*4)
	position := 80
	for i := 0; i < len(data); i++ {
		size := len(data[i])
		copy(ldcData[position:position+size], data[i])
		position += size
	}
	binary.LittleEndian.PutUint32(ldcData[0:4], uint32(ldcHeader.EventSize))
	binary.LittleEndian.PutUint32(ldcData[4:8], uint32(ldcHeader.EventMagic))
	binary.LittleEndian.PutUint32(ldcData[8:12], uint32(ldcHeader.EventHeadSize))
	binary.LittleEndian.PutUint32(ldcData[12:16], uint32(ldcHeader.EventVersion))
	binary.LittleEndian.PutUint32(ldcData[16:20], uint32(ldcHeader.EventType))
	binary.LittleEndian.PutUint32(ldcData[20:24], uint32(ldcHeader.EventRunNb))
	binary.LittleEndian.PutUint32(ldcData[24:28], uint32(ldcHeader.EventId[0]))
	binary.LittleEndian.PutUint32(ldcData[60:64], uint32(ldcHeader.EventTypeAttribute[2]))
	binary.LittleEndian.PutUint32(ldcData[64:68], uint32(ldcHeader.EventLdcId))
	binary.LittleEndian.PutUint32(ldcData[68:72], uint32(ldcHeader.EventGdcId))
	binary.LittleEndian.PutUint32(ldcData[72:76], uint32(ldcHeader.EventTimestampSec))
	binary.LittleEndian.PutUint32(ldcData[76:80], uint32(ldcHeader.EventTimestampUsec))
	return ldcData
}

func addEquipmentHeader(logger duck.DuckLogger, subevent []byte, equipment duck.Equipment) []byte {
	equipmentHeader := duck.EquipmentHeaderStruct{
		EquipmentSize:             duck.EquipmentSizeType(len(subevent) + 7*4), //add header size
		EquipmentType:             duck.EquipmentTypeType(equipment.Type),
		EquipmentId:               duck.EquipmentIdType(equipment.ID),
		EquipmentBasicElementSize: 4,
	}

	equipmentData := make([]byte, len(subevent)+7*4)
	//copy(equipmentData[28:], subevent)
	// Copy data flipping endianness
	numWords := len(subevent) / 4
	for i := 0; i < numWords; i++ {
		word := binary.LittleEndian.Uint32(subevent[i*4 : (i+1)*4])
		binary.BigEndian.PutUint32(equipmentData[28+i*4:28+(i+1)*4], word)
	}

	// Copy remaining bytes (0-3 bytes) without endianness flip
	remainder := len(subevent) % 4
	if remainder > 0 {
		logger.Slog.Error("Non-word-aligned equipment data",
			"equipment_id", equipment.ID,
			"data_size", len(subevent),
			"remainder_bytes", remainder)
		copy(equipmentData[28+numWords*4:28+numWords*4+remainder], subevent[numWords*4:])
	}

	binary.LittleEndian.PutUint32(equipmentData[0:4], uint32(equipmentHeader.EquipmentSize))
	binary.LittleEndian.PutUint32(equipmentData[4:8], uint32(equipmentHeader.EquipmentType))
	binary.LittleEndian.PutUint32(equipmentData[8:12], uint32(equipmentHeader.EquipmentId))
	binary.LittleEndian.PutUint32(equipmentData[24:28], uint32(equipmentHeader.EquipmentBasicElementSize))
	return equipmentData
}
