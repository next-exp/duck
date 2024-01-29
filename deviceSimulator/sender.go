package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// UDPSender handles sending equipment data via UDP packets
type UDPSender struct {
	equipment    duck.Equipment
	conn         *net.UDPConn
	fragmentSize int
	metrics      *SenderMetrics
}

// SenderMetrics tracks UDP sending statistics
type SenderMetrics struct {
	PacketsSent uint64
	EventsSent  uint64
	BytesSent   uint64
	Errors      uint64
}

const (
	// EndOfEventMarker is sent after all packets for an event
	EndOfEventMarker uint32 = 0xfafafafa
	// PacketHeaderSize is the size of seq# + eventID
	PacketHeaderSize = 8
	// DefaultPacketSize is the default packet payload size
	DefaultPacketSize = 500
)

// NewUDPSender creates a new UDP sender for the given equipment
func NewUDPSender(equipment duck.Equipment, fragmentSize int) (*UDPSender, error) {
	if fragmentSize == 0 {
		fragmentSize = 7992 // Default for NEXT-100
	}

	// Resolve UDP address
	addr := fmt.Sprintf("%s:%d", equipment.HostIP, equipment.HostPort)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address %s: %w", addr, err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial UDP address %s: %w", addr, err)
	}

	return &UDPSender{
		equipment:    equipment,
		conn:         conn,
		fragmentSize: fragmentSize,
		metrics:      &SenderMetrics{},
	}, nil
}

// unflipEndianness reverses the LDC endian flip operation
// The LDC flips every 4-byte word from LE to BE when writing to file
// We need to reverse this to get the original UDP packet format
func unflipEndianness(data []byte) []byte {
	result := make([]byte, len(data))

	// Process complete 4-byte words
	numWords := len(data) / 4
	for i := 0; i < numWords; i++ {
		offset := i * 4
		// Read as BigEndian (how LDC wrote it)
		word := binary.BigEndian.Uint32(data[offset : offset+4])
		// Write as LittleEndian (restore original)
		binary.LittleEndian.PutUint32(result[offset:offset+4], word)
	}

	// Copy any remaining bytes
	remainder := len(data) % 4
	if remainder > 0 {
		copy(result[len(data)-remainder:], data[len(data)-remainder:])
	}

	return result
}

// SendEvent sends equipment data for a single event
// The data comes from the .rd file and has been endian-flipped by the LDC
// We need to:
// 1. Un-flip the endianness to restore original UDP format
// 2. Extract packets based on fragmentSize
// 3. Validate sequence counters are consecutive
// 4. Send packets as-is
func (s *UDPSender) SendEvent(eventID uint32, data []byte) error {
	// Un-flip endianness to restore original UDP packet format
	originalData := unflipEndianness(data)

	// The original data now has format: [seq BE][eventID LE][payload]...
	// Find the end-of-event marker (0xfafafafa LE)
	endMarkerPos := -1
	for i := 0; i <= len(originalData)-4; i++ {
		marker := binary.LittleEndian.Uint32(originalData[i : i+4])
		if marker == EndOfEventMarker {
			endMarkerPos = i
			break
		}
	}

	if endMarkerPos < 0 {
		return fmt.Errorf("end-of-event marker not found")
	}

	// Extract packet data (everything before the end marker)
	packetData := originalData[:endMarkerPos]

	// Split data into fragments based on fragment size and validate sequence counters
	// UDP packet format: [seq counter BE 4 bytes][payload...]
	numPackets := (len(packetData) + s.fragmentSize - 1) / s.fragmentSize
	expectedSeq := uint32(0)

	for i := 0; i < numPackets; i++ {
		// Calculate packet boundaries
		start := i * s.fragmentSize
		end := start + s.fragmentSize
		if end > len(packetData) {
			end = len(packetData)
		}

		packet := packetData[start:end]

		// Validate packet has at least sequence counter header (4 bytes)
		if len(packet) < 4 {
			return fmt.Errorf("packet %d too small: %d bytes (need at least 4 for seq counter)", i+1, len(packet))
		}

		// Extract and validate sequence counter (first 4 bytes, Big Endian)
		seq := binary.BigEndian.Uint32(packet[0:4])
		if seq != expectedSeq {
			return fmt.Errorf("packet %d: sequence mismatch - expected %d, got %d", i+1, expectedSeq, seq)
		}

		// Send packet
		n, err := s.conn.Write(packet)
		if err != nil {
			s.metrics.Errors++
			return fmt.Errorf("failed to send packet %d/%d: %w", i+1, numPackets, err)
		}

		s.metrics.PacketsSent++
		s.metrics.BytesSent += uint64(n)
		expectedSeq++

		// Add 100ms delay between packets to prevent UDP reordering
		// This is especially important when sending many packets rapidly
		time.Sleep(100 * time.Millisecond)
	}

	// Send end-of-event marker
	if err := s.SendEndMarker(); err != nil {
		return fmt.Errorf("failed to send end marker: %w", err)
	}

	s.metrics.EventsSent++
	return nil
}

// SendEndMarker sends the 4-byte end-of-event marker
func (s *UDPSender) SendEndMarker() error {
	marker := make([]byte, 4)
	binary.LittleEndian.PutUint32(marker, EndOfEventMarker)

	n, err := s.conn.Write(marker)
	if err != nil {
		s.metrics.Errors++
		return err
	}

	s.metrics.PacketsSent++
	s.metrics.BytesSent += uint64(n)
	return nil
}

// SendRawPacket sends a raw packet without any processing
// Used by generator mode to send pre-built packets
func (s *UDPSender) SendRawPacket(packet []byte) error {
	n, err := s.conn.Write(packet)
	if err != nil {
		s.metrics.Errors++
		return err
	}

	s.metrics.PacketsSent++
	s.metrics.BytesSent += uint64(n)
	return nil
}

// Close closes the UDP connection
// Adds a small delay to allow any in-flight packets to be delivered
func (s *UDPSender) Close() error {
	if s.conn != nil {
		// Send a final end-of-event marker to ensure LDC is in clean state
		// This helps prevent sequence counter issues when restarting
		log.Println("Sending final end-of-event marker before closing")
		s.SendEndMarker()

		// Small delay to allow packets to be delivered
		time.Sleep(100 * time.Millisecond)

		return s.conn.Close()
	}
	return nil
}

// GetMetrics returns a copy of the current metrics
func (s *UDPSender) GetMetrics() SenderMetrics {
	return *s.metrics
}

// ResetMetrics resets all metrics to zero
func (s *UDPSender) ResetMetrics() {
	s.metrics = &SenderMetrics{}
}

// RateLimiter controls the rate of event sending
type RateLimiter struct {
	lastEventTime time.Time
	rateHz        float64
	eventDelay    time.Duration
}

// NewRateLimiter creates a rate limiter with the given rate in Hz
// rateHz is the desired events per second (e.g., 10.0 = 10 events/second)
// If rateHz <= 0, no rate limiting is applied (send as fast as possible)
func NewRateLimiter(rateHz float64) *RateLimiter {
	var eventDelay time.Duration
	if rateHz > 0 {
		eventDelay = time.Duration(float64(time.Second) / rateHz)
	}

	return &RateLimiter{
		lastEventTime: time.Now(),
		rateHz:        rateHz,
		eventDelay:    eventDelay,
	}
}

// Wait waits the appropriate time before sending the next event
// This ensures events are sent at the configured absolute rate in Hz
func (r *RateLimiter) Wait() {
	if r.rateHz <= 0 {
		// No rate limiting, send immediately
		return
	}

	// Calculate how long to wait to maintain the target rate
	elapsed := time.Since(r.lastEventTime)
	if r.eventDelay > elapsed {
		time.Sleep(r.eventDelay - elapsed)
	}

	r.lastEventTime = time.Now()
}
