package main

import (
	"encoding/binary"
	"fmt"
	"slices"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

type EquipmentData struct {
	EventID     int
	EquipmentID int
	Data        []byte
	Error       bool
}

func buildEquipmentData(s *server, bufferData RingBufferData, fragments *[][]byte, bufferPositions *[]int,
	equipmentConfiguration duck.Equipment, equipmentDataChannel chan EquipmentData,
	eventID *int, evtError *bool, expectedSequenceCounter *int, bufferTracking *RingBufferTracking) {

	data := bufferData.data
	*fragments = append(*fragments, data)
	*bufferPositions = append(*bufferPositions, bufferData.position)

	if len(data) == 4 {
		end_word := binary.LittleEndian.Uint32(data)
		if end_word == 0xfafafafa {
			subevent := slices.Concat(*fragments...)
			// Once processed, release all buffer positions
			for i := 0; i < len(*bufferPositions); i++ {
				bufferTracking.ReleasePosition((*bufferPositions)[i])
			}

			equipmentData := addEquipmentHeader(s.logger, subevent, equipmentConfiguration)
			*fragments = nil
			*bufferPositions = nil
			eqData := EquipmentData{
				EventID:     *eventID,
				EquipmentID: equipmentConfiguration.ID,
				Data:        equipmentData,
				Error:       *evtError,
			}

			// If there was a sequence counter error, do not send the data
			if !*evtError {
				equipmentDataChannel <- eqData
			}

			(*eventID)++
			*evtError = false
			*expectedSequenceCounter = 0
		}
	} else {
		// Validate minimum packet length before reading sequence counter
		if len(data) < 4 {
			message := fmt.Sprintf("Error on evt %d! Packet too short: %d bytes, expected at least 4 bytes. eq %d", *eventID, len(data), equipmentConfiguration.ID)
			*evtError = true
			s.metrics.packetErrorCounter.Inc()
			s.stopOnLocalError(message)
			return
		}
		// Read sequence counter ([0,0,0,1], [0,0,0,2], ...)
		dataCounter := binary.BigEndian.Uint32(data[0:4])
		if dataCounter != uint32(*expectedSequenceCounter) {
			message := fmt.Sprintf("Error on evt %d! Packet mismatch, expected sequence counter %d, found %d. Length %d, eq %d", *eventID, *expectedSequenceCounter, dataCounter, len(data), equipmentConfiguration.ID)
			*evtError = true
			s.metrics.packetErrorCounter.Inc()

			// If we continue reading data, errors will be raised until the event is
			// closed with a FAFAFAFA sequence. It that packet is lost, the next event
			// will be corrupted as well.

			// If FAFAFAFA is lost, probably everything will fail, we won't know
			// how many events are lost, and the next correct event will not be sent to
			// the correct GDC

			// Stop processing
			s.stopOnLocalError(message)
		}
		*expectedSequenceCounter++
	}
}

func readPacket(s *server, recvChannel chan RingBufferData,
	equipmentConfiguration duck.Equipment, equipmentDataChannel chan EquipmentData,
	bufferTracking *RingBufferTracking) {

	var fragments [][]byte
	var bufferPositions []int
	eventID := 1
	expectedSequenceCounter := 0
	evtWithError := false

	for {
		select {
		case <-s.getContext().Done():
			s.logger.Slog.Debug("Stop reading packets")
			return
		case bufferData := <-recvChannel:
			buildEquipmentData(s, bufferData, &fragments, &bufferPositions, equipmentConfiguration,
				equipmentDataChannel, &eventID, &evtWithError, &expectedSequenceCounter, bufferTracking)
		}
	}
}
