package main

import (
	"fmt"
	"time"
)

type FileCloserChannels struct {
	EvtReceived  chan int
	EvtDecoded   chan int
	EvtWritten   chan int
	EvtWithError chan int
	DataEnd      chan bool
	CloseFiles   chan bool
}

func fileCloser(
	s *server,
	channels FileCloserChannels,
	subRun int,
	timeoutDuration time.Duration,
) {
	events := make([]int, 0)
	eventsDecoded := make([]int, 0)
	eventsWritten := make([]int, 0)
	eventsWithError := make([]int, 0)
	dataEnded := false

	closeTimer := time.NewTimer(10 * time.Second)
	closeTimer.Stop()

	loop := true

	// Wait for events to be written
	for loop {
		select {
		case evt := <-channels.EvtReceived:
			events = append(events, evt)
		case evt := <-channels.EvtDecoded:
			eventsDecoded = append(eventsDecoded, evt)
		case evt := <-channels.EvtWritten:
			eventsWritten = append(eventsWritten, evt)
		case evt := <-channels.EvtWithError:
			eventsWithError = append(eventsWithError, evt)
		case <-channels.DataEnd:
			dataEnded = true
			closeTimer.Reset(timeoutDuration)
		case <-closeTimer.C:
			// After timeout once the last event has been received, close the files
			message := fmt.Sprintf("Possible event loss at file %d, file was closed after timeout", subRun)
			s.logger.NonStoppingError(message)
			// Send two times, one for each trigger
			channels.CloseFiles <- true
			channels.CloseFiles <- true
			loop = false
		}

		if dataEnded {
			nEvents := len(events)
			nEventsDecoded := len(eventsDecoded)
			nEventsWritten := len(eventsWritten)
			nEventsWithError := len(eventsWithError)

			if (nEvents == (nEventsDecoded + nEventsWithError)) && (nEventsDecoded == nEventsWritten) {
				// Send two times, one for each trigger
				channels.CloseFiles <- true
				channels.CloseFiles <- true
				break
			}
		}
	}
}

func createFileCloserChannels(chSize int) FileCloserChannels {
	fileCloser := FileCloserChannels{
		EvtReceived:  make(chan int, chSize),
		EvtDecoded:   make(chan int, chSize),
		EvtWritten:   make(chan int, chSize),
		EvtWithError: make(chan int, chSize),
		DataEnd:      make(chan bool, chSize),
		CloseFiles:   make(chan bool, chSize),
	}
	return fileCloser
}
