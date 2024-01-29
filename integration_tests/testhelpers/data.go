//go:build integration
// +build integration

package testhelpers

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// GenerateTestEvent creates UDP packets for a complete event
// eventID: the event identifier
// numPackets: number of packets to generate (excluding end marker)
func GenerateTestEvent(eventID int, numPackets int) [][]byte {
	packets := make([][]byte, 0, numPackets+1)

	// Packet format expected by readPacket: 1400 bytes per packet, last 4 bytes 0xfa 0xfa 0xfa 0xfa as end marker.
	for i := 0; i < numPackets; i++ {
		packet := make([]byte, UDPDataPacketSize)
		// Sequence counter in first 4 bytes (big endian)
		binary.BigEndian.PutUint32(packet[0:4], uint32(i))
		// Fill rest with pattern
		for j := 4; j < 1400; j++ {
			packet[j] = byte((i + j) % 256)
		}
		packets = append(packets, packet)
	}
	// End marker as 4-byte packet 0xfafafafa (little endian)
	endMarker := make([]byte, 4)
	binary.LittleEndian.PutUint32(endMarker, 0xfafafafa)
	packets = append(packets, endMarker)

	return packets
}

// GenerateEventWithSequenceMismatch creates UDP packets with a sequence counter mismatch
// This simulates packet loss where the LDC expects a different sequence number
// The first packet is valid, but the second packet has an incorrect sequence counter
func GenerateEventWithSequenceMismatch() [][]byte {
	packets := make([][]byte, 0, 3)

	// First packet: sequence 0 (correct)
	packet0 := make([]byte, UDPDataPacketSize)
	binary.BigEndian.PutUint32(packet0[0:4], 0) // sequence 0
	for j := 4; j < UDPDataPacketSize; j++ {
		packet0[j] = byte(j % 256)
	}
	packets = append(packets, packet0)

	// Second packet: sequence 5 (WRONG! should be 1) - this triggers the error
	packet1 := make([]byte, UDPDataPacketSize)
	binary.BigEndian.PutUint32(packet1[0:4], 5) // sequence 5, but LDC expects 1
	for j := 4; j < UDPDataPacketSize; j++ {
		packet1[j] = byte((1 + j) % 256)
	}
	packets = append(packets, packet1)

	return packets
}

// UDPSender wraps UDP connection for testing
type UDPSender struct {
	conn *net.UDPConn
	dest *net.UDPAddr
}

// CreateUDPSender creates a UDP sender for testing
func CreateUDPSender(destIP string, destPort int) (*UDPSender, error) {
	// Resolve destination address
	destAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", destIP, destPort))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve destination: %w", err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, destAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP connection: %w", err)
	}

	return &UDPSender{
		conn: conn,
		dest: destAddr,
	}, nil
}

// Send sends a UDP packet
func (u *UDPSender) Send(data []byte) error {
	_, err := u.conn.Write(data)
	return err
}

// SendEvent sends all packets for a complete event
func (u *UDPSender) SendEvent(packets [][]byte) error {
	for _, packet := range packets {
		if err := u.Send(packet); err != nil {
			return fmt.Errorf("failed to send packet: %w", err)
		}
		// Small delay between packets to simulate real equipment
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// Close closes the UDP connection
func (u *UDPSender) Close() error {
	if u.conn != nil {
		return u.conn.Close()
	}
	return nil
}

// GenerateMultiEquipmentEvent generates packets for multiple equipments for the same event
// equipmentIDs: list of equipment IDs to generate packets for
// eventID: the event identifier (same for all equipments)
// numPacketsPerEquipment: number of packets to generate for each equipment
// Returns a map keyed by equipment ID containing packet arrays for each equipment
func GenerateMultiEquipmentEvent(equipmentIDs []int, eventID int, numPacketsPerEquipment int) map[int][][]byte {
	result := make(map[int][][]byte)

	for _, eqID := range equipmentIDs {
		packets := make([][]byte, 0, numPacketsPerEquipment+1)

		// Generate data packets for this equipment
		for i := 0; i < numPacketsPerEquipment; i++ {
			packet := make([]byte, UDPDataPacketSize)
			// Sequence counter in first 4 bytes (big endian)
			binary.BigEndian.PutUint32(packet[0:4], uint32(i))
			// Equipment ID in bytes 4-7 for identification
			binary.BigEndian.PutUint32(packet[4:8], uint32(eqID))
			// Event ID in bytes 8-11
			binary.BigEndian.PutUint32(packet[8:12], uint32(eventID))
			// Fill rest with pattern that includes equipment ID
			for j := 12; j < UDPDataPacketSize; j++ {
				packet[j] = byte((eqID + i + j) % 256)
			}
			packets = append(packets, packet)
		}

		// Add end marker
		endMarker := make([]byte, 4)
		binary.LittleEndian.PutUint32(endMarker, 0xfafafafa)
		packets = append(packets, endMarker)

		result[eqID] = packets
	}

	return result
}

// SendMultiEquipmentEvent sends packets from multiple equipments concurrently
// equipmentPackets: map of equipment ID to packets (from GenerateMultiEquipmentEvent)
// ip: destination IP address
// ports: map of equipment ID to destination port
// Returns error if any equipment fails to send
func SendMultiEquipmentEvent(equipmentPackets map[int][][]byte, ip string, ports map[int]int) error {
	type sendResult struct {
		eqID int
		err  error
	}

	resultChan := make(chan sendResult, len(equipmentPackets))

	// Send from each equipment concurrently
	for eqID, packets := range equipmentPackets {
		go func(equipmentID int, pkt [][]byte) {
			sender, err := CreateUDPSender(ip, ports[equipmentID])
			if err != nil {
				resultChan <- sendResult{eqID: equipmentID, err: fmt.Errorf("failed to create sender: %w", err)}
				return
			}
			defer sender.Close()

			if err := sender.SendEvent(pkt); err != nil {
				resultChan <- sendResult{eqID: equipmentID, err: fmt.Errorf("failed to send event: %w", err)}
				return
			}

			resultChan <- sendResult{eqID: equipmentID, err: nil}
		}(eqID, packets)
	}

	// Collect results
	var firstErr error
	for i := 0; i < len(equipmentPackets); i++ {
		result := <-resultChan
		if result.err != nil && firstErr == nil {
			firstErr = result.err
		}
	}

	return firstErr
}

// GenerateEventWithSequenceSkip creates packets where the sequence counter skips
// This simulates a lost packet, triggering a sequence mismatch error in LDC
// skipAtPacket: the packet index where the skip occurs (0-indexed)
// The sequence counter will be normal up to skipAtPacket, then skip by 1
func GenerateEventWithSequenceSkip(equipmentID, eventID, numPackets, skipAtPacket int) [][]byte {
	packets := make([][]byte, 0, numPackets+1)

	for i := 0; i < numPackets; i++ {
		packet := make([]byte, UDPDataPacketSize)

		// Calculate sequence counter - skip one value at skipAtPacket
		seq := i
		if i >= skipAtPacket {
			seq = i + 1 // Skip one sequence number
		}

		// Sequence counter in first 4 bytes (big endian) - read by LDC
		binary.BigEndian.PutUint32(packet[0:4], uint32(seq))
		// Equipment ID (little endian - will be flipped by LDC)
		binary.LittleEndian.PutUint32(packet[4:8], uint32(equipmentID))
		// Event ID (little endian - will be flipped by LDC)
		binary.LittleEndian.PutUint32(packet[8:12], uint32(eventID))

		// Fill rest with pattern
		for j := 12; j < UDPDataPacketSize; j++ {
			packet[j] = byte((equipmentID + i + j) % 256)
		}
		packets = append(packets, packet)
	}

	// Add end marker
	endMarker := make([]byte, 4)
	binary.LittleEndian.PutUint32(endMarker, 0xfafafafa)
	packets = append(packets, endMarker)

	return packets
}
