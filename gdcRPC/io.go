package main

import (
	"fmt"
	"os"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

func getDecodedOutputFilename(s *server, subrun int, trigger int, path string) (string, error) {
	var fname string
	run, ok := s.getContext().Value("runNumber").(int)
	if !ok {
		message := fmt.Errorf("error getting run number from context")
		s.logger.Slog.Error(message.Error())
		return fname, message
	}
	gdcConfiguration, ok := s.getContext().Value("gdcConfiguration").(*duck.GDCConfiguration)
	if !ok {
		message := fmt.Errorf("error getting gdc configuration from context")
		s.logger.Slog.Error(message.Error())
		return fname, message
	}
	fname = fmt.Sprintf("%s/run_%d_%04d_ldc%d_trg%d.waveforms.h5", path, run, subrun, gdcConfiguration.ID, trigger)
	return fname, nil
}

func openFile(s *server, subrun int, path string) (*os.File, error) {
	run, ok := s.getContext().Value("runNumber").(int)
	if !ok {
		message := fmt.Errorf("error getting run number from context")
		s.logger.Slog.Error(message.Error())
		return nil, message
	}
	gdcConfiguration, ok := s.getContext().Value("gdcConfiguration").(*duck.GDCConfiguration)
	if !ok {
		message := fmt.Errorf("error getting gdc configuration from context")
		s.logger.Slog.Error(message.Error())
		return nil, message
	}
	experiment, ok := s.getContext().Value("experiment").(string)
	if !ok {
		message := fmt.Errorf("error getting experiment name from context")
		s.logger.Slog.Error(message.Error())
		return nil, message
	}

	// This is using host instead of name for compatilibity with the NEXT100 processing with TOPI
	//filename := fmt.Sprintf("%s/run_%d.%s.%s.%05d.rd", path, run, gdcConfiguration.Name, experiment, subrun)
	filename := fmt.Sprintf("%s/run_%d.%s.%s.%05d.rd", path, run, gdcConfiguration.Host, experiment, subrun)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", path, err)
	}

	f, err := os.Create(filename)
	return f, err
}

func writeBinaryData(s *server, data []byte, writer fileWriter) error {
	start := time.Now()
	_, err := writer.Write(data)
	duration := time.Since(start)
	s.metrics.writeTimeHistogram.Observe(float64(duration.Milliseconds()))
	return err
}

func stopOnIOError(s *server, err error) {
	message := fmt.Errorf("IO error %w", err)
	s.stopOnLocalError(message.Error())
}
