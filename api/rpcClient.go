package main

import (
	"context"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

// RPCClient interface for all RPC operations to LDCs/GDCs
// This allows mocking in tests, similar to SSHClient in restart.go
type RPCClient interface {
	// Ping operations
	Ping(ctx context.Context, ip string, port int) (bool, error)

	// Server control
	StartServer(ctx context.Context, ip string, port int) error
	StopServer(ctx context.Context, ip string, port int) error
	GetState(ctx context.Context, ip string, port int) error

	// Statistics
	FetchRunStatistics(ctx context.Context, ip string, port int) (duck.RunStatistics, error)
}

// RealRPCClient implements RPCClient using actual RPC calls
type RealRPCClient struct{}

func NewRealRPCClient() RPCClient {
	return &RealRPCClient{}
}

func (r *RealRPCClient) Ping(ctx context.Context, ip string, port int) (bool, error) {
	return pingDevicesRPC(ip, port)
}

func (r *RealRPCClient) StartServer(ctx context.Context, ip string, port int) error {
	return startServerRPC(ip, port)
}

func (r *RealRPCClient) StopServer(ctx context.Context, ip string, port int) error {
	return stopServerRPC(ip, port)
}

func (r *RealRPCClient) GetState(ctx context.Context, ip string, port int) error {
	return getStateRPC(ip, port)
}

func (r *RealRPCClient) FetchRunStatistics(ctx context.Context, ip string, port int) (duck.RunStatistics, error) {
	return fetchRunStatisticsRPC(ip, port)
}

