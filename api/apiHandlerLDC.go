package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"github.com/jmbenlloch/next_duck/pkg/database"
	"golang.org/x/exp/maps"
)

// ===== LDC Management =====

func (s *DuckAPIServer) CreateLDC(ctx context.Context, req *connect.Request[pb.CreateLDCRequest]) (*connect.Response[pb.CreateLDCResponse], error) {
	ldcCreate := database.CreateLDCParams{
		Name:           sql.NullString{String: req.Msg.Name, Valid: req.Msg.Name != ""},
		Hostname:       sql.NullString{String: req.Msg.Hostname, Valid: req.Msg.Hostname != ""},
		Ip:             sql.NullString{String: req.Msg.Ip, Valid: req.Msg.Ip != ""},
		Enabled:        sql.NullBool{Bool: req.Msg.Enabled, Valid: true},
		GrpcPort:       sql.NullInt32{Int32: req.Msg.GrpcPort, Valid: true},
		PrometheusPort: sql.NullInt32{Int32: req.Msg.PrometheusPort, Valid: true},
	}

	_, err := s.queries.CreateLDC(ctx, ldcCreate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.CreateLDCResponse{
		Success: true,
		Message: "LDC created successfully",
	}), nil
}

func (s *DuckAPIServer) GetLDCs(ctx context.Context, req *connect.Request[pb.GetLDCsRequest]) (*connect.Response[pb.GetLDCsResponse], error) {
	ldcsList, err := s.queries.ListLDCs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	equipmentsList, err := s.queries.ListEquipments(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	ldcMap := make(map[int32]*pb.LDC)

	for _, ldc := range ldcsList {
		ldcMap[ldc.ID] = &pb.LDC{
			Id:             ldc.ID,
			Name:           ldc.Name.String,
			Hostname:       ldc.Hostname.String,
			Ip:             ldc.Ip.String,
			Enabled:        ldc.Enabled.Bool,
			GrpcPort:       ldc.GrpcPort.Int32,
			PrometheusPort: ldc.PrometheusPort.Int32,
			Equipments:     make([]*pb.Equipment, 0),
		}
	}

	for _, equipment := range equipmentsList {
		if ldc, exists := ldcMap[equipment.Ldcid.Int32]; exists {
			ldc.Equipments = append(ldc.Equipments, &pb.Equipment{
				Id:       equipment.ID,
				Type:     equipment.Type.Int32,
				DeviceIp: equipment.DeviceIp.String,
				HostIp:   equipment.HostIp.String,
				HostPort: equipment.HostPort.Int32,
				LdcId:    equipment.Ldcid.Int32,
				Enabled:  equipment.Enabled.Bool,
			})
		}
	}

	ldcs := maps.Values(ldcMap)
	sort.Slice(ldcs, func(i, j int) bool {
		return ldcs[i].Name < ldcs[j].Name
	})

	return connect.NewResponse(&pb.GetLDCsResponse{
		Ldcs: ldcs,
	}), nil
}

func (s *DuckAPIServer) GetLDC(ctx context.Context, req *connect.Request[pb.GetLDCRequest]) (*connect.Response[pb.GetLDCResponse], error) {
	ldc, err := s.queries.GetLDC(ctx, req.Msg.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("LDC not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.GetLDCResponse{
		Ldc: &pb.LDC{
			Id:             ldc.ID,
			Name:           ldc.Name.String,
			Hostname:       ldc.Hostname.String,
			Ip:             ldc.Ip.String,
			Enabled:        ldc.Enabled.Bool,
			GrpcPort:       ldc.GrpcPort.Int32,
			PrometheusPort: ldc.PrometheusPort.Int32,
			Equipments:     make([]*pb.Equipment, 0),
		},
	}), nil
}

func (s *DuckAPIServer) UpdateLDC(ctx context.Context, req *connect.Request[pb.UpdateLDCRequest]) (*connect.Response[pb.UpdateLDCResponse], error) {
	ldcUpdated := database.UpdateLDCParams{
		ID:             req.Msg.Id,
		Name:           sql.NullString{String: req.Msg.Name, Valid: req.Msg.Name != ""},
		Hostname:       sql.NullString{String: req.Msg.Hostname, Valid: req.Msg.Hostname != ""},
		Ip:             sql.NullString{String: req.Msg.Ip, Valid: req.Msg.Ip != ""},
		Enabled:        sql.NullBool{Bool: req.Msg.Enabled, Valid: true},
		GrpcPort:       sql.NullInt32{Int32: req.Msg.GrpcPort, Valid: true},
		PrometheusPort: sql.NullInt32{Int32: req.Msg.PrometheusPort, Valid: true},
	}

	err := s.queries.UpdateLDC(ctx, ldcUpdated)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.UpdateLDCResponse{
		Success: true,
		Message: "LDC updated successfully",
	}), nil
}

func (s *DuckAPIServer) DeleteLDC(ctx context.Context, req *connect.Request[pb.DeleteLDCRequest]) (*connect.Response[pb.DeleteLDCResponse], error) {
	err := s.queries.DeleteLDC(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.DeleteLDCResponse{
		Success: true,
		Message: "LDC deleted successfully",
	}), nil
}
