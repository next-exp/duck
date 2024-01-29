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

func TestGetDecoderConfiguration_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	expected := database.Decoderparam{
		ExtTrigger:       1,
		TrgCode1:         100,
		TrgCode2:         200,
		ReadPmts:         sql.NullBool{Bool: true, Valid: true},
		ReadSipms:        sql.NullBool{Bool: true, Valid: true},
		ReadTrigger:      sql.NullBool{Bool: true, Valid: true},
		SplitTrigger:     sql.NullBool{Bool: false, Valid: true},
		NoDb:             sql.NullBool{Bool: false, Valid: true},
		Discard:          sql.NullBool{Bool: false, Valid: true},
		Host:             "localhost",
		User:             "testuser",
		Passwd:           "testpass",
		DbName:           "testdb",
		WriteData:        sql.NullBool{Bool: true, Valid: true},
		UseBlosc:         sql.NullBool{Bool: true, Valid: true},
		BloscAlgorithm:   "lz4",
		CompressionLevel: 5,
		BitShuffle:       "yes",
	}

	mockQuerier.On("GetDecoderParams", mock.Anything).Return(expected, nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetDecoderConfigurationRequest{})

	resp, err := server.GetDecoderConfiguration(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int32(expected.ExtTrigger), resp.Msg.Configuration.ExtTrigger)
	assert.Equal(t, int32(expected.TrgCode1), resp.Msg.Configuration.TrgCode_1)
	assert.Equal(t, int32(expected.TrgCode2), resp.Msg.Configuration.TrgCode_2)
	assert.Equal(t, expected.ReadPmts.Bool, resp.Msg.Configuration.ReadPmts)
	assert.Equal(t, expected.ReadSipms.Bool, resp.Msg.Configuration.ReadSipms)
	assert.Equal(t, expected.ReadTrigger.Bool, resp.Msg.Configuration.ReadTrigger)
	assert.Equal(t, expected.SplitTrigger.Bool, resp.Msg.Configuration.SplitTrigger)
	assert.Equal(t, expected.NoDb.Bool, resp.Msg.Configuration.NoDb)
	assert.Equal(t, expected.Discard.Bool, resp.Msg.Configuration.Discard)
	assert.Equal(t, expected.Host, resp.Msg.Configuration.Host)
	assert.Equal(t, expected.User, resp.Msg.Configuration.User)
	assert.Equal(t, expected.Passwd, resp.Msg.Configuration.Password)
	assert.Equal(t, expected.DbName, resp.Msg.Configuration.DbName)
	assert.Equal(t, expected.WriteData.Bool, resp.Msg.Configuration.WriteData)
	assert.Equal(t, expected.UseBlosc.Bool, resp.Msg.Configuration.UseBlosc)
	assert.Equal(t, expected.BloscAlgorithm, resp.Msg.Configuration.BloscAlgorithm)
	assert.Equal(t, int32(expected.CompressionLevel), resp.Msg.Configuration.CompressionLevel)
	assert.Equal(t, expected.BitShuffle, resp.Msg.Configuration.BitShuffle)
	mockQuerier.AssertExpectations(t)
}

func TestGetDecoderConfiguration_NotFound(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("GetDecoderParams", mock.Anything).Return(database.Decoderparam{}, sql.ErrNoRows)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetDecoderConfigurationRequest{})

	resp, err := server.GetDecoderConfiguration(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not found")
	mockQuerier.AssertExpectations(t)
}

func TestGetDecoderConfiguration_DatabaseError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("GetDecoderParams", mock.Anything).
		Return(database.Decoderparam{}, fmt.Errorf("database connection failed"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.GetDecoderConfigurationRequest{})

	resp, err := server.GetDecoderConfiguration(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error")
	mockQuerier.AssertExpectations(t)
}

func TestUpdateDecoderConfiguration_Success(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	testConfig := &pb.DecoderConfiguration{
		ExtTrigger: 2, TrgCode_1: 150, TrgCode_2: 250,
		ReadPmts: false, ReadSipms: true, ReadTrigger: false, SplitTrigger: true,
		NoDb: true, Discard: true, Host: "testhost", User: "admin",
		Password: "adminpass", DbName: "proddb", WriteData: false, UseBlosc: false,
		BloscAlgorithm: "zstd", CompressionLevel: 9, BitShuffle: "no",
	}

	mockQuerier.On("TruncateDecoderParams", mock.Anything).Return(nil)
	mockQuerier.On("UpdateDecoderParams", mock.Anything, mock.MatchedBy(func(arg database.UpdateDecoderParamsParams) bool {
		return arg.ExtTrigger == 2 &&
			arg.TrgCode1 == 150 &&
			arg.TrgCode2 == 250 &&
			arg.ReadPmts.Bool == false &&
			arg.ReadSipms.Bool == true &&
			arg.ReadTrigger.Bool == false &&
			arg.SplitTrigger.Bool == true &&
			arg.NoDb.Bool == true &&
			arg.Discard.Bool == true &&
			arg.Host == "testhost" &&
			arg.User == "admin" &&
			arg.Passwd == "adminpass" &&
			arg.DbName == "proddb" &&
			arg.WriteData.Bool == false &&
			arg.UseBlosc.Bool == false &&
			arg.BloscAlgorithm == "zstd" &&
			arg.CompressionLevel == 9 &&
			arg.BitShuffle == "no"
	})).Return(nil)

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.UpdateDecoderConfigurationRequest{
		Configuration: testConfig,
	})

	resp, err := server.UpdateDecoderConfiguration(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Msg.Success)
	assert.Equal(t, "Decoder configuration updated successfully", resp.Msg.Message)
	mockQuerier.AssertExpectations(t)
}

func TestUpdateDecoderConfiguration_TruncateError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("TruncateDecoderParams", mock.Anything).
		Return(fmt.Errorf("truncate failed"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.UpdateDecoderConfigurationRequest{
		Configuration: &pb.DecoderConfiguration{
			ExtTrigger: 1, TrgCode_1: 100, TrgCode_2: 200,
			ReadPmts: true, ReadSipms: true, ReadTrigger: true,
			SplitTrigger: false, NoDb: false, Discard: false,
			Host: "localhost", User: "testuser", Password: "testpass",
			DbName: "testdb", WriteData: true, UseBlosc: true,
			BloscAlgorithm: "lz4", CompressionLevel: 5, BitShuffle: "yes",
		},
	})

	resp, err := server.UpdateDecoderConfiguration(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error truncating")
	mockQuerier.AssertExpectations(t)
}

func TestUpdateDecoderConfiguration_UpdateError(t *testing.T) {
	mockQuerier := mocks.NewQuerier(t)

	mockQuerier.On("TruncateDecoderParams", mock.Anything).Return(nil)
	mockQuerier.On("UpdateDecoderParams", mock.Anything, mock.Anything).
		Return(fmt.Errorf("update failed"))

	server := NewDuckAPIServerWithQuerier(mockQuerier, "", "", false)
	req := connect.NewRequest(&pb.UpdateDecoderConfigurationRequest{
		Configuration: &pb.DecoderConfiguration{
			ExtTrigger: 1, TrgCode_1: 100, TrgCode_2: 200,
			ReadPmts: true, ReadSipms: true, ReadTrigger: true,
			SplitTrigger: false, NoDb: false, Discard: false,
			Host: "localhost", User: "testuser", Password: "testpass",
			DbName: "testdb", WriteData: true, UseBlosc: true,
			BloscAlgorithm: "lz4", CompressionLevel: 5, BitShuffle: "yes",
		},
	})

	resp, err := server.UpdateDecoderConfiguration(context.Background(), req)

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "database error updating")
	mockQuerier.AssertExpectations(t)
}
