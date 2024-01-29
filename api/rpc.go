package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	duck "github.com/jmbenlloch/next_duck/pkg"
	pb "github.com/jmbenlloch/next_duck/rpc/control"
	pbconnect "github.com/jmbenlloch/next_duck/rpc/control/controlconnect"
)

func startServerRPC(ip string, port int) error {
	addr := fmt.Sprintf("%s:%d", ip, port)
	client := pbconnect.NewRunControlClient(
		http.DefaultClient,
		"http://"+addr,
	)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	message := fmt.Sprintf("Sending start to %s", addr)
	logger.Slog.Debug(message)

	r, err := client.StartRun(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		message := fmt.Errorf("could not start server via ConnectRPC: %w", err)
		return message
	}
	message = fmt.Sprintf("message from %s: %s", addr, r.Msg.GetMessage())
	logger.Slog.Debug(message)
	return err
}

func stopServerRPC(ip string, port int) error {
	addr := fmt.Sprintf("%s:%d", ip, port)
	client := pbconnect.NewRunControlClient(
		http.DefaultClient,
		"http://"+addr,
	)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	message := fmt.Sprintf("Sending stop to %s", addr)
	logger.Slog.Debug(message)

	r, err := client.StopRun(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		message := fmt.Errorf("could not stop server via ConnectRPC: %w", err)
		return message
	}
	message = fmt.Sprintf("message from %s: %s", addr, r.Msg.GetMessage())
	logger.Slog.Debug(message)
	return err
}

func pingDevicesRPC(ip string, port int) (bool, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	client := pbconnect.NewRunControlClient(
		http.DefaultClient,
		"http://"+addr,
	)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r, err := client.PingDevices(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		message := fmt.Errorf("error calling ping via ConnectRPC: %w", err)
		return false, message
	}
	return r.Msg.Success, err
}

func fetchRunStatisticsRPC(ip string, port int) (duck.RunStatistics, error) {
	addr := fmt.Sprintf("%s:%d", ip, port)
	client := pbconnect.NewRunControlClient(
		http.DefaultClient,
		"http://"+addr,
	)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r, err := client.GetRunStatistics(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		message := fmt.Errorf("error getting run statistics via ConnectRPC: %w", err)
		return duck.RunStatistics{}, message
	}
	result := duck.RunStatistics{
		Events: r.Msg.GetEvents(),
		Bytes:  r.Msg.GetBytes(),
	}
	return result, err
}

func getStateRPC(ip string, port int) error {
	addr := fmt.Sprintf("%s:%d", ip, port)
	client := pbconnect.NewRunControlClient(
		http.DefaultClient,
		"http://"+addr,
	)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	message := fmt.Sprintf("Getting state from to %s", addr)
	logger.Slog.Debug(message)

	r, err := client.GetState(ctx, connect.NewRequest(&pb.DuckRequest{}))
	if err != nil {
		message := fmt.Errorf("could not get server state via ConnectRPC: %w", err)
		return message
	}
	message = fmt.Sprintf("message from %s: %s", addr, r.Msg.GetMessage())
	logger.Slog.Debug(message)
	return err
}
