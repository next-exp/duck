package main

import (
	"context"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
)

// ===== Run Control Operations =====

func (s *DuckAPIServer) StartRun(ctx context.Context, req *connect.Request[pb.StartRunRequest]) (*connect.Response[pb.StartRunResponse], error) {
	startProcesses(s)
	return connect.NewResponse(&pb.StartRunResponse{
		Success: true,
		Message: "starting processes",
	}), nil
}

func (s *DuckAPIServer) StopRun(ctx context.Context, req *connect.Request[pb.StopRunRequest]) (*connect.Response[pb.StopRunResponse], error) {
	stopProcesses(s)
	return connect.NewResponse(&pb.StopRunResponse{
		Success: true,
		Message: "stopping processes",
	}), nil
}

func (s *DuckAPIServer) ForceStopRun(ctx context.Context, req *connect.Request[pb.ForceStopRunRequest]) (*connect.Response[pb.ForceStopRunResponse], error) {
	forceStopProcesses(s)
	return connect.NewResponse(&pb.ForceStopRunResponse{
		Success: true,
		Message: "force stopping processes",
	}), nil
}

func (s *DuckAPIServer) GetProcessStates(ctx context.Context, req *connect.Request[pb.GetProcessStatesRequest]) (*connect.Response[pb.GetProcessStatesResponse], error) {
	getProcessStates(s)
	return connect.NewResponse(&pb.GetProcessStatesResponse{
		Success: true,
		Message: "get processes states",
	}), nil
}

func (s *DuckAPIServer) RestartServices(ctx context.Context, req *connect.Request[pb.RestartServicesRequest]) (*connect.Response[pb.RestartServicesResponse], error) {
	restartServices(s.configFilename)
	return connect.NewResponse(&pb.RestartServicesResponse{
		Success: true,
		Message: "restarting services",
	}), nil
}
