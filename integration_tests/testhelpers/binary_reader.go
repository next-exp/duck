//go:build integration
// +build integration

package testhelpers

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

const (
	GDCHeaderSize       = 80
	LDCHeaderSize       = 80
	EquipmentHeaderSize = 28
	EventMagicNumber    = 0xDA1E5AFE
)

type BinaryEvent struct {
	EventID   uint32
	GDCID     uint32
	RunNumber uint32
	Size      uint32
	LDCData   []LDCEventData
	RawData   []byte
	Offset    int
}

type LDCEventData struct {
	LDCID         uint32
	EventID       uint32
	Size          uint32
	EquipmentData []EquipmentEventData
	Offset        int
}

type EquipmentEventData struct {
	EquipmentID   uint32
	EquipmentType uint32
	Size          uint32
	Data          []byte
	Offset        int
}

type BinaryFileReader struct {
	data     []byte
	position int
}

func ReadBinaryEventsFromFile(filePath string) ([]BinaryEvent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	reader := &BinaryFileReader{data: data}
	var events []BinaryEvent

	for reader.position < len(reader.data) {
		event, err := reader.readEvent()
		if err != nil {
			return nil, fmt.Errorf("error reading event at position %d: %w", reader.position, err)
		}
		if event == nil {
			break
		}
		events = append(events, *event)
	}

	return events, nil
}

func (r *BinaryFileReader) readEvent() (*BinaryEvent, error) {
	if r.position+GDCHeaderSize > len(r.data) {
		return nil, nil
	}

	eventStart := r.position

	eventSize := binary.LittleEndian.Uint32(r.data[r.position+0 : r.position+4])
	if eventSize == 0 || int(eventSize) > len(r.data)-r.position {
		return nil, nil
	}

	magic := binary.LittleEndian.Uint32(r.data[r.position+4 : r.position+8])
	if magic != EventMagicNumber {
		return nil, fmt.Errorf("invalid magic number: expected 0x%08X, got 0x%08X", EventMagicNumber, magic)
	}

	runNumber := binary.LittleEndian.Uint32(r.data[r.position+20 : r.position+24])
	eventID := binary.LittleEndian.Uint32(r.data[r.position+24 : r.position+28])
	gdcID := binary.LittleEndian.Uint32(r.data[r.position+68 : r.position+72])

	event := &BinaryEvent{
		EventID:   eventID,
		GDCID:     gdcID,
		RunNumber: runNumber,
		Size:      eventSize,
		Offset:    eventStart,
	}

	event.RawData = r.data[r.position : r.position+int(eventSize)]

	ldcStart := r.position + GDCHeaderSize
	ldcEnd := r.position + int(eventSize)

	for ldcStart < ldcEnd {
		ldcData, err := r.readLDCData(ldcStart)
		if err != nil {
			return nil, err
		}
		if ldcData == nil {
			break
		}
		event.LDCData = append(event.LDCData, *ldcData)
		ldcStart += int(ldcData.Size)
	}

	r.position += int(eventSize)

	return event, nil
}

func (r *BinaryFileReader) readLDCData(offset int) (*LDCEventData, error) {
	if offset+LDCHeaderSize > len(r.data) {
		return nil, nil
	}

	ldcSize := binary.LittleEndian.Uint32(r.data[offset+0 : offset+4])
	if ldcSize == 0 || offset+int(ldcSize) > len(r.data) {
		return nil, nil
	}

	ldcID := binary.LittleEndian.Uint32(r.data[offset+64 : offset+68])
	eventID := binary.LittleEndian.Uint32(r.data[offset+24 : offset+28])

	ldc := &LDCEventData{
		LDCID:   ldcID,
		EventID: eventID,
		Size:    ldcSize,
		Offset:  offset,
	}

	equipStart := offset + LDCHeaderSize
	equipEnd := offset + int(ldcSize)

	for equipStart < equipEnd {
		equipData, err := r.readEquipmentData(equipStart)
		if err != nil {
			return nil, err
		}
		if equipData == nil {
			break
		}
		ldc.EquipmentData = append(ldc.EquipmentData, *equipData)
		equipStart += int(equipData.Size)
	}

	return ldc, nil
}

func (r *BinaryFileReader) readEquipmentData(offset int) (*EquipmentEventData, error) {
	if offset+EquipmentHeaderSize > len(r.data) {
		return nil, nil
	}

	equipSize := binary.LittleEndian.Uint32(r.data[offset+0 : offset+4])
	if equipSize == 0 || offset+int(equipSize) > len(r.data) {
		return nil, nil
	}

	equipType := binary.LittleEndian.Uint32(r.data[offset+4 : offset+8])
	equipID := binary.LittleEndian.Uint32(r.data[offset+8 : offset+12])

	dataSize := int(equipSize) - EquipmentHeaderSize
	if dataSize < 0 {
		return nil, fmt.Errorf("invalid equipment data size: %d", equipSize)
	}

	equip := &EquipmentEventData{
		EquipmentID:   equipID,
		EquipmentType: equipType,
		Size:          equipSize,
		Offset:        offset,
	}

	if dataSize > 0 {
		equip.Data = r.data[offset+EquipmentHeaderSize : offset+int(equipSize)]
	}

	return equip, nil
}

func FindBinaryFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".rd" {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

func ExtractEventIDs(events []BinaryEvent) []uint32 {
	ids := make([]uint32, len(events))
	for i, event := range events {
		ids[i] = event.EventID
	}
	return ids
}

func VerifyEventHasAllEquipments(event BinaryEvent, expectedEquipIDs []uint32) error {
	foundEquipIDs := make(map[uint32]bool)

	for _, ldc := range event.LDCData {
		for _, equip := range ldc.EquipmentData {
			foundEquipIDs[equip.EquipmentID] = true
		}
	}

	for _, expectedID := range expectedEquipIDs {
		if !foundEquipIDs[expectedID] {
			return fmt.Errorf("event %d missing equipment ID %d (found: %v)", event.EventID, expectedID, getKeys(foundEquipIDs))
		}
	}

	return nil
}

func VerifyEventHasAllLDCs(event BinaryEvent, expectedLDCIDs []uint32) error {
	foundLDCIDs := make(map[uint32]bool)

	for _, ldc := range event.LDCData {
		foundLDCIDs[ldc.LDCID] = true
	}

	for _, expectedID := range expectedLDCIDs {
		if !foundLDCIDs[expectedID] {
			return fmt.Errorf("event %d missing LDC ID %d (found: %v)", event.EventID, expectedID, getKeys(foundLDCIDs))
		}
	}

	return nil
}

func VerifyEquipmentDataPattern(equip EquipmentEventData, eventID uint32) error {
	data := equip.Data
	if len(data) < 12 {
		return fmt.Errorf("equipment %d data too short: %d bytes", equip.EquipmentID, len(data))
	}

	for i := 0; i < len(data)/4; i++ {
		word := binary.BigEndian.Uint32(data[i*4 : (i+1)*4])

		if i == 0 {
			continue
		}
		if i == 1 {
			if word != equip.EquipmentID {
				return fmt.Errorf("equipment %d data at word 1: expected equipment ID %d, got %d", equip.EquipmentID, equip.EquipmentID, word)
			}
		}
		if i == 2 {
			if word != eventID {
				return fmt.Errorf("equipment %d data at word 2: expected event ID %d, got %d", equip.EquipmentID, eventID, word)
			}
		}
	}

	return nil
}

func getKeys(m map[uint32]bool) []uint32 {
	keys := make([]uint32, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func CountEventsByGDC(events []BinaryEvent) map[uint32]int {
	counts := make(map[uint32]int)
	for _, event := range events {
		counts[event.GDCID]++
	}
	return counts
}

func CountEquipmentsInEvent(event BinaryEvent) map[uint32]int {
	counts := make(map[uint32]int)
	for _, ldc := range event.LDCData {
		for _, equip := range ldc.EquipmentData {
			counts[equip.EquipmentID]++
		}
	}
	return counts
}
