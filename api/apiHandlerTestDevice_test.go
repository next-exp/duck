package main

import (
	"context"
	"database/sql"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStartTestDevices_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "/nonexistent/config.yml", "", false)
	req := connect.NewRequest(&pb.StartTestDevicesRequest{})

	// Mock ListLDCs to return an error
	mockQuerier.On("ListLDCs", mock.Anything).
		Return([]database.Ldc{}, sql.ErrConnDone)

	resp, err := server.StartTestDevices(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestStartTestDevices_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.StartTestDevicesRequest{})

	// Mock ListLDCs to return one enabled LDC
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{
		{ID: 1, Enabled: sql.NullBool{Bool: true, Valid: true}},
	}, nil)

	// Mock ListEquipments to return one enabled equipment
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{
		{
			ID:      1,
			Ldcid:   sql.NullInt32{Int32: 1, Valid: true},
			Enabled: sql.NullBool{Bool: true, Valid: true},
		},
	}, nil)

	// The handler will try to connect to the device and fail, but returns a response with Success=false
	// instead of returning an error
	resp, err := server.StartTestDevices(context.Background(), req)

	// Note: This will fail to connect to the actual device simulator, but we're testing
	// that the database calls work correctly and the logic is correct
	// The handler collects errors and returns Success=false in the response
	require.NoError(t, err) // No error returned, just a response with Success=false
	require.NotNil(t, resp)
	assert.False(t, resp.Msg.Success, "Response should indicate failure due to connection error")
	assert.Contains(t, resp.Msg.Message, "failed to start", "Message should mention device startup failure")

	mockQuerier.AssertExpectations(t)
}

func TestStopTestDevices_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "/nonexistent/config.yml", "", false)
	req := connect.NewRequest(&pb.StopTestDevicesRequest{})

	// Mock ListLDCs to return an error
	mockQuerier.On("ListLDCs", mock.Anything).
		Return([]database.Ldc{}, sql.ErrConnDone)

	resp, err := server.StopTestDevices(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestGetTestDevicesStates_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "/nonexistent/config.yml", "", false)
	req := connect.NewRequest(&pb.GetTestDevicesStatesRequest{})

	// Mock ListLDCs to return an error
	mockQuerier.On("ListLDCs", mock.Anything).
		Return([]database.Ldc{}, sql.ErrConnDone)

	resp, err := server.GetTestDevicesStates(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestListTestDevices_DatabaseErrorOnLDCs(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "/nonexistent/config.yml", "", false)
	req := connect.NewRequest(&pb.ListTestDevicesRequest{})

	// Mock ListLDCs to return an error
	mockQuerier.On("ListLDCs", mock.Anything).
		Return([]database.Ldc{}, sql.ErrConnDone)

	resp, err := server.ListTestDevices(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestListTestDevices_DatabaseErrorOnEquipments(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "/nonexistent/config.yml", "", false)
	req := connect.NewRequest(&pb.ListTestDevicesRequest{})

	// Mock ListLDCs to succeed
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{
		{ID: 1, Enabled: sql.NullBool{Bool: true, Valid: true}},
	}, nil)

	// Mock ListEquipments to return an error
	mockQuerier.On("ListEquipments", mock.Anything).
		Return([]database.Equipment{}, sql.ErrConnDone)

	resp, err := server.ListTestDevices(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestListTestDevices_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.ListTestDevicesRequest{})

	// Mock ListLDCs
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{
		{
			ID:      1,
			Name:    sql.NullString{String: "ldc1", Valid: true},
			Enabled: sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:      2,
			Name:    sql.NullString{String: "ldc2", Valid: true},
			Enabled: sql.NullBool{Bool: true, Valid: true},
		},
	}, nil)

	// Mock ListEquipments
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{
		{
			ID:        1,
			Type:      sql.NullInt32{Int32: 1, Valid: true},
			DeviceIp:  sql.NullString{String: "192.168.1.1", Valid: true},
			HostIp:    sql.NullString{String: "10.0.1.1", Valid: true},
			HostPort:  sql.NullInt32{Int32: 9001, Valid: true},
			Ldcid:     sql.NullInt32{Int32: 1, Valid: true},
			Enabled:   sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:        2,
			Type:      sql.NullInt32{Int32: 2, Valid: true},
			DeviceIp:  sql.NullString{String: "192.168.2.1", Valid: true},
			HostIp:    sql.NullString{String: "10.0.2.1", Valid: true},
			HostPort:  sql.NullInt32{Int32: 9002, Valid: true},
			Ldcid:     sql.NullInt32{Int32: 1, Valid: true},
			Enabled:   sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:        3,
			Type:      sql.NullInt32{Int32: 1, Valid: true},
			DeviceIp:  sql.NullString{String: "192.168.3.1", Valid: true},
			HostIp:    sql.NullString{String: "10.0.3.1", Valid: true},
			HostPort:  sql.NullInt32{Int32: 9003, Valid: true},
			Ldcid:     sql.NullInt32{Int32: 2, Valid: true},
			Enabled:   sql.NullBool{Bool: true, Valid: true},
		},
	}, nil)

	// Mock GetSimulatorParams for equipment 1
	mockQuerier.On("GetSimulatorParams", mock.Anything, int32(1)).
		Return(database.Simulatorparam{
			Equipmentid: 1,
			Replayrate:  10.5,
			Maxevents:   1000,
			Packetsize:  1024,
		}, nil)

	// Mock GetSimulatorParams for equipment 2 (not found)
	mockQuerier.On("GetSimulatorParams", mock.Anything, int32(2)).
		Return(database.Simulatorparam{}, sql.ErrNoRows)

	// Mock GetSimulatorParams for equipment 3 (not found)
	mockQuerier.On("GetSimulatorParams", mock.Anything, int32(3)).
		Return(database.Simulatorparam{}, sql.ErrNoRows)

	resp, err := server.ListTestDevices(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	// Check that we got 3 devices
	require.Len(t, resp.Msg.Devices, 3)

	// Check first device
	dev1 := resp.Msg.Devices[0]
	assert.Equal(t, int32(1), dev1.EquipmentId)
	assert.Equal(t, "device-1", dev1.DeviceName)
	assert.Equal(t, "192.168.1.1", dev1.HostIp)
	assert.Equal(t, int32(9001), dev1.HostPort)
	assert.Equal(t, int32(50070), dev1.GrpcPort)
	assert.Equal(t, int32(12130), dev1.PrometheusPort)
	assert.Equal(t, float64(10.5), dev1.RateHz)
	assert.Equal(t, int32(1024), dev1.PacketSize)
	assert.Equal(t, int32(1000), dev1.MaxEvents)
	assert.True(t, dev1.Enabled)

	// Check second device (no simulator params, should have defaults)
	dev2 := resp.Msg.Devices[1]
	assert.Equal(t, int32(2), dev2.EquipmentId)
	assert.Equal(t, "device-2", dev2.DeviceName)
	assert.Equal(t, "192.168.2.1", dev2.HostIp)
	assert.Equal(t, int32(9002), dev2.HostPort)
	assert.Equal(t, float64(0.1), dev2.RateHz)    // default value
	assert.Equal(t, int32(500), dev2.PacketSize)   // default value
	assert.Equal(t, int32(0), dev2.MaxEvents)      // default value

	// Check third device
	dev3 := resp.Msg.Devices[2]
	assert.Equal(t, int32(3), dev3.EquipmentId)
	assert.Equal(t, "device-3", dev3.DeviceName)
	assert.Equal(t, "192.168.3.1", dev3.HostIp)

	mockQuerier.AssertExpectations(t)
}

func TestGetTestDeviceStatistics_DatabaseErrorOnLDCs(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "/nonexistent/config.yml", "", false)
	req := connect.NewRequest(&pb.GetTestDeviceStatisticsRequest{DeviceId: 1})

	// Mock ListLDCs to return an error
	mockQuerier.On("ListLDCs", mock.Anything).
		Return([]database.Ldc{}, sql.ErrConnDone)

	resp, err := server.GetTestDeviceStatistics(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestGetTestDeviceStatistics_DeviceNotFound(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetTestDeviceStatisticsRequest{DeviceId: 999})

	// Mock ListLDCs
	mockQuerier.On("ListLDCs", mock.Anything).Return([]database.Ldc{
		{ID: 1, Enabled: sql.NullBool{Bool: true, Valid: true}},
	}, nil)

	// Mock ListEquipments (device 999 not in list)
	mockQuerier.On("ListEquipments", mock.Anything).Return([]database.Equipment{
		{
			ID:      1,
			Ldcid:   sql.NullInt32{Int32: 1, Valid: true},
			Enabled: sql.NullBool{Bool: true, Valid: true},
		},
	}, nil)

	resp, err := server.GetTestDeviceStatistics(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "device not found")
	mockQuerier.AssertExpectations(t)
}
