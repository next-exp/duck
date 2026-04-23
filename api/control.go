package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

var stopProcessesMutex sync.Mutex

// writeFileFn is injectable for testing
var writeFileFn = os.WriteFile

func startProcesses(s *DuckAPIServer) {
	configuration, err := duck.ReadConfigurationFromDB(s.queries)
	if err != nil {
		message := fmt.Errorf("could not read configuration from database: %w", err)
		logger.Slog.Error(message.Error())
		return
	}
	enabledLDCs := duck.EnabledLDCs(configuration.LDCs)
	enabledGDCs := duck.EnabledGDCs(configuration.GDCs)

	success := sendPingToAllLDCs(enabledLDCs, s.rpcClient)
	if !success {
		return
	}

	success = startGDCs(enabledGDCs, s.rpcClient)
	if !success {
		return
	}

	// Wait for GDCs to start their TCP data listeners before starting LDCs
	// GDCs need time to: receive StartRun, start goroutine, read DB config, and listen on TCP port
	time.Sleep(1 * time.Second)

	success = startLDCs(enabledLDCs, s.rpcClient)
	if !success {
		return
	}

	runNumber, err := getRun(s.queries)
	if err != nil {
		message := fmt.Errorf("could not get run number: %w", err)
		logger.Slog.Error(message.Error())
		return
	}
	err = updateRunStartTime(s.queries, int(runNumber.ID))
	if err != nil {
		message := fmt.Errorf("could not store start time for run %d: %w", runNumber.ID, err)
		logger.Slog.Error(message.Error())
	}

	// Update /tmp/date_runnumber.txt for Raul Java
	if !s.devVersion {
		fileContent := fmt.Sprintf("%d", runNumber.ID)
		err = writeFileFn("/tmp/date_runnumber.txt", []byte(fileContent), 0644)
		if err != nil {
			message := fmt.Errorf("could not write to /tmp/date_runnumber.txt: %w", err)
			logger.Slog.Error(message.Error())
		}
	}
}

func stopProcesses(s *DuckAPIServer) {
	configuration, err := duck.ReadConfigurationFromDB(s.queries)
	if err != nil {
		message := fmt.Errorf("could not read configuration from database: %w", err)
		logger.Slog.Error(message.Error())
		return
	}
	enabledLDCs := duck.EnabledLDCs(configuration.LDCs)

	// Stop only LDCs. GDCs will stop if there are no remaining LDC connections
	stopLDCs(enabledLDCs, s.rpcClient)

	stopProcessesMutex.Lock()
	defer stopProcessesMutex.Unlock() // Unlock the mutex when the function exits

	runNumber, err := getRun(s.queries)
	if err != nil {
		message := fmt.Errorf("could not get run number: %w", err)
		logger.Slog.Error(message.Error())
		return
	}

	// The has to be previously started and it should not have a stop time yet
	if runNumber.Start.Valid && !runNumber.Stop.Valid {
		err = updateRunStopTime(s.queries, int(runNumber.ID))
		if err != nil {
			message := fmt.Errorf("could not store stop time for run %d: %w", runNumber.ID, err)
			logger.Slog.Error(message.Error())
		}

		err = updateRunStatistics(s, int(runNumber.ID))
		if err != nil {
			message := fmt.Errorf("could not store run statistics for run %d: %w", runNumber.ID, err)
			logger.Slog.Error(message.Error())
		}

		err = addRun(s.queries)
		if err != nil {
			message := fmt.Errorf("could not store next run number in DB: %w", err)
			logger.Slog.Error(message.Error())
		}
	}
}

func forceStopProcesses(s *DuckAPIServer) {
	configuration, err := duck.ReadConfigurationFromDB(s.queries)
	if err != nil {
		message := fmt.Errorf("could not read configuration from database: %w", err)
		logger.Slog.Error(message.Error())
		return
	}
	enabledLDCs := duck.EnabledLDCs(configuration.LDCs)
	enabledGDCs := duck.EnabledGDCs(configuration.GDCs)

	stopGDCs(enabledGDCs, s.rpcClient)
	stopLDCs(enabledLDCs, s.rpcClient)

	runNumber, err := getRun(s.queries)
	if err != nil {
		message := fmt.Errorf("could not get run number: %w", err)
		logger.Slog.Error(message.Error())
		return
	}

	err = updateRunStopTime(s.queries, int(runNumber.ID))
	if err != nil {
		message := fmt.Errorf("could not store stop time for run %d: %w", runNumber.ID, err)
		logger.Slog.Error(message.Error())
	}

	// Force stop is used when there is some error,
	// many times, when that happens, there will be errors to getting statistics
	// so we will not store them
	//err = updateRunStatistics(s, runNumber)
	//if err != nil {
	//	message := fmt.Errorf("could not store run statistics for run %d: %w", runNumber, err)
	//	logger.Slog.Error(message.Error())
	//}

	err = addRun(s.queries)
	if err != nil {
		message := fmt.Errorf("could not store next run number in DB: %w", err)
		logger.Slog.Error(message.Error())
	}
}

func updateRunStatistics(s *DuckAPIServer, run int) error {
	configuration, err := duck.ReadConfigurationFromDB(s.queries)
	if err != nil {
		message := fmt.Errorf("could not read configuration from database: %w", err)
		return message
	}
	enabledLDCs := duck.EnabledLDCs(configuration.LDCs)
	enabledGDCs := duck.EnabledGDCs(configuration.GDCs)

	for i := 0; i < len(enabledGDCs); i++ {
		updateGDCStatistics(enabledGDCs[i], s.queries, run, s.rpcClient)
	}

	for i := 0; i < len(enabledLDCs); i++ {
		updateLDCStatistics(enabledLDCs[i], s.queries, run, s.rpcClient)
	}
	return nil
}

func getProcessStates(s *DuckAPIServer) {
	configuration, err := duck.ReadConfigurationFromDB(s.queries)
	if err != nil {
		message := fmt.Errorf("could not read configuration from database: %w", err)
		logger.Slog.Error(message.Error())
		return
	}
	enabledLDCs := duck.EnabledLDCs(configuration.LDCs)
	enabledGDCs := duck.EnabledGDCs(configuration.GDCs)
	getLDCStates(enabledLDCs, s.rpcClient)
	getGDCStates(enabledGDCs, s.rpcClient)
}
