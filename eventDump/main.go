package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// equipmentData holds equipment header and data for sorting
type equipmentData struct {
	header    *duck.EquipmentHeaderStruct
	data      []byte
	ldcHeader *duck.EventHeaderStruct
	ldcOffset int
}

func main() {
	// Parse command-line flags
	maxEvents := flag.Int("n", -1, "Maximum number of events to dump (-1 for all)")
	headersOnly := flag.Bool("h", false, "Print headers only (no binary dump)")
	sortedByEquipID := flag.Bool("s", false, "Prescan and sort equipments by equipment ID")
	flag.Parse()

	// Get filename from remaining arguments
	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [-n num_events] [-h] [-s] filename\n", os.Args[0])
		os.Exit(1)
	}
	filename := flag.Arg(0)

	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Print header
	if *maxEvents > 0 {
		fmt.Printf("Taking max. %d events\n", *maxEvents)
	}

	// Process events
	eventCount := 0
	for {
		// Check if we've reached the maximum number of events
		if *maxEvents > 0 && eventCount >= *maxEvents {
			break
		}

		// Read and dump event
		done, err := dumpEvent(file, eventCount, *headersOnly, *sortedByEquipID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading event: %v\n", err)
			break
		}
		if done {
			break
		}

		eventCount++
	}
}

// readEventHeader reads an event header from the file
func readEventHeader(file *os.File) (*duck.EventHeaderStruct, error) {
	// Read the header (80 bytes = 20 uint32 values)
	headerBytes := make([]byte, 80)
	n, err := file.Read(headerBytes)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("EOF")
	}
	if n < 80 {
		return nil, fmt.Errorf("incomplete header: got %d bytes, expected 80", n)
	}

	// Parse the header
	header := &duck.EventHeaderStruct{
		EventSize:     duck.EventSizeType(binary.LittleEndian.Uint32(headerBytes[0:4])),
		EventMagic:    duck.EventMagicType(binary.LittleEndian.Uint32(headerBytes[4:8])),
		EventHeadSize: duck.EventHeadSizeType(binary.LittleEndian.Uint32(headerBytes[8:12])),
		EventVersion:  duck.EventVersionType(binary.LittleEndian.Uint32(headerBytes[12:16])),
		EventType:     duck.EventTypeType(binary.LittleEndian.Uint32(headerBytes[16:20])),
		EventRunNb:    duck.EventRunNbType(binary.LittleEndian.Uint32(headerBytes[20:24])),
		EventId: duck.EventIdType{
			binary.LittleEndian.Uint32(headerBytes[24:28]),
			binary.LittleEndian.Uint32(headerBytes[28:32]),
		},
		EventTriggerPattern: duck.EventTriggerPatternType{
			binary.LittleEndian.Uint32(headerBytes[32:36]),
			binary.LittleEndian.Uint32(headerBytes[36:40]),
			binary.LittleEndian.Uint32(headerBytes[40:44]),
			binary.LittleEndian.Uint32(headerBytes[44:48]),
		},
		EventDetectorPattern: duck.EventDetectorPatternType(binary.LittleEndian.Uint32(headerBytes[48:52])),
		EventTypeAttribute: duck.EventTypeAttributeType{
			binary.LittleEndian.Uint32(headerBytes[52:56]),
			binary.LittleEndian.Uint32(headerBytes[56:60]),
			binary.LittleEndian.Uint32(headerBytes[60:64]),
		},
		EventLdcId:         duck.EventLdcIdType(binary.LittleEndian.Uint32(headerBytes[64:68])),
		EventGdcId:         duck.EventGdcIdType(binary.LittleEndian.Uint32(headerBytes[68:72])),
		EventTimestampSec:  duck.EventTimestampSecType(binary.LittleEndian.Uint32(headerBytes[72:76])),
		EventTimestampUsec: duck.EventTimestampUsecType(binary.LittleEndian.Uint32(headerBytes[76:80])),
	}

	return header, nil
}

// countSubEvents counts the number of sub-events in event data
func countSubEvents(eventData []byte) int {
	count := 0
	offset := 0
	for offset < len(eventData) {
		if offset+80 > len(eventData) {
			break
		}
		eventSize := binary.LittleEndian.Uint32(eventData[offset : offset+4])
		if eventSize == 0 || int(eventSize) > len(eventData)-offset {
			break
		}
		count++
		offset += int(eventSize)
	}
	return count
}

// readEquipmentHeader reads an equipment header from a byte slice
func readEquipmentHeader(data []byte) *duck.EquipmentHeaderStruct {
	return &duck.EquipmentHeaderStruct{
		EquipmentSize: duck.EquipmentSizeType(binary.LittleEndian.Uint32(data[0:4])),
		EquipmentType: duck.EquipmentTypeType(binary.LittleEndian.Uint32(data[4:8])),
		EquipmentId:   duck.EquipmentIdType(binary.LittleEndian.Uint32(data[8:12])),
		EquipmentTypeAttribute: duck.EquipmentTypeAttributeType{
			binary.LittleEndian.Uint32(data[12:16]),
			binary.LittleEndian.Uint32(data[16:20]),
			binary.LittleEndian.Uint32(data[20:24]),
		},
		EquipmentBasicElementSize: duck.EquipmentBasicElementSizeType(binary.LittleEndian.Uint32(data[24:28])),
	}
}

// dumpEvent reads and dumps a single event
func dumpEvent(file *os.File, eventNum int, headersOnly bool, sortedByEquipID bool) (bool, error) {
	// Read the GDC event header
	gdcHeader, err := readEventHeader(file)
	if err != nil {
		return true, err
	}

	// Calculate the remaining data size (event size includes header)
	remainingSize := int(gdcHeader.EventSize) - 80

	// Read the remaining event data
	eventData := make([]byte, remainingSize)
	n, err := file.Read(eventData)
	if err != nil {
		return true, fmt.Errorf("error reading event data: %w", err)
	}
	if n < remainingSize {
		return true, fmt.Errorf("incomplete event data: got %d bytes, expected %d", n, remainingSize)
	}

	// Count sub-events
	subEventCount := countSubEvents(eventData)

	// Print separator line
	printEqualsLine()

	// Print GDC event header
	printEventHeaderWithSubevents(gdcHeader, "GDC", 0, subEventCount)

	if sortedByEquipID {
		// Prescan mode: collect all equipment data, then sort and print
		dumpEventSorted(eventData, headersOnly)
	} else {
		// Normal mode: print as we parse
		dumpEventNormal(eventData, headersOnly)
	}

	return false, nil
}

// dumpEventNormal prints event data in the order it appears in the file
func dumpEventNormal(eventData []byte, headersOnly bool) {
	// Parse sub-events (LDC events)
	offset := 0
	for offset < len(eventData) {
		// Check if there's enough data for a header
		if offset+80 > len(eventData) {
			break
		}

		// Parse LDC header from the event data
		ldcHeader := parseLDCHeader(eventData[offset : offset+80])

		// Print separator before each LDC subevent
		printDottedLine()

		// Print LDC header
		printEventHeader(ldcHeader, "LDC", offset)

		// Parse equipment headers within this LDC event
		equipOffset := offset + 80
		for equipOffset < offset+int(ldcHeader.EventSize) {
			if equipOffset+28 > len(eventData) {
				break
			}

			equipHeader := readEquipmentHeader(eventData[equipOffset : equipOffset+28])
			equipDataSize := int(equipHeader.EquipmentSize) - 28

			// Print equipment header and data
			printEquipmentHeader(equipHeader)

			// Print equipment data (hex dump) if not headers-only
			if !headersOnly && equipDataSize > 0 && equipOffset+28+equipDataSize <= len(eventData) {
				printHexDump(eventData[equipOffset+28 : equipOffset+28+equipDataSize])
			}

			equipOffset += int(equipHeader.EquipmentSize)
		}

		offset += int(ldcHeader.EventSize)
	}
}

// dumpEventSorted prescans all equipment, sorts by ID, then prints
func dumpEventSorted(eventData []byte, headersOnly bool) {
	// First pass: collect all equipment data
	var equipments []equipmentData

	offset := 0
	for offset < len(eventData) {
		// Check if there's enough data for a header
		if offset+80 > len(eventData) {
			break
		}

		// Parse LDC header from the event data
		ldcHeader := parseLDCHeader(eventData[offset : offset+80])

		// Parse equipment headers within this LDC event
		equipOffset := offset + 80
		for equipOffset < offset+int(ldcHeader.EventSize) {
			if equipOffset+28 > len(eventData) {
				break
			}

			equipHeader := readEquipmentHeader(eventData[equipOffset : equipOffset+28])
			equipDataSize := int(equipHeader.EquipmentSize) - 28

			// Collect equipment data
			var data []byte
			if equipDataSize > 0 && equipOffset+28+equipDataSize <= len(eventData) {
				data = eventData[equipOffset+28 : equipOffset+28+equipDataSize]
			}

			equipments = append(equipments, equipmentData{
				header:    equipHeader,
				data:      data,
				ldcHeader: ldcHeader,
				ldcOffset: offset,
			})

			equipOffset += int(equipHeader.EquipmentSize)
		}

		offset += int(ldcHeader.EventSize)
	}

	// Sort by equipment ID
	sort.Slice(equipments, func(i, j int) bool {
		return equipments[i].header.EquipmentId < equipments[j].header.EquipmentId
	})

	// Print sorted equipment
	var lastLDCId duck.EventLdcIdType = 0xFFFFFFFF // Invalid initial value
	for _, equip := range equipments {
		// Print LDC header when we encounter a new LDC
		if lastLDCId != equip.ldcHeader.EventLdcId {
			printDottedLine()
			printEventHeader(equip.ldcHeader, "LDC", equip.ldcOffset)
			lastLDCId = equip.ldcHeader.EventLdcId
		}

		// Print equipment header and data
		printEquipmentHeader(equip.header)

		// Print equipment data (hex dump) if not headers-only
		if !headersOnly && len(equip.data) > 0 {
			printHexDump(equip.data)
		}
	}
}

// parseLDCHeader parses an LDC header from a byte slice
func parseLDCHeader(data []byte) *duck.EventHeaderStruct {
	return &duck.EventHeaderStruct{
		EventSize:     duck.EventSizeType(binary.LittleEndian.Uint32(data[0:4])),
		EventMagic:    duck.EventMagicType(binary.LittleEndian.Uint32(data[4:8])),
		EventHeadSize: duck.EventHeadSizeType(binary.LittleEndian.Uint32(data[8:12])),
		EventVersion:  duck.EventVersionType(binary.LittleEndian.Uint32(data[12:16])),
		EventType:     duck.EventTypeType(binary.LittleEndian.Uint32(data[16:20])),
		EventRunNb:    duck.EventRunNbType(binary.LittleEndian.Uint32(data[20:24])),
		EventId: duck.EventIdType{
			binary.LittleEndian.Uint32(data[24:28]),
			binary.LittleEndian.Uint32(data[28:32]),
		},
		EventTriggerPattern: duck.EventTriggerPatternType{
			binary.LittleEndian.Uint32(data[32:36]),
			binary.LittleEndian.Uint32(data[36:40]),
			binary.LittleEndian.Uint32(data[40:44]),
			binary.LittleEndian.Uint32(data[44:48]),
		},
		EventDetectorPattern: duck.EventDetectorPatternType(binary.LittleEndian.Uint32(data[48:52])),
		EventTypeAttribute: duck.EventTypeAttributeType{
			binary.LittleEndian.Uint32(data[52:56]),
			binary.LittleEndian.Uint32(data[56:60]),
			binary.LittleEndian.Uint32(data[60:64]),
		},
		EventLdcId:         duck.EventLdcIdType(binary.LittleEndian.Uint32(data[64:68])),
		EventGdcId:         duck.EventGdcIdType(binary.LittleEndian.Uint32(data[68:72])),
		EventTimestampSec:  duck.EventTimestampSecType(binary.LittleEndian.Uint32(data[72:76])),
		EventTimestampUsec: duck.EventTimestampUsecType(binary.LittleEndian.Uint32(data[76:80])),
	}
}

// printEventHeaderWithSubevents prints a GDC event header with subevent count
func printEventHeaderWithSubevents(header *duck.EventHeaderStruct, eventType string, offset int, subevents int) {
	// Format timestamp
	timestamp := time.Unix(int64(header.EventTimestampSec), int64(header.EventTimestampUsec)*1000)
	timestampStr := timestamp.Format("Mon Jan 2 15:04:05 2006") + fmt.Sprintf(" +%06dusec", header.EventTimestampUsec)

	// Print size and version line with subevents
	fmt.Printf("Size:%d (header:%d) Version:0x%08x Type:%s Subevents:%d\n",
		header.EventSize, header.EventHeadSize, header.EventVersion,
		getEventTypeName(header.EventType), subevents)

	// Format LDC and GDC IDs
	ldcIdStr := "VOID"
	if header.EventLdcId != duck.LDC_VOID {
		ldcIdStr = fmt.Sprintf("%d", header.EventLdcId)
	}
	gdcIdStr := "VOID"
	if header.EventGdcId != duck.GDC_VOID {
		gdcIdStr = fmt.Sprintf("%d", header.EventGdcId)
	}

	// Print run info line
	fmt.Printf("RunNb:%d nbInRun:%d burstNb:%d nbInBurst:%d ldcId:%s gdcId:%s time:%s Attributes:%s(%08x.%08x.%08x)\n",
		header.EventRunNb, header.EventId[0], header.EventId[1], 0,
		ldcIdStr, gdcIdStr, timestampStr,
		getAttributesStr(header.EventTypeAttribute),
		header.EventTypeAttribute[0], header.EventTypeAttribute[1], header.EventTypeAttribute[2])

	// Print trigger and detector patterns
	fmt.Printf("triggerPattern: 0)%08x 1)%08x 2)%08x 3)%08x detectorPattern:%08x[invalid]\n",
		header.EventTriggerPattern[0], header.EventTriggerPattern[1],
		header.EventTriggerPattern[2], header.EventTriggerPattern[3],
		header.EventDetectorPattern)
}

// printEventHeader prints an event header in the format matching the example
func printEventHeader(header *duck.EventHeaderStruct, eventType string, offset int) {
	// Format timestamp
	timestamp := time.Unix(int64(header.EventTimestampSec), int64(header.EventTimestampUsec)*1000)
	timestampStr := timestamp.Format("Mon Jan 2 15:04:05 2006") + fmt.Sprintf(" +%06dusec", header.EventTimestampUsec)

	// Determine number of subevents (only for GDC events)
	subeventsStr := ""
	if eventType == "GDC" {
		// Count subevents - we'll need to parse through the data, but for now we'll skip this
		subeventsStr = " Subevents:?"
	}

	// Print size and version line
	fmt.Printf("Size:%d (header:%d) Version:0x%08x Type:%s%s\n",
		header.EventSize, header.EventHeadSize, header.EventVersion,
		getEventTypeName(header.EventType), subeventsStr)

	// Format LDC and GDC IDs
	ldcIdStr := "VOID"
	if header.EventLdcId != duck.LDC_VOID {
		ldcIdStr = fmt.Sprintf("%d", header.EventLdcId)
	}
	gdcIdStr := "VOID"
	if header.EventGdcId != duck.GDC_VOID {
		gdcIdStr = fmt.Sprintf("%d", header.EventGdcId)
	}

	// Print run info line
	fmt.Printf("RunNb:%d nbInRun:%d burstNb:%d nbInBurst:%d ldcId:%s gdcId:%s time:%s Attributes:%s(%08x.%08x.%08x)\n",
		header.EventRunNb, header.EventId[0], header.EventId[1], 0,
		ldcIdStr, gdcIdStr, timestampStr,
		getAttributesStr(header.EventTypeAttribute),
		header.EventTypeAttribute[0], header.EventTypeAttribute[1], header.EventTypeAttribute[2])

	// Print trigger and detector patterns
	fmt.Printf("triggerPattern: 0)%08x 1)%08x 2)%08x 3)%08x detectorPattern:%08x[invalid]\n",
		header.EventTriggerPattern[0], header.EventTriggerPattern[1],
		header.EventTriggerPattern[2], header.EventTriggerPattern[3],
		header.EventDetectorPattern)
}

// printEquipmentHeader prints an equipment header
func printEquipmentHeader(header *duck.EquipmentHeaderStruct) {
	fmt.Printf(" - Equipment: size:%d type:%d id:%d basicElementSize:%d attributes:%s(%08x.%08x.%08x)\n",
		header.EquipmentSize, header.EquipmentType, header.EquipmentId,
		header.EquipmentBasicElementSize,
		getEquipmentAttributesStr(header.EquipmentTypeAttribute),
		header.EquipmentTypeAttribute[0], header.EquipmentTypeAttribute[1], header.EquipmentTypeAttribute[2])
}

// printHexDump prints a hex dump of data in the format matching the example
func printHexDump(data []byte) {
	for i := 0; i < len(data); i += 16 {
		// Print offset
		fmt.Printf("  %2d) ", i)

		// Print hex values
		for j := 0; j < 16 && i+j < len(data); j += 4 {
			if i+j+3 < len(data) {
				word := binary.LittleEndian.Uint32(data[i+j : i+j+4])
				fmt.Printf("%08x ", word)
			} else {
				// Handle incomplete words at the end
				for k := 0; k < 4 && i+j+k < len(data); k++ {
					fmt.Printf("%02x", data[i+j+k])
				}
				fmt.Printf(" ")
			}
		}

		// Print ASCII representation
		fmt.Printf("| ")
		for j := 0; j < 16 && i+j < len(data); j += 4 {
			for k := 0; k < 4 && i+j+k < len(data); k++ {
				c := data[i+j+k]
				if c >= 32 && c <= 126 {
					fmt.Printf("%c", c)
				} else {
					fmt.Printf(".")
				}
			}
			fmt.Printf(" ")
		}
		fmt.Printf("|\n")
	}
}

// printEqualsLine prints an equals separator line
func printEqualsLine() {
	fmt.Println("===============================================================================")
}

// printDottedLine prints a separator line
func printDottedLine() {
	fmt.Println("...............................................................................")
}

// getEventTypeName returns the name of an event type
func getEventTypeName(eventType duck.EventTypeType) string {
	switch eventType {
	case duck.START_OF_RUN:
		return "StartOfRun"
	case duck.END_OF_RUN:
		return "EndOfRun"
	case duck.PHYSICS_EVENT:
		return "PhysicsEvent"
	case duck.CALIBRATION_EVENT:
		return "CalibrationEvent"
	default:
		return fmt.Sprintf("Unknown(%d)", eventType)
	}
}

// getAttributesStr returns a string representation of event attributes
func getAttributesStr(attr duck.EventTypeAttributeType) string {
	attrs := []string{}
	if attr[2]&uint32(duck.SUPER_EVENT) != 0 {
		attrs = append(attrs, "Super")
	}
	if attr[2]&uint32(duck.ORIGINAL_EVENT) != 0 {
		attrs = append(attrs, "OriginalEvent")
	}
	if len(attrs) == 0 {
		return "noAttr"
	}
	result := ""
	for i, a := range attrs {
		if i > 0 {
			result += "+"
		}
		result += a
	}
	return result
}

// getEquipmentAttributesStr returns a string representation of equipment attributes
func getEquipmentAttributesStr(attr duck.EquipmentTypeAttributeType) string {
	// For equipment, typically all zeros means no attributes
	if attr[0] == 0 && attr[1] == 0 && attr[2] == 0 {
		return "noAttr"
	}
	return "hasAttr"
}
