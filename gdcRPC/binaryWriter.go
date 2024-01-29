package main

import (
	"fmt"
	"os"
)

// fileWriter interface defines the methods we need for writing binary data
// This allows us to test with mock implementations
type fileWriter interface {
	Write(data []byte) (int, error)
	Close() error
}

// osFileWrapper wraps *os.File to implement the fileWriter interface
type osFileWrapper struct {
	file *os.File
}

func (w *osFileWrapper) Write(data []byte) (int, error) {
	return w.file.Write(data)
}

func (w *osFileWrapper) Close() error {
	return w.file.Close()
}

func binaryWriter(s *server, dataCh chan []byte, filenumber int, path string) {
	var fd *os.File
	var err error
	fd, err = openFile(s, filenumber, path)
	if err != nil {
		stopOnIOError(s, err)
		return
	}

	// Wrap the os.File to implement fileWriter interface
	writer := &osFileWrapper{file: fd}
	binaryWriterWithFile(s, dataCh, writer)
}

// binaryWriterWithFile is the core logic that can accept any fileWriter implementation
// This allows for easy testing with mock writers
func binaryWriterWithFile(s *server, dataCh chan []byte, writer fileWriter) {
	// Ensure writer is always closed, even on error
	defer func() {
		writer.Close()
		s.metrics.filesOpenedCounter.Dec()
	}()

	loop := true
	for loop {
		select {
		case <-s.getContext().Done():
			// Context cancelled
			loop = false
		case data, ok := <-dataCh:
			if !ok {
				// Channel closed
				loop = false
			} else {
				err := writeBinaryData(s, data, writer)
				if err != nil {
					message := fmt.Errorf("error writing GDC data: %w", err)
					s.logger.Slog.Error(message.Error())
					return
				}
			}
		}
	}
}
