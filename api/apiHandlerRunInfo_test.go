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

func TestCheckDisabled_NothingDisabled(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("GetDisabledGDCs", mock.Anything).Return([]database.Gdc{}, nil)
	mockQuerier.On("GetDisabledLDCs", mock.Anything).Return([]database.Ldc{}, nil)
	mockQuerier.On("GetDisabledEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetGDCsWithoutWrite", mock.Anything).Return([]database.Gdc{}, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CheckDisabledRequest{})

	resp, err := server.CheckDisabled(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Msg.Warnings.Gdcs)
	assert.Empty(t, resp.Msg.Warnings.Ldcs)
	assert.Empty(t, resp.Msg.Warnings.Equipments)
	assert.Empty(t, resp.Msg.Warnings.Writing)
	mockQuerier.AssertExpectations(t)
}

func TestCheckDisabled_DisabledGDCs(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	disabledGDCs := []database.Gdc{
		{
			ID:             1,
			Name:           sql.NullString{String: "gdc1", Valid: true},
			Hostname:       sql.NullString{String: "host1", Valid: true},
			Ip:             sql.NullString{String: "192.168.1.1", Valid: true},
			Port:           sql.NullInt32{Int32: 8080, Valid: true},
			GrpcPort:       sql.NullInt32{Int32: 50051, Valid: true},
			PrometheusPort: sql.NullInt32{Int32: 12112, Valid: true},
			Datapath:       sql.NullString{String: "/data", Valid: true},
			Enabled:        sql.NullBool{Bool: false, Valid: true},
			Writeoutput:    sql.NullBool{Bool: true, Valid: true},
			Decode:         sql.NullBool{Bool: true, Valid: true},
		},
		{
			ID:             2,
			Name:           sql.NullString{String: "gdc2", Valid: true},
			Hostname:       sql.NullString{String: "host2", Valid: true},
			Ip:             sql.NullString{String: "192.168.1.2", Valid: true},
			Port:           sql.NullInt32{Int32: 8080, Valid: true},
			GrpcPort:       sql.NullInt32{Int32: 50052, Valid: true},
			PrometheusPort: sql.NullInt32{Int32: 12113, Valid: true},
			Datapath:       sql.NullString{String: "/data", Valid: true},
			Enabled:        sql.NullBool{Bool: false, Valid: true},
			Writeoutput:    sql.NullBool{Bool: true, Valid: true},
			Decode:         sql.NullBool{Bool: true, Valid: true},
		},
	}

	mockQuerier.On("GetDisabledGDCs", mock.Anything).Return(disabledGDCs, nil)
	mockQuerier.On("GetDisabledLDCs", mock.Anything).Return([]database.Ldc{}, nil)
	mockQuerier.On("GetDisabledEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetGDCsWithoutWrite", mock.Anything).Return([]database.Gdc{}, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CheckDisabledRequest{})

	resp, err := server.CheckDisabled(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Msg.Warnings.Gdcs, 2)
	assert.Contains(t, resp.Msg.Warnings.Gdcs, "gdc1")
	assert.Contains(t, resp.Msg.Warnings.Gdcs, "gdc2")
	assert.Empty(t, resp.Msg.Warnings.Ldcs)
	assert.Empty(t, resp.Msg.Warnings.Equipments)
	assert.Empty(t, resp.Msg.Warnings.Writing)
	mockQuerier.AssertExpectations(t)
}

func TestCheckDisabled_DisabledLDCs(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	disabledLDCs := []database.Ldc{
		{
			ID:             1,
			Name:           sql.NullString{String: "ldc1", Valid: true},
			Hostname:       sql.NullString{String: "host1", Valid: true},
			Ip:             sql.NullString{String: "192.168.1.1", Valid: true},
			GrpcPort:       sql.NullInt32{Int32: 50052, Valid: true},
			PrometheusPort: sql.NullInt32{Int32: 12113, Valid: true},
			Enabled:        sql.NullBool{Bool: false, Valid: true},
		},
	}

	mockQuerier.On("GetDisabledGDCs", mock.Anything).Return([]database.Gdc{}, nil)
	mockQuerier.On("GetDisabledLDCs", mock.Anything).Return(disabledLDCs, nil)
	mockQuerier.On("GetDisabledEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetGDCsWithoutWrite", mock.Anything).Return([]database.Gdc{}, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CheckDisabledRequest{})

	resp, err := server.CheckDisabled(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Msg.Warnings.Gdcs)
	assert.Len(t, resp.Msg.Warnings.Ldcs, 1)
	assert.Contains(t, resp.Msg.Warnings.Ldcs, "ldc1")
	assert.Empty(t, resp.Msg.Warnings.Equipments)
	assert.Empty(t, resp.Msg.Warnings.Writing)
	mockQuerier.AssertExpectations(t)
}

func TestCheckDisabled_DisabledEquipments(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	disabledEquipments := []database.Equipment{
		{
			ID:       1,
			Type:     sql.NullInt32{Int32: 22, Valid: true},
			DeviceIp: sql.NullString{String: "192.168.1.1", Valid: true},
			HostIp:   sql.NullString{String: "192.168.1.2", Valid: true},
			HostPort: sql.NullInt32{Int32: 6006, Valid: true},
			Enabled:  sql.NullBool{Bool: false, Valid: true},
			Ldcid:    sql.NullInt32{Int32: 1, Valid: true},
		},
	}

	mockQuerier.On("GetDisabledGDCs", mock.Anything).Return([]database.Gdc{}, nil)
	mockQuerier.On("GetDisabledLDCs", mock.Anything).Return([]database.Ldc{}, nil)
	mockQuerier.On("GetDisabledEquipments", mock.Anything).Return(disabledEquipments, nil)
	mockQuerier.On("GetGDCsWithoutWrite", mock.Anything).Return([]database.Gdc{}, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CheckDisabledRequest{})

	resp, err := server.CheckDisabled(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Msg.Warnings.Gdcs)
	assert.Empty(t, resp.Msg.Warnings.Ldcs)
	assert.Len(t, resp.Msg.Warnings.Equipments, 1)
	assert.Contains(t, resp.Msg.Warnings.Equipments, "Equipment 1")
	assert.Empty(t, resp.Msg.Warnings.Writing)
	mockQuerier.AssertExpectations(t)
}

func TestCheckDisabled_DisabledWriting(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	gdcsWithoutWrite := []database.Gdc{
		{
			ID:             1,
			Name:           sql.NullString{String: "gdc1", Valid: true},
			Hostname:       sql.NullString{String: "host1", Valid: true},
			Ip:             sql.NullString{String: "192.168.1.1", Valid: true},
			Port:           sql.NullInt32{Int32: 8080, Valid: true},
			GrpcPort:       sql.NullInt32{Int32: 50051, Valid: true},
			PrometheusPort: sql.NullInt32{Int32: 12112, Valid: true},
			Datapath:       sql.NullString{String: "/data", Valid: true},
			Enabled:        sql.NullBool{Bool: true, Valid: true},
			Writeoutput:    sql.NullBool{Bool: false, Valid: true},
			Decode:         sql.NullBool{Bool: true, Valid: true},
		},
	}

	mockQuerier.On("GetDisabledGDCs", mock.Anything).Return([]database.Gdc{}, nil)
	mockQuerier.On("GetDisabledLDCs", mock.Anything).Return([]database.Ldc{}, nil)
	mockQuerier.On("GetDisabledEquipments", mock.Anything).Return([]database.Equipment{}, nil)
	mockQuerier.On("GetGDCsWithoutWrite", mock.Anything).Return(gdcsWithoutWrite, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CheckDisabledRequest{})

	resp, err := server.CheckDisabled(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Msg.Warnings.Gdcs)
	assert.Empty(t, resp.Msg.Warnings.Ldcs)
	assert.Empty(t, resp.Msg.Warnings.Equipments)
	assert.Len(t, resp.Msg.Warnings.Writing, 1)
	assert.Contains(t, resp.Msg.Warnings.Writing, "gdc1")
	mockQuerier.AssertExpectations(t)
}

func TestCheckDisabled_GDCQueryError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("GetDisabledGDCs", mock.Anything).
		Return([]database.Gdc{}, fmt.Errorf("database error"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.CheckDisabledRequest{})

	resp, err := server.CheckDisabled(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}
