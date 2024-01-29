package main

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jmbenlloch/next_duck/ldcRPC/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

// MockPingerFactoryAdapter adapts testhelpers.MockPingerFactory to implement PingerFactory
type MockPingerFactoryAdapter struct {
	factory *testhelpers.MockPingerFactory
}

func (a *MockPingerFactoryAdapter) NewPinger(ip string) (Pinger, error) {
	pinger, err := a.factory.NewPinger(ip)
	if err != nil {
		return nil, err
	}
	return pinger.(Pinger), nil
}

// Unit tests for pingDevices function using mock pingers
// These tests don't require network access and cover all code paths

func TestPingDevices_AllDisabled_ReturnsTrue(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - all equipment disabled, should return true without pinging
	equipments := []duck.Equipment{
		testhelpers.NewTestEquipmentDisabled(1, 22),
		testhelpers.NewTestEquipmentDisabled(2, 22),
	}

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should return true since no enabled devices to ping
	assert.True(t, result)
}

func TestPingDevices_EmptyList_ReturnsTrue(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - empty equipment list
	equipments := []duck.Equipment{}

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should return true since nothing to ping
	assert.True(t, result)
}

func TestPingDevices_NilList_ReturnsTrue(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - nil equipment list
	var equipments []duck.Equipment = nil

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should return true since nothing to ping
	assert.True(t, result)
}

func TestPingDevices_MixedEnabledDisabled_SkipsDisabled(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - disabled and enabled equipment
	// Disabled should be skipped, enabled should be pinged
	equipments := []duck.Equipment{
		{
			ID:       1,
			Type:     22,
			DeviceIP: "127.0.0.1",
			HostIP:   "127.0.0.1",
			HostPort: 6006,
			LDC_ID:   1,
			Enabled:  false, // DISABLED - should be skipped
		},
		{
			ID:       2,
			Type:     22,
			DeviceIP: "192.168.1.1",
			HostIP:   "127.0.0.1",
			HostPort: 6007,
			LDC_ID:   1,
			Enabled:  true,
		},
	}

	mockPinger.SetPingSuccess("192.168.1.1", true)

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should succeed because only enabled device is pinged and succeeds
	assert.True(t, result, "Should succeed when enabled device ping succeeds")
}

func TestPingDevices_SingleDeviceSuccessfulPing_ReturnsTrue(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - single enabled device
	equipments := []duck.Equipment{
		{
			ID:       1,
			Type:     22,
			DeviceIP: "192.168.1.1",
			HostIP:   "127.0.0.1",
			HostPort: 6006,
			LDC_ID:   1,
			Enabled:  true,
		},
	}

	mockPinger.SetPingSuccess("192.168.1.1", true)

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should succeed with successful ping
	assert.True(t, result, "Should succeed with successful ping")

	// Verify configuration was set correctly
	mockAdapter := mockFactory.GetLastCreatedAdapter()
	assert.Equal(t, 2, mockAdapter.GetCount(), "Count should be set to 2")
	assert.Equal(t, 500*time.Millisecond, mockAdapter.GetTimeout(), "Timeout should be set to 500ms")
}

func TestPingDevices_NoPacketsReceived_ReturnsFalse(t *testing.T) {
	logger, logBuffer := testhelpers.SetupTestWithCapture()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - enabled device where ping returns 0 packets
	equipments := []duck.Equipment{
		{
			ID:       1,
			Type:     22,
			DeviceIP: "192.168.1.1",
			HostIP:   "127.0.0.1",
			HostPort: 6006,
			LDC_ID:   1,
			Enabled:  true,
		},
	}

	mockPinger.SetPacketsRecv("192.168.1.1", 0)

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should fail when no packets received
	assert.False(t, result, "Should fail when no packets received")

	// Verify error was logged
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "did not respond to ping request", "Error message should be logged")
	assert.Contains(t, logOutput, "192.168.1.1", "Device IP should be in error log")
}

func TestPingDevices_PartialPacketsReceived_ReturnsTrue(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - enabled device where ping returns some packets but less than expected
	equipments := []duck.Equipment{
		{
			ID:       1,
			Type:     22,
			DeviceIP: "192.168.1.1",
			HostIP:   "127.0.0.1",
			HostPort: 6006,
			LDC_ID:   1,
			Enabled:  true,
		},
	}

	// Set 1 packet received (less than Count=2, but still >= 1 so should succeed)
	mockPinger.SetPacketsRecv("192.168.1.1", 1)

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should succeed because we only check if packets >= 1
	assert.True(t, result, "Should succeed with at least 1 packet received")
}

func TestPingDevices_NewPingerError_ReturnsFalse(t *testing.T) {
	logger, logBuffer := testhelpers.SetupTestWithCapture()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - enabled device where NewPinger fails
	equipments := []duck.Equipment{
		{
			ID:       1,
			Type:     22,
			DeviceIP: "invalid-ip",
			HostIP:   "127.0.0.1",
			HostPort: 6006,
			LDC_ID:   1,
			Enabled:  true,
		},
	}

	mockPinger.SetPingError("invalid-ip", errors.New("invalid IP address"))

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should fail when NewPinger returns error
	assert.False(t, result, "Should fail when NewPinger returns error")

	// Verify error was logged
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "invalid IP address", "Error message should be logged")
}

func TestPingDevices_RunError_ReturnsFalse(t *testing.T) {
	logger, logBuffer := testhelpers.SetupTestWithCapture()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - enabled device where Run() fails
	equipments := []duck.Equipment{
		{
			ID:       1,
			Type:     22,
			DeviceIP: "192.168.1.1",
			HostIP:   "127.0.0.1",
			HostPort: 6006,
			LDC_ID:   1,
			Enabled:  true,
		},
	}

	mockPinger.SetRunError("192.168.1.1", errors.New("ping execution failed"))

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should fail when Run() returns error
	assert.False(t, result, "Should fail when Run() returns error")

	// Verify error was logged
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "ping execution failed", "Error message should be logged")
}

func TestPingDevices_MultipleDevicesAllSucceed_ReturnsTrue(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - multiple enabled devices
	equipments := []duck.Equipment{
		{
			ID:       1,
			Type:     22,
			DeviceIP: "192.168.1.1",
			HostIP:   "127.0.0.1",
			HostPort: 6006,
			LDC_ID:   1,
			Enabled:  true,
		},
		{
			ID:       2,
			Type:     22,
			DeviceIP: "192.168.1.2",
			HostIP:   "127.0.0.1",
			HostPort: 6007,
			LDC_ID:   1,
			Enabled:  true,
		},
		{
			ID:       3,
			Type:     22,
			DeviceIP: "192.168.1.3",
			HostIP:   "127.0.0.1",
			HostPort: 6008,
			LDC_ID:   1,
			Enabled:  true,
		},
	}

	mockPinger.SetPingSuccess("192.168.1.1", true)
	mockPinger.SetPingSuccess("192.168.1.2", true)
	mockPinger.SetPingSuccess("192.168.1.3", true)

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should succeed when all devices ping successfully
	assert.True(t, result, "Should succeed when all devices ping successfully")
}

func TestPingDevices_MultipleDevicesFirstFails_ReturnsFalse(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - multiple enabled devices, first one fails
	equipments := []duck.Equipment{
		{
			ID:       1,
			Type:     22,
			DeviceIP: "192.168.1.1",
			HostIP:   "127.0.0.1",
			HostPort: 6006,
			LDC_ID:   1,
			Enabled:  true,
		},
		{
			ID:       2,
			Type:     22,
			DeviceIP: "192.168.1.2",
			HostIP:   "127.0.0.1",
			HostPort: 6007,
			LDC_ID:   1,
			Enabled:  true,
		},
	}

	mockPinger.SetPingSuccess("192.168.1.1", false)
	mockPinger.SetPingSuccess("192.168.1.2", true)

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should fail when first device fails (short-circuit)
	assert.False(t, result, "Should fail when first device fails")
}

func TestPingDevices_MultipleDevicesSecondFails_ReturnsFalse(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - multiple enabled devices, second one fails
	equipments := []duck.Equipment{
		{
			ID:       1,
			Type:     22,
			DeviceIP: "192.168.1.1",
			HostIP:   "127.0.0.1",
			HostPort: 6006,
			LDC_ID:   1,
			Enabled:  true,
		},
		{
			ID:       2,
			Type:     22,
			DeviceIP: "192.168.1.2",
			HostIP:   "127.0.0.1",
			HostPort: 6007,
			LDC_ID:   1,
			Enabled:  true,
		},
	}

	mockPinger.SetPingSuccess("192.168.1.1", true)
	mockPinger.SetPingSuccess("192.168.1.2", false)

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should fail when second device fails
	assert.False(t, result, "Should fail when second device fails")
}

func TestPingDevices_MultipleDevicesMixedEnabledDisabled_ReturnsTrue(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - mixed enabled and disabled devices
	equipments := []duck.Equipment{
		{
			ID:       1,
			Type:     22,
			DeviceIP: "192.168.1.1",
			HostIP:   "127.0.0.1",
			HostPort: 6006,
			LDC_ID:   1,
			Enabled:  true,
		},
		{
			ID:       2,
			Type:     22,
			DeviceIP: "192.168.1.2",
			HostIP:   "127.0.0.1",
			HostPort: 6007,
			LDC_ID:   1,
			Enabled:  false, // Disabled - should be skipped
		},
		{
			ID:       3,
			Type:     22,
			DeviceIP: "192.168.1.3",
			HostIP:   "127.0.0.1",
			HostPort: 6008,
			LDC_ID:   1,
			Enabled:  true,
		},
	}

	mockPinger.SetPingSuccess("192.168.1.1", true)
	mockPinger.SetPingSuccess("192.168.1.3", true)

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should succeed when all enabled devices ping successfully
	assert.True(t, result, "Should succeed when all enabled devices ping successfully")
}

func TestPingDevices_MultipleDisabled_ReturnsTrue(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - many disabled equipments
	equipments := []duck.Equipment{
		testhelpers.NewTestEquipmentDisabled(1, 22),
		testhelpers.NewTestEquipmentDisabled(2, 22),
		testhelpers.NewTestEquipmentDisabled(3, 22),
		testhelpers.NewTestEquipmentDisabled(4, 22),
		testhelpers.NewTestEquipmentDisabled(5, 22),
	}

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should return true since all are disabled
	assert.True(t, result)
}

func TestPingDevices_SingleDisabled_ReturnsTrue(t *testing.T) {
	logger := testhelpers.SetupTest()
	metrics := NewMetricsRegistry()

	s := &server{
		logger:  logger,
		metrics: metrics,
	}

	mockPinger := testhelpers.NewMockPinger()
	mockFactory := testhelpers.NewMockPingerFactory(mockPinger)
	factory := &MockPingerFactoryAdapter{factory: mockFactory}

	// Setup - single disabled equipment
	equipments := []duck.Equipment{
		testhelpers.NewTestEquipmentDisabled(1, 22),
	}

	// Act
	result := pingDevices(s, factory, equipments)

	// Assert - should return true
	assert.True(t, result)
}
