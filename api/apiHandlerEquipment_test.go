package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"github.com/jmbenlloch/next_duck/pkg/database/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockResult implements sql.Result for testing
type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (m mockResult) LastInsertId() (int64, error) { return m.lastInsertID, nil }
func (m mockResult) RowsAffected() (int64, error) { return m.rowsAffected, nil }

func TestCreateEquipment_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("CreateEquipment", mock.Anything, mock.MatchedBy(func(arg database.CreateEquipmentParams) bool {
		return arg.Type.Int32 == 22 &&
			arg.DeviceIp.String == "192.168.1.1" &&
			arg.HostIp.String == "192.168.1.2" &&
			arg.HostPort.Int32 == 6006 &&
			arg.Ldcid.Int32 == 1 &&
			arg.Enabled.Bool == true
	})).Return(mockResult{lastInsertID: 1, rowsAffected: 1}, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CreateEquipmentRequest{
		Type:     22,
		DeviceIp: "192.168.1.1",
		HostIp:   "192.168.1.2",
		HostPort: 6006,
		LdcId:    1,
		Enabled:  true,
	})

	resp, err := server.CreateEquipment(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "Equipment created successfully", resp.Msg.Message)
	mockQuerier.AssertExpectations(t)
}

func TestCreateEquipment_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("CreateEquipment", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("database connection failed"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CreateEquipmentRequest{
		Type:     22,
		DeviceIp: "192.168.1.1",
		HostIp:   "192.168.1.2",
		HostPort: 6006,
		LdcId:    1,
		Enabled:  true,
	})

	resp, err := server.CreateEquipment(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestGetEquipments_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expectedEquipments := []database.Equipment{
		{
			ID:       1,
			Type:     sql.NullInt32{Int32: 22, Valid: true},
			DeviceIp: sql.NullString{String: "192.168.1.1", Valid: true},
			HostIp:   sql.NullString{String: "192.168.1.2", Valid: true},
			HostPort: sql.NullInt32{Int32: 6006, Valid: true},
			Ldcid:    sql.NullInt32{Int32: 1, Valid: true},
			Enabled:  sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:       2,
			Type:     sql.NullInt32{Int32: 22, Valid: true},
			DeviceIp: sql.NullString{String: "192.168.2.1", Valid: true},
			HostIp:   sql.NullString{String: "192.168.2.2", Valid: true},
			HostPort: sql.NullInt32{Int32: 6006, Valid: true},
			Ldcid:    sql.NullInt32{Int32: 1, Valid: true},
			Enabled:  sql.NullBool{Bool: true, Valid: true},
		},
	}

	mockQuerier.On("ListEquipments", mock.Anything).Return(expectedEquipments, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetEquipmentsRequest{})

	resp, err := server.GetEquipments(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Msg.Equipments, 2)

	assert.Equal(t, int32(1), resp.Msg.Equipments[0].Id)
	assert.Equal(t, int32(22), resp.Msg.Equipments[0].Type)
	assert.Equal(t, "192.168.1.1", resp.Msg.Equipments[0].DeviceIp)
	assert.Equal(t, "192.168.1.2", resp.Msg.Equipments[0].HostIp)
	assert.Equal(t, int32(6006), resp.Msg.Equipments[0].HostPort)
	assert.Equal(t, int32(1), resp.Msg.Equipments[0].LdcId)
	assert.Equal(t, true, resp.Msg.Equipments[0].Enabled)

	assert.Equal(t, int32(2), resp.Msg.Equipments[1].Id)
	assert.Equal(t, int32(22), resp.Msg.Equipments[1].Type)
	assert.Equal(t, "192.168.2.1", resp.Msg.Equipments[1].DeviceIp)
	assert.Equal(t, "192.168.2.2", resp.Msg.Equipments[1].HostIp)
	assert.Equal(t, int32(6006), resp.Msg.Equipments[1].HostPort)
	assert.Equal(t, int32(1), resp.Msg.Equipments[1].LdcId)
	assert.Equal(t, true, resp.Msg.Equipments[1].Enabled)

	mockQuerier.AssertExpectations(t)
}

func TestGetEquipments_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("ListEquipments", mock.Anything).
		Return([]database.Equipment{}, fmt.Errorf("database connection failed"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetEquipmentsRequest{})

	resp, err := server.GetEquipments(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestGetEquipment_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expected := database.Equipment{
		ID:       1,
		Type:     sql.NullInt32{Int32: 22, Valid: true},
		DeviceIp: sql.NullString{String: "192.168.1.1", Valid: true},
		HostIp:   sql.NullString{String: "192.168.1.2", Valid: true},
		HostPort: sql.NullInt32{Int32: 6006, Valid: true},
		Ldcid:    sql.NullInt32{Int32: 1, Valid: true},
		Enabled:  sql.NullBool{Bool: true, Valid: true},
	}

	mockQuerier.On("GetEquipment", mock.Anything, int32(1)).Return(expected, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetEquipmentRequest{Id: 1})

	resp, err := server.GetEquipment(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), resp.Msg.Equipment.Id)
	assert.Equal(t, int32(22), resp.Msg.Equipment.Type)
	assert.Equal(t, "192.168.1.1", resp.Msg.Equipment.DeviceIp)
	assert.Equal(t, "192.168.1.2", resp.Msg.Equipment.HostIp)
	assert.Equal(t, int32(6006), resp.Msg.Equipment.HostPort)
	assert.Equal(t, int32(1), resp.Msg.Equipment.LdcId)
	assert.Equal(t, true, resp.Msg.Equipment.Enabled)
	mockQuerier.AssertExpectations(t)
}

func TestGetEquipment_NotFound(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("GetEquipment", mock.Anything, int32(999)).
		Return(database.Equipment{}, sql.ErrNoRows)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetEquipmentRequest{Id: 999})

	resp, err := server.GetEquipment(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	mockQuerier.AssertExpectations(t)
}

func TestUpdateEquipment_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("UpdateEquipment", mock.Anything, mock.MatchedBy(func(arg database.UpdateEquipmentParams) bool {
		return arg.ID == 1 &&
			arg.Type.Int32 == 23 &&
			arg.DeviceIp.String == "192.168.3.3" &&
			arg.HostIp.String == "192.168.3.4" &&
			arg.HostPort.Int32 == 6007 &&
			arg.Ldcid.Int32 == 2 &&
			arg.Enabled.Bool == false
	})).Return(nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.UpdateEquipmentRequest{
		Id:       1,
		Type:     23,
		DeviceIp: "192.168.3.3",
		HostIp:   "192.168.3.4",
		HostPort: 6007,
		LdcId:    2,
		Enabled:  false,
	})

	resp, err := server.UpdateEquipment(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "Equipment updated successfully", resp.Msg.Message)
	mockQuerier.AssertExpectations(t)
}

func TestUpdateEquipment_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("UpdateEquipment", mock.Anything, mock.Anything).
		Return(fmt.Errorf("database error"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.UpdateEquipmentRequest{
		Id:       1,
		Type:     23,
		DeviceIp: "192.168.3.3",
		HostIp:   "192.168.3.4",
		HostPort: 6007,
		LdcId:    2,
		Enabled:  false,
	})

	resp, err := server.UpdateEquipment(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestDeleteEquipment_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("DeleteEquipment", mock.Anything, int32(1)).Return(nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.DeleteEquipmentRequest{Id: 1})

	resp, err := server.DeleteEquipment(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "Equipment deleted successfully", resp.Msg.Message)
	mockQuerier.AssertExpectations(t)
}

func TestDeleteEquipment_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("DeleteEquipment", mock.Anything, int32(1)).
		Return(fmt.Errorf("database error"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.DeleteEquipmentRequest{Id: 1})

	resp, err := server.DeleteEquipment(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}
