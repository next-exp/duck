package main

import (
	"context"
	"fmt"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/jmbenlloch/next_duck/pkg/database"
)

func pingDevices(ldc duck.LDCConfiguration, grpcPort int, errorChan chan error, rpcClient RPCClient) {
	if ldc.Enabled {
		message := fmt.Sprintf("pinging LDC %s", ldc.Name)
		logger.Slog.Debug(message)

		ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
		defer cancel()

		success, err := rpcClient.Ping(ctx, ldc.IP, grpcPort)
		if err != nil {
			message := fmt.Errorf("error executing ping on %s: %w", ldc.Name, err)
			errorChan <- message
			return
		}
		if !success {
			errorChan <- fmt.Errorf("ping unsuccessful from LDC %s", ldc.Name)
			return
		}
		errorChan <- nil
	} else {
		// Disabled device - send nil to indicate no error
		errorChan <- nil
	}
}

func sendPingToAllLDCs(ldcs []duck.LDCConfiguration, rpcClient RPCClient) bool {
	// returns true if success
	nLDCs := len(ldcs)
	errorChan := make(chan error, nLDCs)
	for i := 0; i < nLDCs; i++ {
		go pingDevices(ldcs[i], ldcs[i].GRPCPort, errorChan, rpcClient)
	}
	success := true
	for i := 0; i < nLDCs; i++ {
		err := <-errorChan
		if err != nil {
			logger.Slog.Error(err.Error())
			success = false
		}
	}
	return success
}

func startServer(ip string, name string, grpcPort int, errorChan chan error, rpcClient RPCClient) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	err := rpcClient.StartServer(ctx, ip, grpcPort)
	if err != nil {
		message := fmt.Errorf("could not start server %s: %w", name, err)
		errorChan <- message
		return
	}
	errorChan <- nil
}

func stopServer(ip string, name string, grpcPort int, errorChan chan error, rpcClient RPCClient) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	err := rpcClient.StopServer(ctx, ip, grpcPort)
	if err != nil {
		message := fmt.Errorf("could not stop server %s: %w", name, err)
		errorChan <- message
		return
	}
	errorChan <- nil
}

func getServerState(ip string, name string, grpcPort int, errorChan chan error, rpcClient RPCClient) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	err := rpcClient.GetState(ctx, ip, grpcPort)
	if err != nil {
		message := fmt.Errorf("could not get server state %s: %w", name, err)
		errorChan <- message
		return
	}
	errorChan <- nil
}

func updateGDCStatistics(gdc duck.GDCConfiguration, queries database.Querier, run int, rpcClient RPCClient) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	results, err := rpcClient.FetchRunStatistics(ctx, gdc.IP, gdc.GRPCPort)
	if err != nil {
		message := fmt.Errorf("could not fetch run statistics from GDC %s: %w", gdc.Name, err)
		logger.Slog.Error(message.Error())
		return
	}

	results.Host = gdc.Name
	logger.Summary(results)

	err = insertGDCEvents(queries, run, gdc.ID, results.Events)
	if err != nil {
		message := fmt.Errorf("could not store event statistics from GDC %s: %w", gdc.Name, err)
		logger.Slog.Error(message.Error())
	}

	err = insertGDCBytes(queries, run, gdc.ID, results.Bytes)
	if err != nil {
		message := fmt.Errorf("could not store bytes statistics from GDC %s: %w", gdc.Name, err)
		logger.Slog.Error(message.Error())
	}

	err = insertGDCErrorCount(queries, run, gdc.ID, results.Errors)
	if err != nil {
		message := fmt.Errorf("could not store error statistics from GDC %s: %w", gdc.Name, err)
		logger.Slog.Error(message.Error())
	}
}

func updateLDCStatistics(ldc duck.LDCConfiguration, queries database.Querier, run int, rpcClient RPCClient) {
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	results, err := rpcClient.FetchRunStatistics(ctx, ldc.IP, ldc.GRPCPort)
	if err != nil {
		message := fmt.Errorf("could not fetch run statistics from LDC %s: %w", ldc.Name, err)
		logger.Slog.Error(message.Error())
		return
	}

	results.Host = ldc.Name
	logger.Summary(results)

	err = insertLDCEvents(queries, run, ldc.ID, results.Events)
	if err != nil {
		message := fmt.Errorf("could not store event statistics from LDC %s: %w", ldc.Name, err)
		logger.Slog.Error(message.Error())
	}

	err = insertLDCBytes(queries, run, ldc.ID, results.Bytes)
	if err != nil {
		message := fmt.Errorf("could not store bytes statistics from LDC %s: %w", ldc.Name, err)
		logger.Slog.Error(message.Error())
	}

	err = insertLDCErrorCount(queries, run, ldc.ID, results.Errors)
	if err != nil {
		message := fmt.Errorf("could not store error statistics from LDC %s: %w", ldc.Name, err)
		logger.Slog.Error(message.Error())
	}
}

func startGDCs(gdcs []duck.GDCConfiguration, rpcClient RPCClient) bool {
	// returns true if success
	nGDCs := len(gdcs)
	errorChan := make(chan error, nGDCs)
	for i := 0; i < nGDCs; i++ {
		go startServer(gdcs[i].IP, gdcs[i].Name, gdcs[i].GRPCPort, errorChan, rpcClient)
	}
	success := true
	for i := 0; i < nGDCs; i++ {
		err := <-errorChan
		if err != nil {
			logger.Slog.Error(err.Error())
			success = false
		}
	}
	return success
}

func startLDCs(ldcs []duck.LDCConfiguration, rpcClient RPCClient) bool {
	// returns true if success
	nLDCs := len(ldcs)
	errorChan := make(chan error, nLDCs)
	for i := 0; i < nLDCs; i++ {
		go startServer(ldcs[i].IP, ldcs[i].Name, ldcs[i].GRPCPort, errorChan, rpcClient)
	}
	success := true
	for i := 0; i < nLDCs; i++ {
		err := <-errorChan
		if err != nil {
			logger.Slog.Error(err.Error())
			success = false
		}
	}
	return success
}

func stopGDCs(gdcs []duck.GDCConfiguration, rpcClient RPCClient) bool {
	// returns true if success
	nGDCs := len(gdcs)
	errorChan := make(chan error, nGDCs)
	for i := 0; i < nGDCs; i++ {
		go stopServer(gdcs[i].IP, gdcs[i].Name, gdcs[i].GRPCPort, errorChan, rpcClient)
	}
	success := true
	for i := 0; i < nGDCs; i++ {
		err := <-errorChan
		if err != nil {
			logger.Slog.Error(err.Error())
			success = false
		}
	}
	return success
}

func stopLDCs(ldcs []duck.LDCConfiguration, rpcClient RPCClient) bool {
	// returns true if success
	nLDCs := len(ldcs)
	errorChan := make(chan error, nLDCs)
	for i := 0; i < nLDCs; i++ {
		go stopServer(ldcs[i].IP, ldcs[i].Name, ldcs[i].GRPCPort, errorChan, rpcClient)
	}
	success := true
	for i := 0; i < nLDCs; i++ {
		err := <-errorChan
		if err != nil {
			logger.Slog.Error(err.Error())
			success = false
		}
	}
	return success
}

func getLDCStates(ldcs []duck.LDCConfiguration, rpcClient RPCClient) bool {
	// returns true if success
	nLDCs := len(ldcs)
	errorChan := make(chan error, nLDCs)
	for i := 0; i < nLDCs; i++ {
		go getServerState(ldcs[i].IP, ldcs[i].Name, ldcs[i].GRPCPort, errorChan, rpcClient)
	}
	success := true
	for i := 0; i < nLDCs; i++ {
		err := <-errorChan
		if err != nil {
			logger.NonStoppingError(err.Error())
			success = false
		}
	}
	return success
}

func getGDCStates(gdcs []duck.GDCConfiguration, rpcClient RPCClient) bool {
	// returns true if success
	nGDCs := len(gdcs)
	errorChan := make(chan error, nGDCs)
	for i := 0; i < nGDCs; i++ {
		go getServerState(gdcs[i].IP, gdcs[i].Name, gdcs[i].GRPCPort, errorChan, rpcClient)
	}
	success := true
	for i := 0; i < nGDCs; i++ {
		err := <-errorChan
		if err != nil {
			logger.NonStoppingError(err.Error())
			success = false
		}
	}
	return success
}

func updateAllGDCStatistics(gdcs []duck.GDCConfiguration, queries database.Querier, run int, rpcClient RPCClient) {
	nGDCs := len(gdcs)
	doneChan := make(chan struct{}, nGDCs)
	for i := 0; i < nGDCs; i++ {
		go func(gdc duck.GDCConfiguration) {
			defer func() { doneChan <- struct{}{} }()
			updateGDCStatistics(gdc, queries, run, rpcClient)
		}(gdcs[i])
	}
	for i := 0; i < nGDCs; i++ {
		<-doneChan
	}
}

func updateAllLDCStatistics(ldcs []duck.LDCConfiguration, queries database.Querier, run int, rpcClient RPCClient) {
	nLDCs := len(ldcs)
	doneChan := make(chan struct{}, nLDCs)
	for i := 0; i < nLDCs; i++ {
		go func(ldc duck.LDCConfiguration) {
			defer func() { doneChan <- struct{}{} }()
			updateLDCStatistics(ldc, queries, run, rpcClient)
		}(ldcs[i])
	}
	for i := 0; i < nLDCs; i++ {
		<-doneChan
	}
}
