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

// TestCreateLDC_Success tests adding an LDC successfully
func TestCreateLDC_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("CreateLDC", mock.Anything, mock.MatchedBy(func(arg database.CreateLDCParams) bool {
		return arg.Name.String == "test_ldc" &&
			arg.Hostname.String == "testhost" &&
			arg.Ip.String == "127.0.0.1" &&
			arg.Enabled.Bool == true &&
			arg.GrpcPort.Int32 == 50052 &&
			arg.PrometheusPort.Int32 == 12113
	})).Return(mockResult{lastInsertID: 1, rowsAffected: 1}, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CreateLDCRequest{
		Name:           "test_ldc",
		Hostname:       "testhost",
		Ip:             "127.0.0.1",
		Enabled:        true,
		GrpcPort:       50052,
		PrometheusPort: 12113,
	})

	resp, err := server.CreateLDC(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "LDC created successfully", resp.Msg.Message)
	mockQuerier.AssertExpectations(t)
}

// TestGetLDCs_Success tests retrieving all LDCs
func TestGetLDCs_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expectedLDCs := []database.Ldc{
		{
			ID:             1,
			Name:           sql.NullString{String: "ldc1", Valid: true},
			Hostname:       sql.NullString{String: "host1", Valid: true},
			Ip:             sql.NullString{String: "192.168.1.1", Valid: true},
			Enabled:        sql.NullBool{Bool: true, Valid: true},
			GrpcPort:       sql.NullInt32{Int32: 50052, Valid: true},
			PrometheusPort: sql.NullInt32{Int32: 12113, Valid: true},
		},
		{
			ID:             2,
			Name:           sql.NullString{String: "ldc2", Valid: true},
			Hostname:       sql.NullString{String: "host2", Valid: true},
			Ip:             sql.NullString{String: "192.168.1.2", Valid: true},
			Enabled:        sql.NullBool{Bool: true, Valid: true},
			GrpcPort:       sql.NullInt32{Int32: 50053, Valid: true},
			PrometheusPort: sql.NullInt32{Int32: 12114, Valid: true},
		},
	}

	expectedEquipments := []database.Equipment{
		{
			ID:       1,
			Type:     sql.NullInt32{Int32: 22, Valid: true},
			DeviceIp: sql.NullString{String: "10.0.1.1", Valid: true},
			HostIp:   sql.NullString{String: "10.0.1.2", Valid: true},
			HostPort: sql.NullInt32{Int32: 6006, Valid: true},
			Ldcid:    sql.NullInt32{Int32: 1, Valid: true},
			Enabled:  sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:       2,
			Type:     sql.NullInt32{Int32: 22, Valid: true},
			DeviceIp: sql.NullString{String: "10.0.2.1", Valid: true},
			HostIp:   sql.NullString{String: "10.0.2.2", Valid: true},
			HostPort: sql.NullInt32{Int32: 6006, Valid: true},
			Ldcid:    sql.NullInt32{Int32: 2, Valid: true},
			Enabled:  sql.NullBool{Bool: true, Valid: true},
		},
	}

	mockQuerier.On("ListLDCs", mock.Anything).Return(expectedLDCs, nil)
	mockQuerier.On("ListEquipments", mock.Anything).Return(expectedEquipments, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetLDCsRequest{})

	resp, err := server.GetLDCs(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Msg.Ldcs, 2)

	// Check first LDC
	assert.Equal(t, int32(1), resp.Msg.Ldcs[0].Id)
	assert.Equal(t, "ldc1", resp.Msg.Ldcs[0].Name)
	assert.Equal(t, "host1", resp.Msg.Ldcs[0].Hostname)
	assert.Equal(t, "192.168.1.1", resp.Msg.Ldcs[0].Ip)
	assert.Equal(t, true, resp.Msg.Ldcs[0].Enabled)
	assert.Equal(t, int32(50052), resp.Msg.Ldcs[0].GrpcPort)
	assert.Equal(t, int32(12113), resp.Msg.Ldcs[0].PrometheusPort)

	// Check equipments for first LDC
	assert.Len(t, resp.Msg.Ldcs[0].Equipments, 1)
	if len(resp.Msg.Ldcs[0].Equipments) > 0 {
		eq := resp.Msg.Ldcs[0].Equipments[0]
		assert.Equal(t, int32(1), eq.Id)
		assert.Equal(t, int32(22), eq.Type)
		assert.Equal(t, "10.0.1.1", eq.DeviceIp)
		assert.Equal(t, "10.0.1.2", eq.HostIp)
		assert.Equal(t, int32(6006), eq.HostPort)
		assert.Equal(t, int32(1), eq.LdcId)
		assert.Equal(t, true, eq.Enabled)
	}

	// Check second LDC
	assert.Equal(t, int32(2), resp.Msg.Ldcs[1].Id)
	assert.Equal(t, "ldc2", resp.Msg.Ldcs[1].Name)
	assert.Equal(t, "host2", resp.Msg.Ldcs[1].Hostname)
	assert.Equal(t, "192.168.1.2", resp.Msg.Ldcs[1].Ip)
	assert.Equal(t, true, resp.Msg.Ldcs[1].Enabled)
	assert.Equal(t, int32(50053), resp.Msg.Ldcs[1].GrpcPort)
	assert.Equal(t, int32(12114), resp.Msg.Ldcs[1].PrometheusPort)

	// Check equipments for second LDC
	assert.Len(t, resp.Msg.Ldcs[1].Equipments, 1)
	if len(resp.Msg.Ldcs[1].Equipments) > 0 {
		eq := resp.Msg.Ldcs[1].Equipments[0]
		assert.Equal(t, int32(2), eq.Id)
		assert.Equal(t, int32(22), eq.Type)
		assert.Equal(t, "10.0.2.1", eq.DeviceIp)
		assert.Equal(t, "10.0.2.2", eq.HostIp)
		assert.Equal(t, int32(6006), eq.HostPort)
		assert.Equal(t, int32(2), eq.LdcId)
		assert.Equal(t, true, eq.Enabled)
	}

	mockQuerier.AssertExpectations(t)
}

func TestGetLDCs_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("ListLDCs", mock.Anything).
		Return([]database.Ldc{}, fmt.Errorf("database connection failed"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetLDCsRequest{})

	resp, err := server.GetLDCs(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

// TestGetLDC_Success tests retrieving a single LDC by ID
func TestGetLDC_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expected := database.Ldc{
		ID:             1,
		Name:           sql.NullString{String: "ldc1", Valid: true},
		Hostname:       sql.NullString{String: "host1", Valid: true},
		Ip:             sql.NullString{String: "192.168.1.1", Valid: true},
		Enabled:        sql.NullBool{Bool: true, Valid: true},
		GrpcPort:       sql.NullInt32{Int32: 50052, Valid: true},
		PrometheusPort: sql.NullInt32{Int32: 12113, Valid: true},
	}

	mockQuerier.On("GetLDC", mock.Anything, int32(1)).Return(expected, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetLDCRequest{Id: 1})

	resp, err := server.GetLDC(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), resp.Msg.Ldc.Id)
	assert.Equal(t, "ldc1", resp.Msg.Ldc.Name)
	assert.Equal(t, "host1", resp.Msg.Ldc.Hostname)
	assert.Equal(t, "192.168.1.1", resp.Msg.Ldc.Ip)
	assert.Equal(t, true, resp.Msg.Ldc.Enabled)
	assert.Equal(t, int32(50052), resp.Msg.Ldc.GrpcPort)
	assert.Equal(t, int32(12113), resp.Msg.Ldc.PrometheusPort)
	mockQuerier.AssertExpectations(t)
}

func TestGetLDC_NotFound(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("GetLDC", mock.Anything, int32(999)).
		Return(database.Ldc{}, sql.ErrNoRows)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetLDCRequest{Id: 999})

	resp, err := server.GetLDC(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	mockQuerier.AssertExpectations(t)
}

// TestDeleteLDC_Success tests deleting an LDC successfully
func TestDeleteLDC_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("DeleteLDC", mock.Anything, int32(1)).Return(nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.DeleteLDCRequest{Id: 1})

	resp, err := server.DeleteLDC(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "LDC deleted successfully", resp.Msg.Message)
	mockQuerier.AssertExpectations(t)
}

func TestDeleteLDC_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("DeleteLDC", mock.Anything, int32(1)).
		Return(fmt.Errorf("database error"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.DeleteLDCRequest{Id: 1})

	resp, err := server.DeleteLDC(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

// TestUpdateLDC_Success tests updating an LDC successfully
func TestUpdateLDC_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("UpdateLDC", mock.Anything, mock.MatchedBy(func(arg database.UpdateLDCParams) bool {
		return arg.ID == 1 &&
			arg.Name.String == "updated_ldc" &&
			arg.Hostname.String == "updatedhost" &&
			arg.Ip.String == "127.0.0.2" &&
			arg.Enabled.Bool == false &&
			arg.GrpcPort.Int32 == 50054 &&
			arg.PrometheusPort.Int32 == 12115
	})).Return(nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.UpdateLDCRequest{
		Id:             1,
		Name:           "updated_ldc",
		Hostname:       "updatedhost",
		Ip:             "127.0.0.2",
		Enabled:        false,
		GrpcPort:       50054,
		PrometheusPort: 12115,
	})

	resp, err := server.UpdateLDC(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "LDC updated successfully", resp.Msg.Message)
	mockQuerier.AssertExpectations(t)
}

func TestUpdateLDC_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("UpdateLDC", mock.Anything, mock.Anything).
		Return(fmt.Errorf("database error"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.UpdateLDCRequest{
		Id:             1,
		Name:           "updated_ldc",
		Hostname:       "updatedhost",
		Ip:             "127.0.0.2",
		Enabled:        false,
		GrpcPort:       50054,
		PrometheusPort: 12115,
	})

	resp, err := server.UpdateLDC(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

// TestCreateLDC_DatabaseError tests adding an LDC with database error
func TestCreateLDC_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("CreateLDC", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("database connection failed"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CreateLDCRequest{
		Name:           "test_ldc",
		Hostname:       "testhost",
		Ip:             "127.0.0.1",
		Enabled:        true,
		GrpcPort:       50052,
		PrometheusPort: 12113,
	})

	resp, err := server.CreateLDC(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}
