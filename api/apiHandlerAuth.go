package main

import (
	"context"

	"connectrpc.com/connect"
	pb "github.com/jmbenlloch/next_duck/rpc/api"
)

// ===== Authentication =====

func (s *DuckAPIServer) GetToken(ctx context.Context, req *connect.Request[pb.GetTokenRequest]) (*connect.Response[pb.GetTokenResponse], error) {
	return connect.NewResponse(&pb.GetTokenResponse{
		Token: s.centrifugalToken,
	}), nil
}
