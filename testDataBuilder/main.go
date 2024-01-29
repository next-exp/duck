package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"unsafe"
)

func getRawdataFiles(dataPath string) ([]string, error) {
	files, err := os.ReadDir(dataPath)
	if err != nil {
		return nil, fmt.Errorf("error reading data path: %w", err)
	}

	var rawdataFiles []string
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), "rd") {
			continue
		}
		if !file.IsDir() {
			rawdataFiles = append(rawdataFiles, fmt.Sprintf("%s/%s", dataPath, file.Name()))
		}
	}

	if len(rawdataFiles) == 0 {
		return nil, fmt.Errorf("no files found in the data path: %s", dataPath)
	}

	return rawdataFiles, nil
}

func processFile(filePath string, ldcID uint32, evtData map[int][]byte) error {
	file, err := os.Open(filePath)
	if err != nil {
		message := fmt.Errorf("Error opening file: %w", err)
		return message
	}
	defer file.Close()

	// Load all events in file
	for {
		header, eventData, err := readEvent(file)
		_ = eventData
		_ = header
		if err == io.EOF {
			message := fmt.Errorf("All events in the file has been read: %w", err)
			log.Println(message.Error())
			break
		}
		if err != nil {
			message := fmt.Errorf("Error reading event: %w", err)
			log.Println(message.Error())
			return message
		}
		fmt.Println("GDC: ", header.EventGdcId)
		fmt.Println("event ID: ", header.EventId[0])
		ldcData := ReadGDC(eventData, header, ldcID)
		evtData[int(header.EventId[0])] = ldcData
		if err != nil {
			message := fmt.Errorf("Error reading event: %w", err)
			log.Println(message.Error())
			return message
		}
	}

	return nil
}

func readEquipment(eventData []byte, position int, header EventHeaderStruct) int {
	var eqHeader EquipmentHeaderStruct
	eqHeaderSize := unsafe.Sizeof(eqHeader)

	eqHeaderBinary := eventData[position : position+int(eqHeaderSize)]
	eqHeaderReader := bytes.NewReader(eqHeaderBinary)
	binary.Read(eqHeaderReader, binary.LittleEndian, &eqHeader)
	fmt.Printf("Equipment Header: ID=%d, Size=%d bytes, Type=%d\n", eqHeader.EquipmentId, eqHeader.EquipmentSize, eqHeader.EquipmentType)
	nRead := int(eqHeader.EquipmentSize)
	return nRead
}

func readLDC(eventData []byte, position int, ldcID uint32) (int, []byte) {
	var header EventHeaderStruct
	headerSize := unsafe.Sizeof(header)
	ldcHeaderBinary := eventData[position : position+int(headerSize)]
	ldcHeaderReader := bytes.NewReader(ldcHeaderBinary)
	binary.Read(ldcHeaderReader, binary.LittleEndian, &header)

	if uint32(header.EventLdcId) != ldcID {
		// Skip this LDC if it doesn't match the requested LDC ID
		return int(header.EventSize), nil
	}

	fmt.Println("LDC Header:")
	fmt.Printf("  GDC ID: %d\n", header.EventGdcId)
	fmt.Printf("  LDC ID: %d\n", header.EventLdcId)
	fmt.Printf("  Event ID: %d\n", header.EventId[0])
	fmt.Printf("  Event Size: %d bytes\n", header.EventSize)
	fmt.Printf("  Event Head Size: %d bytes\n", header.EventHeadSize)
	fmt.Printf("  Event Type: %d\n", header.EventType)
	fmt.Printf("  Event Version: %d\n", header.EventVersion)
	fmt.Printf("  Event Timestamp: %d %d\n", header.EventTimestampSec, header.EventTimestampUsec)

	// Read equipment header
	startLDCPayload := position + int(header.EventHeadSize)
	startPosition := 0
	for {
		nRead := readEquipment(eventData[startLDCPayload:], startPosition, header)
		// Next equipment
		startPosition += nRead
		if startPosition+int(header.EventHeadSize) >= int(header.EventSize) {
			break
		}
	}

	ldcData := eventData[position : position+int(header.EventSize)]

	return int(header.EventSize), ldcData
}

func ReadGDC(eventData []byte, header EventHeaderStruct, ldcID uint32) []byte {
	// Read LDCs
	position := 0
	for {
		nRead, ldcData := readLDC(eventData, position, ldcID)

		// check data
		if ldcData != nil {
			readLDC(ldcData, 0, ldcID)
			return ldcData
		}

		// Next LDC
		position += nRead
		if position >= len(eventData) {
			break
		}
	}
	log.Fatalf("Error: No LDC data found for LDC ID %d in event %d", ldcID, header.EventId[0])
	return nil
}

var fd *os.File

func main() {
	dataPath := flag.String("data", "", "Data files path")
	ldcID := flag.Int("ldc", 0, "LDC ID")
	flag.Parse()

	fmt.Println("LDC ID:", *ldcID)

	rawFiles, err := getRawdataFiles(*dataPath)
	if err != nil {
		log.Fatalf("Error getting raw data files: %v", err)
	}

	fileout := fmt.Sprintf("%s/ldc%d.raw", *dataPath, *ldcID)
	fd, err = os.Create(fileout)

	evtData := make(map[int][]byte)

	for _, rawdataFile := range rawFiles {
		fmt.Println("Processing file:", rawdataFile)
		processFile(rawdataFile, uint32(*ldcID), evtData)
	}

	var sortedEventIDs []int
	for eventID := range evtData {
		sortedEventIDs = append(sortedEventIDs, eventID)
	}
	sort.Ints(sortedEventIDs)

	previousEventID := sortedEventIDs[0] - 1
	for _, eventID := range sortedEventIDs {
		if eventID != previousEventID+1 {
			log.Printf("Warning: Missing event ID %d, previous was %d\n", eventID, previousEventID)
		}
		previousEventID = eventID

		data := evtData[eventID]
		fmt.Printf("Writing event %d with data size %d bytes\n", eventID, len(data))
		err := writeBinaryData(data, fd)
		if err != nil {
			log.Fatalf("Error writing event data to file: %v", err)
		}
	}
}

func readEvent(file *os.File) (EventHeaderStruct, []byte, error) {
	var header EventHeaderStruct
	headerSize := unsafe.Sizeof(header)
	headerBinary := make([]byte, headerSize)
	nRead, err := file.Read(headerBinary)
	if err != nil {
		return header, nil, err
	}

	if nRead == 0 {
		return header, nil, err
	}

	headerReader := bytes.NewReader(headerBinary)
	binary.Read(headerReader, binary.LittleEndian, &header)

	//log.Println("Event ID: ", header.EventId[0])

	payloadSize := uint32(header.EventSize) - uint32(headerSize)
	eventData := make([]byte, payloadSize)
	file.Read(eventData)
	//combinedData := append(headerBinary, eventData...)
	return header, eventData, nil
}
