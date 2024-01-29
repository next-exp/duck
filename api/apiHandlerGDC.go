package main

import (
	"context"
	"database/sql"
	"fmt"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
	"github.com/jmbenlloch/next_duck/pkg/database"
)

// ===== GDC Management =====

func (s *DuckAPIServer) CreateGDC(ctx context.Context, req *connect.Request[pb.CreateGDCRequest]) (*connect.Response[pb.CreateGDCResponse], error) {
	gdcCreate := database.CreateGDCParams{
		Name:           sql.NullString{String: req.Msg.Name, Valid: req.Msg.Name != ""},
		Hostname:       sql.NullString{String: req.Msg.Hostname, Valid: req.Msg.Hostname != ""},
		Ip:             sql.NullString{String: req.Msg.Ip, Valid: req.Msg.Ip != ""},
		Port:           sql.NullInt32{Int32: req.Msg.Port, Valid: true},
		Datapath:       sql.NullString{String: req.Msg.Datapath, Valid: req.Msg.Datapath != ""},
		Enabled:        sql.NullBool{Bool: req.Msg.Enabled, Valid: true},
		Writeoutput:    sql.NullBool{Bool: req.Msg.WriteOutput, Valid: true},
		Decode:         sql.NullBool{Bool: req.Msg.Decode, Valid: true},
		GrpcPort:       sql.NullInt32{Int32: req.Msg.GrpcPort, Valid: true},
		PrometheusPort: sql.NullInt32{Int32: req.Msg.PrometheusPort, Valid: true},
	}

	_, err := s.queries.CreateGDC(ctx, gdcCreate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.CreateGDCResponse{
		Success: true,
		Message: "GDC created successfully",
	}), nil
}

func (s *DuckAPIServer) GetGDCs(ctx context.Context, req *connect.Request[pb.GetGDCsRequest]) (*connect.Response[pb.GetGDCsResponse], error) {
	gdcsList, err := s.queries.ListGDCs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	var gdcs []*pb.GDC
	for _, gdc := range gdcsList {
		gdcs = append(gdcs, &pb.GDC{
			Id:             gdc.ID,
			Name:           gdc.Name.String,
			Hostname:       gdc.Hostname.String,
			Ip:             gdc.Ip.String,
			Port:           gdc.Port.Int32,
			Datapath:       gdc.Datapath.String,
			Enabled:        gdc.Enabled.Bool,
			WriteOutput:    gdc.Writeoutput.Bool,
			Decode:         gdc.Decode.Bool,
			GrpcPort:       gdc.GrpcPort.Int32,
			PrometheusPort: gdc.PrometheusPort.Int32,
		})
	}

	return connect.NewResponse(&pb.GetGDCsResponse{
		Gdcs: gdcs,
	}), nil
}

func (s *DuckAPIServer) GetGDC(ctx context.Context, req *connect.Request[pb.GetGDCRequest]) (*connect.Response[pb.GetGDCResponse], error) {
	gdc, err := s.queries.GetGDC(ctx, req.Msg.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("GDC not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.GetGDCResponse{
		Gdc: &pb.GDC{
			Id:             gdc.ID,
			Name:           gdc.Name.String,
			Hostname:       gdc.Hostname.String,
			Ip:             gdc.Ip.String,
			Port:           gdc.Port.Int32,
			Datapath:       gdc.Datapath.String,
			Enabled:        gdc.Enabled.Bool,
			WriteOutput:    gdc.Writeoutput.Bool,
			Decode:         gdc.Decode.Bool,
			GrpcPort:       gdc.GrpcPort.Int32,
			PrometheusPort: gdc.PrometheusPort.Int32,
		},
	}), nil
}

func (s *DuckAPIServer) UpdateGDC(ctx context.Context, req *connect.Request[pb.UpdateGDCRequest]) (*connect.Response[pb.UpdateGDCResponse], error) {
	gdcUpdated := database.UpdateGDCParams{
		ID:             req.Msg.Id,
		Name:           sql.NullString{String: req.Msg.Name, Valid: req.Msg.Name != ""},
		Hostname:       sql.NullString{String: req.Msg.Hostname, Valid: req.Msg.Hostname != ""},
		Ip:             sql.NullString{String: req.Msg.Ip, Valid: req.Msg.Ip != ""},
		Port:           sql.NullInt32{Int32: req.Msg.Port, Valid: true},
		Datapath:       sql.NullString{String: req.Msg.Datapath, Valid: req.Msg.Datapath != ""},
		Enabled:        sql.NullBool{Bool: req.Msg.Enabled, Valid: true},
		Writeoutput:    sql.NullBool{Bool: req.Msg.WriteOutput, Valid: true},
		Decode:         sql.NullBool{Bool: req.Msg.Decode, Valid: true},
		GrpcPort:       sql.NullInt32{Int32: req.Msg.GrpcPort, Valid: true},
		PrometheusPort: sql.NullInt32{Int32: req.Msg.PrometheusPort, Valid: true},
	}

	err := s.queries.UpdateGDC(ctx, gdcUpdated)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.UpdateGDCResponse{
		Success: true,
		Message: "GDC updated successfully",
	}), nil
}

func (s *DuckAPIServer) DeleteGDC(ctx context.Context, req *connect.Request[pb.DeleteGDCRequest]) (*connect.Response[pb.DeleteGDCResponse], error) {
	err := s.queries.DeleteGDC(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database error: %w", err))
	}

	return connect.NewResponse(&pb.DeleteGDCResponse{
		Success: true,
		Message: "GDC deleted successfully",
	}), nil
}
