package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/maps"
)

func ldcServer(s *server, configFilename string) {
	for {
		// Wait for run to start
		s.runActiveCond.L.Lock()
		for !s.runActive {
			s.runActiveCond.Wait()
		}
		s.runActiveCond.L.Unlock()

		// Re-read configuration for each run (in case it changed)
		configuration, err := duck.ReadConfiguration(configFilename, nil, nil)
		if err != nil {
			message := fmt.Sprintf("Error reading configuration %s", err.Error())
			s.logger.Slog.Error(message)
			s.runActiveCond.L.Lock()
			s.runActive = false
			s.runActiveCond.L.Unlock()
			s.setState(duck.INITIALIZED)
			continue
		}
		ldcConfiguration, err := duck.GetLDCConfiguration(configuration.LDCs, s.ldcName)
		if err != nil {
			message := fmt.Sprintf("Error reading LDC configuration %s", err.Error())
			s.logger.Slog.Error(message)
			s.runActiveCond.L.Lock()
			s.runActive = false
			s.runActiveCond.L.Unlock()
			s.setState(duck.INITIALIZED)
			continue
		}

		// Protect context modifications with ctxMutex
		s.ctxMutex.Lock()
		s.ctx = context.WithValue(s.ctx, "ldcConfiguration", ldcConfiguration)
		s.ctx = context.WithValue(s.ctx, "runNumber", configuration.RunNumber)
		s.ctx = context.WithValue(s.ctx, "metricsChBufferSize", configuration.Duck.MetricsCh)
		s.ctx = context.WithValue(s.ctx, "packetBufferSize", configuration.Duck.PacketSize)
		s.ctx = context.WithValue(s.ctx, "nPacketsInBuffer", configuration.Duck.NPacketsInBuffer)
		s.ctx = context.WithValue(s.ctx, "bufferTimeout", configuration.Duck.BufferTimeout)
		s.ctxMutex.Unlock()

		duck.AddRunNumberToLogger(&s.logger, configuration.RunNumber)

		// Get GDC connection pool
		enabledGDCs := duck.EnabledGDCs(configuration.GDCs)
		gdcConnections, err := gdcGDCConnetionPool(enabledGDCs)
		if err != nil {
			s.logger.Slog.Error(err.Error())
			s.runActiveCond.L.Lock()
			s.runActive = false
			s.runActiveCond.L.Unlock()
			s.setState(duck.INITIALIZED)
			continue
		}

		// This should be buffered
		equipmentChBufferSize := configuration.Duck.EquipmentCh
		equipmentDataChannel := make(chan EquipmentData, equipmentChBufferSize)
		receiveChannels := make(map[int]chan RingBufferData)
		nEnabledEquipments := int(0)
		nPacketsInBuffer := configuration.Duck.NPacketsInBuffer

		var wg sync.WaitGroup

		// Channel to signal when equipment sockets are ready
		readyChan := make(chan struct{}, len(ldcConfiguration.Equipments))

		for i := 0; i < len(ldcConfiguration.Equipments); i++ {
			recvChBufferSize := configuration.Duck.ReceiveCh
			recvChannel := make(chan RingBufferData, recvChBufferSize)
			equipment := ldcConfiguration.Equipments[i]
			// Only read data from enabled equipments
			if equipment.Enabled {
				bufferTracking := NewRingBufferTracking(nPacketsInBuffer)
				receiveChannels[equipment.ID] = recvChannel
				wg.Add(2)
				go func(eq duck.Equipment, ch chan RingBufferData, bt *RingBufferTracking) {
					defer wg.Done()
					readoutEquipment(s, eq, ch, bt, readyChan)
				}(equipment, recvChannel, bufferTracking)
				go func(eq duck.Equipment, ch chan RingBufferData, bt *RingBufferTracking) {
					defer wg.Done()
					readPacket(s, ch, eq, equipmentDataChannel, bt)
				}(equipment, recvChannel, bufferTracking)
				nEnabledEquipments++
			}
		}

		wg.Add(2)
		go func() {
			defer wg.Done()
			updateChannelGauges(s, equipmentDataChannel, receiveChannels)
		}()

		s.ctxMutex.Lock()
		s.ctx = context.WithValue(s.ctx, "nEnabledEquipments", nEnabledEquipments)
		s.ctxMutex.Unlock()

		go func() {
			defer wg.Done()
			readSubEvents(s, &gdcConnections, equipmentDataChannel, ldcConfiguration, &enabledGDCs)
		}()

		// Wait for all equipment sockets to be ready before transitioning to RUNNING
		for i := 0; i < nEnabledEquipments; i++ {
			select {
			case <-readyChan:
				// Equipment socket is ready
			case <-s.getContext().Done():
				// Context cancelled during startup
				s.logger.Slog.Warn("Context cancelled while waiting for equipment sockets")
			}
		}

		// Transition to RUNNING state once all sockets are ready
		s.setState(duck.RUNNING)

		// Wait for stop signal
		s.runActiveCond.L.Lock()
		for s.runActive {
			s.runActiveCond.Wait()
		}
		s.runActiveCond.L.Unlock()

		// Cancel context to signal all goroutines to stop
		s.ctxMutex.Lock()
		s.cancelCtx()
		s.ctxMutex.Unlock()

		// Wait for goroutines to finish
		s.logger.Slog.Info("Stopping LDC")
		wg.Wait()

		// Reset state to INITIALIZED for next run
		s.setState(duck.INITIALIZED)
	}
}

func readSubEvents(s *server, gdcConnections *[]net.Conn, equipmentDataChannel chan EquipmentData,
	ldcConfiguration *duck.LDCConfiguration, gdcs *[]duck.GDCConfiguration) {

	metricsChBufferSize, ok := s.getContext().Value("metricsChBufferSize").(int)
	if !ok {
		message := fmt.Errorf("error getting metrics channel buffer size from context")
		s.logger.Slog.Error(message.Error())
		return
	}
	metrics := make(chan int, metricsChBufferSize)
	go duck.ProcessMetrics(metrics, s.logger, s.metrics.eventCounter, s.metrics.sizeCounter, s.getContext(), duck.RealTicker{})

	// Map: EventID -> EquipmentID -> Data
	eventData := make(map[int]map[int][]byte)
	currentGDC := 0
	nGDCs := len(*gdcConnections)
	for {
		select {
		case <-s.getContext().Done():
			message := fmt.Sprintf("Stopping mounting subevent on LDC %d", ldcConfiguration.ID)
			s.logger.Slog.Info(message)
			for i := 0; i < nGDCs; i++ {
				message := fmt.Sprintf("Close connection from LDC %d to GDC %d", ldcConfiguration.ID, (*gdcs)[i].ID)
				s.logger.Slog.Info(message)
				(*gdcConnections)[i].Close()
			}
			return
		case data := <-equipmentDataChannel:
			if !data.Error {
				eventMounted, err := mountSubEvent(s, &eventData, &data, ldcConfiguration, (*gdcConnections)[currentGDC], currentGDC, metrics)
				// In case of error, stop processing
				if err != nil {
					message := fmt.Sprintf("Error mounting subevent: %s", err.Error())
					s.stopOnLocalError(message)
				}
				// Send in round robin to GDCs
				if eventMounted {
					currentGDC = (currentGDC + 1) % nGDCs
				}
			} else {
				// If there is an error due to packet loss,
				// skip the event and move to the next GDC
				previousGDC := currentGDC
				currentGDC = (currentGDC + 1) % nGDCs
				message := fmt.Sprintf("Error on evt %d. Skipping one GDC (%s), moving to %s",
					data.EventID, (*gdcs)[previousGDC].Name, (*gdcs)[currentGDC].Name)
				s.logger.Slog.Error(message)
			}
		}
	}
}

func mountSubEvent(s *server, eventData *map[int]map[int][]byte, data *EquipmentData, ldcConfiguration *duck.LDCConfiguration,
	gdcConnection net.Conn, gdcId int, metrics chan int) (bool, error) {
	if _, exists := (*eventData)[data.EventID]; !exists {
		(*eventData)[data.EventID] = make(map[int][]byte)
		s.metrics.incompleteEventsCounter.Inc()
	}
	(*eventData)[data.EventID][data.EquipmentID] = data.Data

	err := error(nil)
	eventMounted := false

	runNumber, success := s.getContext().Value("runNumber").(int)
	if !success {
		err = errors.New("error getting run number from context")
		return false, err
	}
	nEnabledEquipments, _ := s.getContext().Value("nEnabledEquipments").(int)
	if !success {
		err = errors.New("error getting number of enabled equipments from context")
		return false, err
	}

	// Check whether we have data from all equipments
	nEquipments := len(maps.Keys((*eventData)[data.EventID]))
	if nEquipments == nEnabledEquipments {
		ldcData := addLDCHeader((*eventData)[data.EventID], data.EventID, *ldcConfiguration, runNumber)
		err = sendDataToGDC(s, ldcData, metrics, gdcConnection, gdcId)
		delete(*eventData, data.EventID)
		eventMounted = true
		s.metrics.incompleteEventsCounter.Dec()
	}
	return eventMounted, err
}

func sendDataToGDC(s *server, ldcData []byte, metrics chan int, gdcConnection net.Conn, gdcId int) error {
	start := time.Now()
	nBytes, err := gdcConnection.Write(ldcData)
	duration := time.Since(start)
	speed := float64(nBytes) / float64(duration.Milliseconds()) * 1000 / 1024 / 1024
	gdcIdStr := fmt.Sprintf("%d", gdcId)
	s.metrics.timeToGdcHistogram.With(prometheus.Labels{"gdc": gdcIdStr}).Observe(speed)
	metrics <- nBytes
	return err
}

func updateChannelGauges(s *server, equipmentDataChannel chan EquipmentData, receiveChannels map[int]chan RingBufferData) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			s.metrics.equipmentDataChannelCounter.Set(float64(len(equipmentDataChannel)))
			for equipmentID, channel := range receiveChannels {
				id := fmt.Sprintf("%d", equipmentID)
				s.metrics.receiveChannelCounters.With(prometheus.Labels{"equipment": id}).Set(float64(len(channel)))
			}
		case <-s.getContext().Done():
			return
		}
	}
}
