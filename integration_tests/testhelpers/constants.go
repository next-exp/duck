//go:build integration
// +build integration

package testhelpers

import "time"

// Network protocol constants
const (
	// UDPDataPacketSize is the size of UDP data packets in bytes
	UDPDataPacketSize = 1400

	// LDCPacketHeaderSize is the size of the LDC packet header in 4-byte words
	// LDC sends 20 words (80 bytes) as event header
	LDCPacketHeaderWords = 20

	// EquipmentPacketHeaderSize is the size of the equipment packet header in 4-byte words
	// Equipment sends 7 words (28 bytes) + payload + end marker
	EquipmentPacketHeaderWords = 7

	// EventEndMarkerSize is the size of the event end marker in bytes
	EventEndMarkerSize = 4
)

// CalculateExpectedEventSize calculates the expected total size of an event in bytes
// Event = LDC header (20 * 4) + Equipment header (7 * 4) + Payload + End marker (4)
func CalculateExpectedEventSize(numPackets int) int64 {
	return int64((LDCPacketHeaderWords * 4) + (EquipmentPacketHeaderWords*4 + UDPDataPacketSize*numPackets + EventEndMarkerSize))
}

// Test timeout constants
const (
	// StateChangeTimeout is the timeout for waiting for state transitions
	StateChangeTimeout = 10 * time.Second

	// EventProcessingTimeout is the timeout for waiting for events to be processed
	EventProcessingTimeout = 5 * time.Second
)

// Test data constants
const (
	// DefaultTestEventID is the default event ID for tests
	DefaultTestEventID = 1

	// DefaultTestPacketCount is the default number of packets per event for tests
	DefaultTestPacketCount = 10
)
