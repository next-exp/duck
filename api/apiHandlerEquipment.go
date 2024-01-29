package main

import (
	"context"
	"database/sql"
	"fmt"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"github.com/jmbenlloch/next_duck/pkg/database"
)

// ===== Equipment Management =====

func (s *DuckAPIServer) CreateEquipment(ctx context.Context, req *connect.Request[pb.CreateEquipmentRequest]) (*connect.Response[pb.CreateEquipmentResponse], error) {
	equipmentCreate := database.CreateEquipmentParams{
		Type:     sql.NullInt32{Int32: req.Msg.Type, Valid: true},
		DeviceIp: sql.NullString{String: req.Msg.DeviceIp, Valid: req.Msg.DeviceIp != ""},
		HostIp:   sql.NullString{String: req.Msg.HostIp, Valid: req.Msg.HostIp != ""},
		HostPort: sql.NullInt32{Int32: req.Msg.HostPort, Valid: true},
		Ldcid:    sql.NullInt32{Int32: req.Msg.LdcId, Valid: true},
		Enabled:  sql.NullBool{Bool: req.Msg.Enabled, Valid: true},
	}

	_, err := s.queries.CreateEquipment(ctx, equipmentCreate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.CreateEquipmentResponse{
		Success: true,
		Message: "Equipment created successfully",
	}), nil
}

func (s *DuckAPIServer) GetEquipments(ctx context.Context, req *connect.Request[pb.GetEquipmentsRequest]) (*connect.Response[pb.GetEquipmentsResponse], error) {
	equipmentsList, err := s.queries.ListEquipments(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	var equipments []*pb.Equipment
	for _, equipment := range equipmentsList {
		equipments = append(equipments, &pb.Equipment{
			Id:       equipment.ID,
			Type:     equipment.Type.Int32,
			DeviceIp: equipment.DeviceIp.String,
			HostIp:   equipment.HostIp.String,
			HostPort: equipment.HostPort.Int32,
			LdcId:    equipment.Ldcid.Int32,
			Enabled:  equipment.Enabled.Bool,
		})
	}

	return connect.NewResponse(&pb.GetEquipmentsResponse{
		Equipments: equipments,
	}), nil
}

func (s *DuckAPIServer) GetEquipment(ctx context.Context, req *connect.Request[pb.GetEquipmentRequest]) (*connect.Response[pb.GetEquipmentResponse], error) {
	equipment, err := s.queries.GetEquipment(ctx, req.Msg.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("Equipment not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.GetEquipmentResponse{
		Equipment: &pb.Equipment{
			Id:       equipment.ID,
			Type:     equipment.Type.Int32,
			DeviceIp: equipment.DeviceIp.String,
			HostIp:   equipment.HostIp.String,
			HostPort: equipment.HostPort.Int32,
			LdcId:    equipment.Ldcid.Int32,
			Enabled:  equipment.Enabled.Bool,
		},
	}), nil
}

func (s *DuckAPIServer) UpdateEquipment(ctx context.Context, req *connect.Request[pb.UpdateEquipmentRequest]) (*connect.Response[pb.UpdateEquipmentResponse], error) {
	equipmentUpdated := database.UpdateEquipmentParams{
		ID:       req.Msg.Id,
		Type:     sql.NullInt32{Int32: req.Msg.Type, Valid: true},
		DeviceIp: sql.NullString{String: req.Msg.DeviceIp, Valid: req.Msg.DeviceIp != ""},
		HostIp:   sql.NullString{String: req.Msg.HostIp, Valid: req.Msg.HostIp != ""},
		HostPort: sql.NullInt32{Int32: req.Msg.HostPort, Valid: true},
		Ldcid:    sql.NullInt32{Int32: req.Msg.LdcId, Valid: true},
		Enabled:  sql.NullBool{Bool: req.Msg.Enabled, Valid: true},
	}

	err := s.queries.UpdateEquipment(ctx, equipmentUpdated)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.UpdateEquipmentResponse{
		Success: true,
		Message: "Equipment updated successfully",
	}), nil
}

func (s *DuckAPIServer) DeleteEquipment(ctx context.Context, req *connect.Request[pb.DeleteEquipmentRequest]) (*connect.Response[pb.DeleteEquipmentResponse], error) {
	err := s.queries.DeleteEquipment(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.DeleteEquipmentResponse{
		Success: true,
		Message: "Equipment deleted successfully",
	}), nil
}
