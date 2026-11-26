package main

import (
	"context"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
)

// ===== Run Control Operations =====

func (s *DuckAPIServer) StartRun(ctx context.Context, req *connect.Request[pb.StartRunRequest]) (*connect.Response[pb.StartRunResponse], error) {
	ok, conflictMsg := s.runTransition.tryBeginStart()
	if !ok {
		return connect.NewResponse(&pb.StartRunResponse{
			Success: false,
			Message: conflictMsg,
		}), nil
	}
	hub := cloneSentryHub(ctx)
	go func() {
		defer recoverToErrorWithHub(&s.runTransition, hub, "start")
		startProcesses(s)
		s.runTransition.setDone()
	}()
	return connect.NewResponse(&pb.StartRunResponse{
		Success: true,
		Message: "starting processes",
	}), nil
}

func (s *DuckAPIServer) StopRun(ctx context.Context, req *connect.Request[pb.StopRunRequest]) (*connect.Response[pb.StopRunResponse], error) {
	ok, conflictMsg := s.runTransition.tryBeginStop()
	if !ok {
		return connect.NewResponse(&pb.StopRunResponse{
			Success: false,
			Message: conflictMsg,
		}), nil
	}
	hub := cloneSentryHub(ctx)
	go func() {
		defer recoverToErrorWithHub(&s.runTransition, hub, "stop")
		stopProcesses(s)
		s.runTransition.setDone()
	}()
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
	return s.beginRestart(ctx)
}

func (s *DuckAPIServer) GetRunTransitionStatus(ctx context.Context, req *connect.Request[pb.GetRunTransitionStatusRequest]) (*connect.Response[pb.GetRunTransitionStatusResponse], error) {
	rt := &s.runTransition
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return connect.NewResponse(&pb.GetRunTransitionStatusResponse{State: stateString(rt.state), Message: rt.errMsg, RestartOperationId: s.restart.id, RestartTargets: s.restart.snapshot(), RestartError: s.restart.failure}), nil
}
