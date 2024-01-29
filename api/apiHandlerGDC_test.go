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

func TestCreateGDC_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("CreateGDC", mock.Anything, mock.MatchedBy(func(arg database.CreateGDCParams) bool {
		return arg.Name.String == "test_gdc" &&
			arg.Hostname.String == "testhost" &&
			arg.Ip.String == "127.0.0.1" &&
			arg.Port.Int32 == 6005 &&
			arg.Datapath.String == "/tmp/test" &&
			arg.Enabled.Bool == true &&
			arg.Writeoutput.Bool == true &&
			arg.Decode.Bool == false &&
			arg.GrpcPort.Int32 == 50051 &&
			arg.PrometheusPort.Int32 == 12112
	})).Return(mockResult{lastInsertID: 1, rowsAffected: 1}, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CreateGDCRequest{
		Name:           "test_gdc",
		Hostname:       "testhost",
		Ip:             "127.0.0.1",
		Port:           6005,
		Datapath:       "/tmp/test",
		Enabled:        true,
		WriteOutput:    true,
		Decode:         false,
		GrpcPort:       50051,
		PrometheusPort: 12112,
	})

	resp, err := server.CreateGDC(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "GDC created successfully", resp.Msg.Message)
	mockQuerier.AssertExpectations(t)
}

func TestCreateGDC_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("CreateGDC", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("database connection failed"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CreateGDCRequest{
		Name:           "test_gdc",
		Hostname:       "testhost",
		Ip:             "127.0.0.1",
		Port:           6005,
		Datapath:       "/tmp/test",
		Enabled:        true,
		WriteOutput:    true,
		Decode:         false,
		GrpcPort:       50051,
		PrometheusPort: 12112,
	})

	resp, err := server.CreateGDC(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestGetGDCs_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expectedGDCs := []database.Gdc{
		{
			ID:             1,
			Name:           sql.NullString{String: "gdc1", Valid: true},
			Hostname:       sql.NullString{String: "host1", Valid: true},
			Ip:             sql.NullString{String: "192.168.1.1", Valid: true},
			Port:           sql.NullInt32{Int32: 6005, Valid: true},
			Datapath:       sql.NullString{String: "/tmp/gdc1", Valid: true},
			Enabled:        sql.NullBool{Bool: true, Valid: true},
			Writeoutput:    sql.NullBool{Bool: true, Valid: true},
			Decode:         sql.NullBool{Bool: false, Valid: true},
			GrpcPort:       sql.NullInt32{Int32: 50051, Valid: true},
			PrometheusPort: sql.NullInt32{Int32: 12112, Valid: true},
		},
		{
			ID:             2,
			Name:           sql.NullString{String: "gdc2", Valid: true},
			Hostname:       sql.NullString{String: "host2", Valid: true},
			Ip:             sql.NullString{String: "192.168.1.2", Valid: true},
			Port:           sql.NullInt32{Int32: 6005, Valid: true},
			Datapath:       sql.NullString{String: "/tmp/gdc2", Valid: true},
			Enabled:        sql.NullBool{Bool: true, Valid: true},
			Writeoutput:    sql.NullBool{Bool: true, Valid: true},
			Decode:         sql.NullBool{Bool: false, Valid: true},
			GrpcPort:       sql.NullInt32{Int32: 50052, Valid: true},
			PrometheusPort: sql.NullInt32{Int32: 12113, Valid: true},
		},
	}

	mockQuerier.On("ListGDCs", mock.Anything).Return(expectedGDCs, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetGDCsRequest{})

	resp, err := server.GetGDCs(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Msg.Gdcs, 2)

	assert.Equal(t, int32(1), resp.Msg.Gdcs[0].Id)
	assert.Equal(t, "gdc1", resp.Msg.Gdcs[0].Name)
	assert.Equal(t, "host1", resp.Msg.Gdcs[0].Hostname)
	assert.Equal(t, "192.168.1.1", resp.Msg.Gdcs[0].Ip)
	assert.Equal(t, int32(6005), resp.Msg.Gdcs[0].Port)
	assert.Equal(t, "/tmp/gdc1", resp.Msg.Gdcs[0].Datapath)
	assert.Equal(t, true, resp.Msg.Gdcs[0].Enabled)
	assert.Equal(t, true, resp.Msg.Gdcs[0].WriteOutput)
	assert.Equal(t, false, resp.Msg.Gdcs[0].Decode)
	assert.Equal(t, int32(50051), resp.Msg.Gdcs[0].GrpcPort)
	assert.Equal(t, int32(12112), resp.Msg.Gdcs[0].PrometheusPort)

	assert.Equal(t, int32(2), resp.Msg.Gdcs[1].Id)
	assert.Equal(t, "gdc2", resp.Msg.Gdcs[1].Name)
	assert.Equal(t, "host2", resp.Msg.Gdcs[1].Hostname)
	assert.Equal(t, "192.168.1.2", resp.Msg.Gdcs[1].Ip)
	assert.Equal(t, int32(6005), resp.Msg.Gdcs[1].Port)
	assert.Equal(t, "/tmp/gdc2", resp.Msg.Gdcs[1].Datapath)
	assert.Equal(t, true, resp.Msg.Gdcs[1].Enabled)
	assert.Equal(t, true, resp.Msg.Gdcs[1].WriteOutput)
	assert.Equal(t, false, resp.Msg.Gdcs[1].Decode)
	assert.Equal(t, int32(50052), resp.Msg.Gdcs[1].GrpcPort)
	assert.Equal(t, int32(12113), resp.Msg.Gdcs[1].PrometheusPort)

	mockQuerier.AssertExpectations(t)
}

func TestGetGDCs_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("ListGDCs", mock.Anything).
		Return([]database.Gdc{}, fmt.Errorf("database connection failed"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetGDCsRequest{})

	resp, err := server.GetGDCs(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestGetGDC_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expected := database.Gdc{
		ID:             1,
		Name:           sql.NullString{String: "gdc1", Valid: true},
		Hostname:       sql.NullString{String: "host1", Valid: true},
		Ip:             sql.NullString{String: "192.168.1.1", Valid: true},
		Port:           sql.NullInt32{Int32: 6005, Valid: true},
		Datapath:       sql.NullString{String: "/tmp/gdc1", Valid: true},
		Enabled:        sql.NullBool{Bool: true, Valid: true},
		Writeoutput:    sql.NullBool{Bool: true, Valid: true},
		Decode:         sql.NullBool{Bool: false, Valid: true},
		GrpcPort:       sql.NullInt32{Int32: 50051, Valid: true},
		PrometheusPort: sql.NullInt32{Int32: 12112, Valid: true},
	}

	mockQuerier.On("GetGDC", mock.Anything, int32(1)).Return(expected, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetGDCRequest{Id: 1})

	resp, err := server.GetGDC(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(1), resp.Msg.Gdc.Id)
	assert.Equal(t, "gdc1", resp.Msg.Gdc.Name)
	assert.Equal(t, "host1", resp.Msg.Gdc.Hostname)
	assert.Equal(t, "192.168.1.1", resp.Msg.Gdc.Ip)
	assert.Equal(t, int32(6005), resp.Msg.Gdc.Port)
	assert.Equal(t, "/tmp/gdc1", resp.Msg.Gdc.Datapath)
	assert.Equal(t, true, resp.Msg.Gdc.Enabled)
	assert.Equal(t, true, resp.Msg.Gdc.WriteOutput)
	assert.Equal(t, false, resp.Msg.Gdc.Decode)
	assert.Equal(t, int32(50051), resp.Msg.Gdc.GrpcPort)
	assert.Equal(t, int32(12112), resp.Msg.Gdc.PrometheusPort)
	mockQuerier.AssertExpectations(t)
}

func TestGetGDC_NotFound(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("GetGDC", mock.Anything, int32(999)).
		Return(database.Gdc{}, sql.ErrNoRows)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetGDCRequest{Id: 999})

	resp, err := server.GetGDC(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	mockQuerier.AssertExpectations(t)
}

func TestUpdateGDC_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("UpdateGDC", mock.Anything, mock.MatchedBy(func(arg database.UpdateGDCParams) bool {
		return arg.ID == 1 &&
			arg.Name.String == "updated_gdc" &&
			arg.Hostname.String == "updatedhost" &&
			arg.Ip.String == "127.0.0.2" &&
			arg.Port.Int32 == 6006 &&
			arg.Datapath.String == "/tmp/updated" &&
			arg.Enabled.Bool == false &&
			arg.Writeoutput.Bool == false &&
			arg.Decode.Bool == true &&
			arg.GrpcPort.Int32 == 50053 &&
			arg.PrometheusPort.Int32 == 12114
	})).Return(nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.UpdateGDCRequest{
		Id:             1,
		Name:           "updated_gdc",
		Hostname:       "updatedhost",
		Ip:             "127.0.0.2",
		Port:           6006,
		Datapath:       "/tmp/updated",
		Enabled:        false,
		WriteOutput:    false,
		Decode:         true,
		GrpcPort:       50053,
		PrometheusPort: 12114,
	})

	resp, err := server.UpdateGDC(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "GDC updated successfully", resp.Msg.Message)
	mockQuerier.AssertExpectations(t)
}

func TestUpdateGDC_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("UpdateGDC", mock.Anything, mock.Anything).
		Return(fmt.Errorf("database error"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.UpdateGDCRequest{
		Id:             1,
		Name:           "updated_gdc",
		Hostname:       "updatedhost",
		Ip:             "127.0.0.2",
		Port:           6006,
		Datapath:       "/tmp/updated",
		Enabled:        false,
		WriteOutput:    false,
		Decode:         true,
		GrpcPort:       50053,
		PrometheusPort: 12114,
	})

	resp, err := server.UpdateGDC(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestDeleteGDC_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("DeleteGDC", mock.Anything, int32(1)).Return(nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.DeleteGDCRequest{Id: 1})

	resp, err := server.DeleteGDC(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "GDC deleted successfully", resp.Msg.Message)
	mockQuerier.AssertExpectations(t)
}

func TestDeleteGDC_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("DeleteGDC", mock.Anything, int32(1)).
		Return(fmt.Errorf("database error"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.DeleteGDCRequest{Id: 1})

	resp, err := server.DeleteGDC(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}
