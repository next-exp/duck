package testhelpers

import (
	"encoding/binary"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// GenerateTestPacket creates a test packet with sequence counter and payload
func GenerateTestPacket(sequenceCounter int, payload []byte) []byte {
	packet := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(packet[0:4], uint32(sequenceCounter))
	copy(packet[4:], payload)
	return packet
}

// GenerateEndMarker creates the end-of-event marker (0xfafafafa)
func GenerateEndMarker() []byte {
	marker := make([]byte, 4)
	binary.LittleEndian.PutUint32(marker, 0xfafafafa)
	return marker
}

// GenerateCompleteEvent creates a complete event with multiple packets and end marker
func GenerateCompleteEvent(numPackets int, packetSize int) [][]byte {
	packets := make([][]byte, 0, numPackets+1)

	for i := 0; i < numPackets; i++ {
		// Create payload filled with sequence-specific data
		payload := make([]byte, packetSize)
		for j := range payload {
			payload[j] = byte((i + j) % 256)
		}
		packets = append(packets, GenerateTestPacket(i, payload))
	}

	// Add end marker
	packets = append(packets, GenerateEndMarker())

	return packets
}

// NewTestEquipment creates a test equipment configuration
func NewTestEquipment(id int, equipmentType int) duck.Equipment {
	return duck.Equipment{
		ID:       id,
		Type:     equipmentType,
		DeviceIP: "192.168.1.1",
		HostIP:   "127.0.0.1",
		HostPort: 6006 + id,
		LDC_ID:   1,
		Enabled:  true,
	}
}

// NewTestLDCConfiguration creates a test LDC configuration
func NewTestLDCConfiguration(id int, name string, numEquipments int) duck.LDCConfiguration {
	equipments := make([]duck.Equipment, numEquipments)
	for i := 0; i < numEquipments; i++ {
		equipments[i] = NewTestEquipment(i+1, 22)
	}

	return duck.LDCConfiguration{
		ID:             id,
		Name:           name,
		Host:           "testhost",
		IP:             "127.0.0.1",
		GRPCPort:       50052,
		PrometheusPort: 12113,
		Enabled:        true,
		Equipments:     equipments,
	}
}

// NewTestGDCConfiguration creates a test GDC configuration
func NewTestGDCConfiguration(id int, name string) duck.GDCConfiguration {
	return duck.GDCConfiguration{
		ID:             id,
		Name:           name,
		Host:           "testhost",
		IP:             "127.0.0.1",
		Port:           6005,
		GRPCPort:       50051,
		PrometheusPort: 12112,
		Path:           "/tmp/test",
		Enabled:        true,
		WriteOutput:    true,
		Decode:         false,
	}
}

// GenerateLargeEvent creates an event with many packets for stress testing
func GenerateLargeEvent(numPackets int, packetSize int) [][]byte {
	packets := make([][]byte, 0, numPackets+1)

	for i := 0; i < numPackets; i++ {
		// Create payload filled with sequence-specific data
		payload := make([]byte, packetSize)
		for j := range payload {
			payload[j] = byte((i + j) % 256)
		}
		packets = append(packets, GenerateTestPacket(i, payload))
	}

	// Add end marker
	packets = append(packets, GenerateEndMarker())

	return packets
}

// NewTestGDCConfigurationList creates multiple GDC configs for testing
func NewTestGDCConfigurationList(count int, enabled bool) []duck.GDCConfiguration {
	gdcs := make([]duck.GDCConfiguration, count)
	for i := 0; i < count; i++ {
		gdcs[i] = duck.GDCConfiguration{
			ID:             i + 1,
			Name:           "testgdc" + string(rune('0'+i+1)),
			Host:           "testhost",
			IP:             "127.0.0.1",
			Port:           6005 + i,
			GRPCPort:       50051 + i,
			PrometheusPort: 12112 + i,
			Path:           "/tmp/test",
			Enabled:        enabled,
			WriteOutput:    true,
			Decode:         false,
		}
	}
	return gdcs
}

// NewTestGDCConfigurationWithPort creates a GDC config with a specific port
func NewTestGDCConfigurationWithPort(id int, name string, port int, enabled bool) duck.GDCConfiguration {
	return duck.GDCConfiguration{
		ID:             id,
		Name:           name,
		Host:           "testhost",
		IP:             "127.0.0.1",
		Port:           port,
		GRPCPort:       50051,
		PrometheusPort: 12112,
		Path:           "/tmp/test",
		Enabled:        enabled,
		WriteOutput:    true,
		Decode:         false,
	}
}

// NewTestEquipmentDisabled creates a disabled test equipment configuration
func NewTestEquipmentDisabled(id int, equipmentType int) duck.Equipment {
	return duck.Equipment{
		ID:       id,
		Type:     equipmentType,
		DeviceIP: "192.168.1.1",
		HostIP:   "127.0.0.1",
		HostPort: 6006 + id,
		LDC_ID:   1,
		Enabled:  false,
	}
}

// NewTestLDCConfigurationWithEquipments creates a test LDC configuration with custom equipments
func NewTestLDCConfigurationWithEquipments(id int, name string, equipments []duck.Equipment) duck.LDCConfiguration {
	return duck.LDCConfiguration{
		ID:             id,
		Name:           name,
		Host:           "testhost",
		IP:             "127.0.0.1",
		GRPCPort:       50052,
		PrometheusPort: 12113,
		Enabled:        true,
		Equipments:     equipments,
	}
}

// NewTestEquipmentWithIP creates a test equipment configuration with custom IPs
func NewTestEquipmentWithIP(id int, equipmentType int, deviceIP, hostIP string, hostPort int) duck.Equipment {
	return duck.Equipment{
		ID:       id,
		Type:     equipmentType,
		DeviceIP: deviceIP,
		HostIP:   hostIP,
		HostPort: hostPort,
		LDC_ID:   1,
		Enabled:  true,
	}
}

// CreateEquipmentEventData creates a test equipment event data packet
// This simulates the data format that would come from an equipment
func CreateEquipmentEventData(eventID, equipmentID uint32, payloadSize int) []byte {
	data := make([]byte, payloadSize)
	// Fill with pattern based on eventID and equipmentID
	for i := 0; i < payloadSize; i++ {
		data[i] = byte((int(eventID) + int(equipmentID) + i) % 256)
	}
	return data
}

// CreateCompleteSubEventData creates complete subevent data for multiple equipments
// Returns a map of equipmentID -> data
func CreateCompleteSubEventData(eventID int, numEquipments int, dataSize int) map[int][]byte {
	data := make(map[int][]byte)
	for eqID := 1; eqID <= numEquipments; eqID++ {
		data[eqID] = CreateEquipmentEventData(uint32(eventID), uint32(eqID), dataSize)
	}
	return data
}

// CreateUDPPacketWithSequence creates a UDP packet with a sequence counter header
// This matches the format expected by the packet processing code
func CreateUDPPacketWithSequence(sequenceCounter uint32, payload []byte) []byte {
	packet := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(packet[0:4], sequenceCounter)
	copy(packet[4:], payload)
	return packet
}

// CreateMultipleUDPPackets creates a sequence of UDP packets for testing
func CreateMultipleUDPPackets(numPackets int, payloadSize int) [][]byte {
	packets := make([][]byte, numPackets)
	for i := 0; i < numPackets; i++ {
		payload := make([]byte, payloadSize)
		for j := 0; j < payloadSize; j++ {
			payload[j] = byte((i + j) % 256)
		}
		packets[i] = CreateUDPPacketWithSequence(uint32(i), payload)
	}
	return packets
}

// CreateRingBufferData creates test RingBufferData for testing readPacket functions
type RingBufferDataFixture struct {
	Data     []byte
	Position int
}

// CreateTestRingBufferData creates test ring buffer data entries
func CreateTestRingBufferData(numEntries int, packetSize int) []RingBufferDataFixture {
	entries := make([]RingBufferDataFixture, numEntries)
	for i := 0; i < numEntries; i++ {
		data := make([]byte, packetSize)
		for j := 0; j < packetSize; j++ {
			data[j] = byte((i + j) % 256)
		}
		entries[i] = RingBufferDataFixture{
			Data:     data,
			Position: i * packetSize,
		}
	}
	return entries
}
