package testhelpers

import (
	"encoding/binary"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

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

// CreateLDCEventData creates a mock LDC event data payload with proper header
func CreateLDCEventData(eventID int, ldcID int, dataSize int) []byte {
	totalSize := 80 + dataSize // 80 bytes for header
	data := make([]byte, totalSize)

	// Event header (simplified version)
	binary.LittleEndian.PutUint32(data[0:4], uint32(totalSize))                    // Event size
	binary.LittleEndian.PutUint32(data[4:8], uint32(duck.EVENT_MAGIC_NUMBER))      // Magic number
	binary.LittleEndian.PutUint32(data[8:12], 80)                                  // Header size
	binary.LittleEndian.PutUint32(data[12:16], uint32(duck.EVENT_CURRENT_VERSION)) // Version
	binary.LittleEndian.PutUint32(data[16:20], uint32(duck.PHYSICS_EVENT))         // Event type
	binary.LittleEndian.PutUint32(data[20:24], 123)                                // Run number
	binary.LittleEndian.PutUint32(data[24:28], uint32(eventID))                    // Event ID
	binary.LittleEndian.PutUint32(data[64:68], uint32(ldcID))                      // LDC ID

	// Fill payload with test data
	for i := 80; i < totalSize; i++ {
		data[i] = byte(i % 256)
	}

	return data
}
