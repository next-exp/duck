package main

import (
	"encoding/binary"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"golang.org/x/exp/maps"
)

func addGDCHeader(ldcData map[int][]byte, eventID int, gdcID int, runNumber int) []byte {
	timestamp := time.Now().UnixMicro()
	timestampSec := timestamp / 1000000
	timestampUSec := timestamp % 1000000

	totalSizeLDCData := 0
	data := maps.Values(ldcData)
	for i := 0; i < len(data); i++ {
		totalSizeLDCData += len(data[i])
	}

	// Build gdc header
	gdcHeader := duck.EventHeaderStruct{
		EventSize:          duck.EventSizeType(totalSizeLDCData + 20*4),
		EventMagic:         duck.EVENT_MAGIC_NUMBER,
		EventHeadSize:      80,
		EventVersion:       duck.EVENT_CURRENT_VERSION,
		EventType:          duck.PHYSICS_EVENT,
		EventRunNb:         duck.EventRunNbType(runNumber),
		EventLdcId:         duck.LDC_VOID,
		EventGdcId:         duck.EventGdcIdType(gdcID),
		EventTimestampSec:  duck.EventTimestampSecType(timestampSec),
		EventTimestampUsec: duck.EventTimestampUsecType(timestampUSec),
	}
	gdcHeader.EventId[0] = uint32(eventID)
	gdcHeader.EventTypeAttribute[2] = duck.ORIGINAL_EVENT | duck.SUPER_EVENT

	gdcData := make([]byte, totalSizeLDCData+20*4)
	position := 80
	for i := 0; i < len(data); i++ {
		size := len(data[i])
		copy(gdcData[position:position+size], data[i])
		position += size
	}

	binary.LittleEndian.PutUint32(gdcData[0:4], uint32(gdcHeader.EventSize))
	binary.LittleEndian.PutUint32(gdcData[4:8], uint32(gdcHeader.EventMagic))
	binary.LittleEndian.PutUint32(gdcData[8:12], uint32(gdcHeader.EventHeadSize))
	binary.LittleEndian.PutUint32(gdcData[12:16], uint32(gdcHeader.EventVersion))
	binary.LittleEndian.PutUint32(gdcData[16:20], uint32(gdcHeader.EventType))
	binary.LittleEndian.PutUint32(gdcData[20:24], uint32(gdcHeader.EventRunNb))
	binary.LittleEndian.PutUint32(gdcData[24:28], uint32(gdcHeader.EventId[0]))
	binary.LittleEndian.PutUint32(gdcData[60:64], uint32(gdcHeader.EventTypeAttribute[2]))
	binary.LittleEndian.PutUint32(gdcData[64:68], uint32(gdcHeader.EventLdcId))
	binary.LittleEndian.PutUint32(gdcData[68:72], uint32(gdcHeader.EventGdcId))
	binary.LittleEndian.PutUint32(gdcData[72:76], uint32(gdcHeader.EventTimestampSec))
	binary.LittleEndian.PutUint32(gdcData[76:80], uint32(gdcHeader.EventTimestampUsec))
	return gdcData
}
