package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"slices"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

type LDCData struct {
	EventID int
	ldcID   int
	Data    []byte
}

func handleConnection(s *server, conn net.Conn, dataChannel chan LDCData,
	ldcConnectionCloseChan chan net.Addr) {
	message := fmt.Sprintf("handle connection %s", conn.RemoteAddr().String())
	s.logger.Slog.Debug(message)

	var data [][]byte
	count := uint32(0)
	size := uint32(0)
	expectedEventID := -1

	ldc := 0

	for {
		select {
		case <-s.getContext().Done():
			// All connections are already closed from the listener...
			return
		default:
			err := readLDCData(s, conn, &data, dataChannel, &count, &size,
				&expectedEventID, &ldc)
			switch err {
			case nil:
				continue
			case io.EOF:
				s.logger.Slog.Debug("closing connection, received EOF from LDC")
				//cancelCtx, _ := s.ctx.Value("cancelCtx").(context.CancelFunc)
				//cancelCtx()
				ldcConnectionCloseChan <- conn.RemoteAddr()
				return
			default:
				message := fmt.Errorf("error reading data from LDC: %w", err)
				s.logger.Slog.Error(message.Error())
				// Maybe we should stop processing here
				continue
			}
		}
	}
}

func readLDCData(s *server, conn net.Conn, data *[][]byte, dataChannel chan LDCData, count *uint32,
	size *uint32, expectedEventID *int, ldc *int) error {
	tmp := make([]byte, 9600)
	nRead, err := conn.Read(tmp)
	if err != nil {
		return err
	}

	processData(s, nRead, tmp, data, dataChannel, count, size, expectedEventID, ldc)
	return nil
}

func processData(s *server, nRead int, tmp []byte, data *[][]byte, dataChannel chan LDCData,
	count *uint32, size *uint32, expectedEventID *int, ldc *int) error {
	nGDCs, ok := s.getContext().Value("nGDCs").(int)
	if !ok {
		message := fmt.Errorf("error getting nGDCs from context: missing or invalid type")
		s.logger.Slog.Error(message.Error())
		s.metrics.evtErrorCounter.Inc()
		return message
	}

	if *size == 0 {
		wholeDataSlice := append(*data, tmp[:nRead])
		wholeData := slices.Concat((wholeDataSlice)...)

		// Read the size of the event when we have the first 4 bytes
		if len(wholeData) >= 4 {
			*size = binary.LittleEndian.Uint32(wholeData[:4])
		} else {
			*data = append(*data, tmp[:nRead])
			*count += uint32(nRead)
			return nil
		}
	}
	remaining := *size - *count

	if nRead >= int(remaining) {
		*data = append(*data, tmp[:remaining])
		wholeData := slices.Concat((*data)...)

		// Validate magic number before processing
		magicNumber := binary.LittleEndian.Uint32(wholeData[4:8])
		if uint32(magicNumber) != uint32(duck.EVENT_MAGIC_NUMBER) {
			message := fmt.Errorf("invalid magic number 0x%08x: expected 0x%08x", magicNumber, duck.EVENT_MAGIC_NUMBER)
			s.metrics.evtErrorCounter.Inc()
			s.stopOnLocalError(message.Error())
			*count = 0
			*data = nil
			*size = 0
			return message
		}

		// Ensure we have enough data for the header
		if len(wholeData) < 80 {
			message := fmt.Errorf("incomplete event header: %d bytes (minimum 80)", len(wholeData))
			s.metrics.evtErrorCounter.Inc()
			s.stopOnLocalError(message.Error())
			*count = 0
			*data = nil
			*size = 0
			return message
		}

		eventID := binary.LittleEndian.Uint32(wholeData[24:28])
		ldcID := binary.LittleEndian.Uint32(wholeData[64:68])
		*ldc = int(ldcID)

		// Save the first event ID received by this GDC
		if *expectedEventID == -1 {
			*expectedEventID = int(eventID)
		}

		if eventID != uint32(*expectedEventID) {
			// Stop processing if the event ID is not the expected one
			message := fmt.Sprintf("GDC Error, expected event %d, found %d, from LDC %d", *expectedEventID, eventID, ldcID)
			s.metrics.evtErrorCounter.Inc()
			s.stopOnLocalError(message)
			*expectedEventID = int(eventID)
		}

		// Send data to the next step
		ldcData := LDCData{
			EventID: int(eventID),
			ldcID:   int(ldcID),
			Data:    wholeData,
		}
		dataChannel <- ldcData

		// Reset everything for the next event
		*count = 0
		*data = nil
		*size = 0
		(*expectedEventID) = (*expectedEventID) + nGDCs

		// If there is more data in the buffer, process it
		nReadUpdated := nRead - int(remaining)
		if nReadUpdated > 0 {
			processData(s, nReadUpdated, tmp[remaining:nRead], data, dataChannel, count, size, expectedEventID, ldc)
		}
	} else {
		*data = append(*data, tmp[:nRead])
		*count += uint32(nRead)
	}

	return nil
}
