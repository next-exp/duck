package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
)

// ===== Run Information =====

func (s *DuckAPIServer) GetRunNumber(ctx context.Context, req *connect.Request[pb.GetRunNumberRequest]) (*connect.Response[pb.GetRunNumberResponse], error) {
	runNumber, err := s.queries.GetLatestRun(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.GetRunNumberResponse{
		RunNumber: runNumber,
	}), nil
}

func (s *DuckAPIServer) CheckDisabled(ctx context.Context, req *connect.Request[pb.CheckDisabledRequest]) (*connect.Response[pb.CheckDisabledResponse], error) {
	warnings := &pb.DisabledWarning{
		Gdcs:       []string{},
		Ldcs:       []string{},
		Equipments: []string{},
		Writing:    []string{},
	}

	// Check disabled GDCs
	disabledGDCs, err := s.queries.GetDisabledGDCs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}
	for _, gdc := range disabledGDCs {
		warnings.Gdcs = append(warnings.Gdcs, gdc.Name.String)
	}

	// Check disabled LDCs
	disabledLDCs, err := s.queries.GetDisabledLDCs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}
	for _, ldc := range disabledLDCs {
		warnings.Ldcs = append(warnings.Ldcs, ldc.Name.String)
	}

	// Check disabled equipments
	disabledEquipments, err := s.queries.GetDisabledEquipments(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}
	for _, equipment := range disabledEquipments {
		warnings.Equipments = append(warnings.Equipments, fmt.Sprintf("Equipment %d", equipment.ID))
	}

	// Check GDCs with write_output disabled
	gdcsWithoutWrite, err := s.queries.GetGDCsWithoutWrite(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	decoderParams, err := s.queries.GetDecoderParams(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}
	decoderWriting := err == nil && decoderParams.WriteData.Bool

	for _, gdc := range gdcsWithoutWrite {
		// Skip if the decoder is writing HDF5 files for this GDC
		if gdc.Decode.Bool && decoderWriting {
			continue
		}
		warnings.Writing = append(warnings.Writing, gdc.Name.String)
	}

	return connect.NewResponse(&pb.CheckDisabledResponse{
		Warnings: warnings,
	}), nil
}
