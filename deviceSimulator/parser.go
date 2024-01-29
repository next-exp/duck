package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// EventData represents a single event's equipment data
type EventData struct {
	EventID     uint32            // Event ID from the event header
	Timestamp   uint64            // Timestamp in microseconds
	Equipments  map[uint32][]byte // Map of equipment ID to raw data payload
}

// FileParser handles parsing of .rd binary files
type FileParser struct {
	filePath string
	file     *os.File
}

// NewFileParser creates a new parser for the given file
func NewFileParser(filePath string) (*FileParser, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &FileParser{
		filePath: filePath,
		file:     file,
	}, nil
}

// Close closes the file
func (p *FileParser) Close() error {
	if p.file != nil {
		return p.file.Close()
	}
	return nil
}

// ParseAll reads all events from the file and returns them
func (p *FileParser) ParseAll() ([]EventData, error) {
	events := []EventData{}

	for {
		event, err := p.parseNextEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

// parseNextEvent reads and parses the next event from the file
func (p *FileParser) parseNextEvent() (EventData, error) {
	event := EventData{
		Equipments: make(map[uint32][]byte),
	}

	// Read GDC event header (80 bytes)
	gdcHeader := make([]byte, 80)
	_, err := io.ReadFull(p.file, gdcHeader)
	if err != nil {
		return event, err
	}

	// Parse GDC header
	eventSize := binary.LittleEndian.Uint32(gdcHeader[0:4])
	eventMagic := binary.LittleEndian.Uint32(gdcHeader[4:8])

	// Validate magic number
	if eventMagic != 0xDA1E5AFE {
		return event, fmt.Errorf("invalid magic number: 0x%X (expected 0xDA1E5AFE)", eventMagic)
	}

	// Extract event ID from header
	eventNbInRun := binary.LittleEndian.Uint32(gdcHeader[24:28])
	event.EventID = eventNbInRun

	// Extract timestamp (seconds + microseconds)
	timestampSec := binary.LittleEndian.Uint32(gdcHeader[72:76])
	timestampUsec := binary.LittleEndian.Uint32(gdcHeader[76:80])
	event.Timestamp = uint64(timestampSec)*1000000 + uint64(timestampUsec)

	// Read remaining event data (excluding header)
	eventDataSize := eventSize - 80
	eventData := make([]byte, eventDataSize)
	_, err = io.ReadFull(p.file, eventData)
	if err != nil {
		return event, fmt.Errorf("failed to read event data: %w", err)
	}

	// Parse LDC sub-events
	offset := uint32(0)
	for offset < eventDataSize {
		// Read LDC header (80 bytes)
		if offset+80 > eventDataSize {
			break
		}

		ldcHeader := eventData[offset : offset+80]
		ldcEventSize := binary.LittleEndian.Uint32(ldcHeader[0:4])
		ldcMagic := binary.LittleEndian.Uint32(ldcHeader[4:8])

		if ldcMagic != 0xDA1E5AFE {
			return event, fmt.Errorf("invalid LDC magic number: 0x%X", ldcMagic)
		}

		// Parse equipment sub-events within this LDC event
		ldcDataSize := ldcEventSize - 80
		ldcDataStart := offset + 80
		ldcDataEnd := ldcDataStart + ldcDataSize

		if ldcDataEnd > eventDataSize {
			return event, fmt.Errorf("LDC data extends beyond event boundary")
		}

		equipOffset := ldcDataStart
		for equipOffset < ldcDataEnd {
			// Read equipment header (28 bytes)
			if equipOffset+28 > ldcDataEnd {
				break
			}

			equipHeader := eventData[equipOffset : equipOffset+28]
			equipSize := binary.LittleEndian.Uint32(equipHeader[0:4])
			equipID := binary.LittleEndian.Uint32(equipHeader[8:12])

			// Extract equipment payload (excluding header)
			equipPayloadSize := equipSize - 28
			equipPayloadStart := equipOffset + 28
			equipPayloadEnd := equipPayloadStart + equipPayloadSize

			if equipPayloadEnd > ldcDataEnd {
				return event, fmt.Errorf("equipment payload extends beyond LDC boundary")
			}

			// Copy equipment payload
			equipPayload := make([]byte, equipPayloadSize)
			copy(equipPayload, eventData[equipPayloadStart:equipPayloadEnd])

			event.Equipments[equipID] = equipPayload

			equipOffset += equipSize
		}

		offset += ldcEventSize
	}

	return event, nil
}

// Reset rewinds the file to the beginning
func (p *FileParser) Reset() error {
	_, err := p.file.Seek(0, io.SeekStart)
	return err
}
