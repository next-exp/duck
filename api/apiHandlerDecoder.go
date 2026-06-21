package main

import (
	"context"
	"database/sql"
	"fmt"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"github.com/jmbenlloch/next_duck/pkg/database"
)

// ===== Decoder Configuration =====

func (s *DuckAPIServer) GetDecoderConfiguration(ctx context.Context, req *connect.Request[pb.GetDecoderConfigurationRequest]) (*connect.Response[pb.GetDecoderConfigurationResponse], error) {
	config, err := s.queries.GetDecoderParams(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("decoder configuration not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.GetDecoderConfigurationResponse{
		Configuration: &pb.DecoderConfiguration{
			ExtTrigger:       int32(config.ExtTrigger),
			TrgCode_1:        int32(config.TrgCode1),
			TrgCode_2:        int32(config.TrgCode2),
			ReadPmts:         config.ReadPmts.Bool,
			ReadSipms:        config.ReadSipms.Bool,
			ReadTrigger:      config.ReadTrigger.Bool,
			ReadFibers:       config.ReadFibers.Bool,
			SplitTrigger:     config.SplitTrigger.Bool,
			NoDb:             config.NoDb.Bool,
			Discard:          config.Discard.Bool,
			Host:             config.Host,
			User:             config.User,
			Password:         config.Passwd,
			DbName:           config.DbName,
			WriteData:        config.WriteData.Bool,
			UseBlosc:         config.UseBlosc.Bool,
			BloscAlgorithm:   config.BloscAlgorithm,
			CompressionLevel: int32(config.CompressionLevel),
			BitShuffle:       config.BitShuffle,
		},
	}), nil
}

func (s *DuckAPIServer) UpdateDecoderConfiguration(ctx context.Context, req *connect.Request[pb.UpdateDecoderConfigurationRequest]) (*connect.Response[pb.UpdateDecoderConfigurationResponse], error) {
	// First truncate the table
	err := s.queries.TruncateDecoderParams(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error truncating: %w", err))
	}

	// Then insert new configuration
	updateParams := database.UpdateDecoderParamsParams{
		ExtTrigger:       int32(req.Msg.Configuration.ExtTrigger),
		TrgCode1:         int32(req.Msg.Configuration.TrgCode_1),
		TrgCode2:         int32(req.Msg.Configuration.TrgCode_2),
		ReadPmts:         sql.NullBool{Bool: req.Msg.Configuration.ReadPmts, Valid: true},
		ReadSipms:        sql.NullBool{Bool: req.Msg.Configuration.ReadSipms, Valid: true},
		ReadTrigger:      sql.NullBool{Bool: req.Msg.Configuration.ReadTrigger, Valid: true},
		ReadFibers:       sql.NullBool{Bool: req.Msg.Configuration.ReadFibers, Valid: true},
		SplitTrigger:     sql.NullBool{Bool: req.Msg.Configuration.SplitTrigger, Valid: true},
		NoDb:             sql.NullBool{Bool: req.Msg.Configuration.NoDb, Valid: true},
		Discard:          sql.NullBool{Bool: req.Msg.Configuration.Discard, Valid: true},
		Host:             req.Msg.Configuration.Host,
		User:             req.Msg.Configuration.User,
		Passwd:           req.Msg.Configuration.Password,
		DbName:           req.Msg.Configuration.DbName,
		WriteData:        sql.NullBool{Bool: req.Msg.Configuration.WriteData, Valid: true},
		UseBlosc:         sql.NullBool{Bool: req.Msg.Configuration.UseBlosc, Valid: true},
		BloscAlgorithm:   req.Msg.Configuration.BloscAlgorithm,
		CompressionLevel: int32(req.Msg.Configuration.CompressionLevel),
		BitShuffle:       req.Msg.Configuration.BitShuffle,
	}

	err = s.queries.UpdateDecoderParams(ctx, updateParams)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error updating: %w", err))
	}

	return connect.NewResponse(&pb.UpdateDecoderConfigurationResponse{
		Success: true,
		Message: "Decoder configuration updated successfully",
	}), nil
}
