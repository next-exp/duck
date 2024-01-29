package main

import (
	"fmt"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/maps"
)

func countEnabledLDCs(ldcs []duck.LDCConfiguration) int {
	nLDCs := 0
	for _, ldc := range ldcs {
		// If all equipments in the LDC are disabled, no data will be sent from that LDC
		nEnabledEquipments := 0
		for _, eq := range ldc.Equipments {
			if eq.Enabled {
				nEnabledEquipments++
			}
		}

		if ldc.Enabled && nEnabledEquipments > 0 {
			nLDCs++
		}
	}
	return nLDCs
}

type WriterChannels struct {
	binaryEvtCh   chan []byte
	decodedTrg1Ch chan DecodedWriterEvent
	decodedTrg2Ch chan DecodedWriterEvent
	fileCloser    FileCloserChannels
}

func readData(s *server, dataChannel chan LDCData, gdcID int, ldcs []duck.LDCConfiguration, path string) {
	// Map: EventID -> ldcID -> Data
	eventData := make(map[int]map[int][]byte)

	writeOutputEnable, _ := s.getContext().Value("writeOutput").(bool)
	decode, _ := s.getContext().Value("decode").(bool)
	writerChSize, _ := s.getContext().Value("writerChBufferSize").(int)
	decoderChSize, _ := s.getContext().Value("decoderChBufferSize").(int)
	nDecoderWorkers, _ := s.getContext().Value("decoderWorkers").(int)
	fileCloserTimeout := 300 * time.Second // Set in DB?
	maxFilesize, ok := s.getContext().Value("maxFilesize").(int)
	if !ok {
		message := fmt.Errorf("error getting max filesize from context")
		s.logger.Slog.Error(message.Error())
		return
	}
	decoderConfig, ok := s.getContext().Value("decoderConfig").(duck.DecoderConfiguration)
	if !ok {
		message := fmt.Errorf("error getting decoder configuration from context")
		s.logger.Slog.Error(message.Error())
	}
	gdcConfiguration, ok := s.getContext().Value("gdcConfiguration").(*duck.GDCConfiguration)
	if !ok {
		message := fmt.Errorf("error getting gdc configuration from context")
		s.logger.Slog.Error(message.Error())
	}

	bytesInFile := 0
	subRun := 0
	s.metrics.subRunCounter.Set(float64(subRun))
	nLDCs := countEnabledLDCs(ldcs)

	writerChannels := WriterChannels{
		binaryEvtCh:   make(chan []byte, writerChSize),
		decodedTrg1Ch: make(chan DecodedWriterEvent, writerChSize),
		decodedTrg2Ch: make(chan DecodedWriterEvent, writerChSize),
		fileCloser:    createFileCloserChannels(writerChSize),
	}

	if writeOutputEnable {
		s.logger.OutputFile(gdcConfiguration.Name, subRun)
		go binaryWriter(s, writerChannels.binaryEvtCh, subRun, path)
		s.metrics.filesOpenedCounter.Inc()
	}
	if decoderConfig.WriteData {
		s.logger.OutputFile(gdcConfiguration.Name, subRun)
		go fileCloser(s, writerChannels.fileCloser, subRun, fileCloserTimeout)
		if decoderConfig.SplitTrigger {
			go decodedWriter(s, writerChannels.decodedTrg1Ch, writerChannels.fileCloser.EvtWritten,
				writerChannels.fileCloser.CloseFiles, subRun, Trg1, path)
			go decodedWriter(s, writerChannels.decodedTrg2Ch, writerChannels.fileCloser.EvtWritten,
				writerChannels.fileCloser.CloseFiles, subRun, Trg2, path)
			s.metrics.h5FilesOpenedCounter.With(prometheus.Labels{"trigger": Trg1.String()}).Inc()
			s.metrics.h5FilesOpenedCounter.With(prometheus.Labels{"trigger": Trg2.String()}).Inc()
		} else {
			go decodedWriter(s, writerChannels.decodedTrg1Ch, writerChannels.fileCloser.EvtWritten,
				writerChannels.fileCloser.CloseFiles, subRun, Trg0, path)
			s.metrics.h5FilesOpenedCounter.With(prometheus.Labels{"trigger": Trg0.String()}).Inc()
		}
	}

	decoderJobsCh := make(chan DecoderJob, decoderChSize)
	if decode {
		launchDecoderWorkers(s, nDecoderWorkers, decoderJobsCh)
	}

	// metrics
	metricsChBufferSize, ok := s.getContext().Value("metricsChBufferSize").(int)
	if !ok {
		message := fmt.Errorf("error getting metrics channel buffer size from context")
		s.logger.Slog.Error(message.Error())
		return
	}
	metrics := make(chan int, metricsChBufferSize)
	go duck.ProcessMetrics(metrics, s.logger, s.metrics.eventCounter, s.metrics.sizeCounter, s.ctx, duck.RealTicker{})

	evtCount := 0
	for {
		select {
		case <-s.getContext().Done():
			message := fmt.Sprintf("Stopping reading data in GDC %d", gdcID)
			s.logger.Slog.Info(message)
			// Close decoder jobs channel, waiting for the workers to exhaust remaining events
			close(decoderJobsCh)
			writerChannels.fileCloser.DataEnd <- true
			close(writerChannels.binaryEvtCh)
			return
		case ldcData := <-dataChannel:
			mountEvent(s, &eventData, &ldcData, writerChannels, &bytesInFile, gdcID, nLDCs, metrics, decoderConfig, decoderJobsCh)
			if bytesInFile > maxFilesize {
				//fmt.Printf("bytesInFile: %d, maxFileSize: %d\n", bytesInFile, maxFilesize)
				subRun++
				s.metrics.subRunCounter.Set(float64(subRun))
				s.logger.OutputFile(gdcConfiguration.Name, subRun)
				if writeOutputEnable {
					// Close current files
					close(writerChannels.binaryEvtCh)

					// Open new files
					writerChannels.binaryEvtCh = make(chan []byte, writerChSize)
					go binaryWriter(s, writerChannels.binaryEvtCh, subRun, path)
					s.metrics.filesOpenedCounter.Inc()
				}
				if decoderConfig.WriteData {
					// Close current files
					writerChannels.fileCloser.DataEnd <- true

					// Open new files
					writerChannels.decodedTrg1Ch = make(chan DecodedWriterEvent, writerChSize)
					writerChannels.decodedTrg2Ch = make(chan DecodedWriterEvent, writerChSize)
					writerChannels.fileCloser = createFileCloserChannels(writerChSize)

					go fileCloser(s, writerChannels.fileCloser, subRun, fileCloserTimeout)
					if decoderConfig.SplitTrigger {
						go decodedWriter(s, writerChannels.decodedTrg1Ch, writerChannels.fileCloser.EvtWritten,
							writerChannels.fileCloser.CloseFiles, subRun, Trg1, path)
						go decodedWriter(s, writerChannels.decodedTrg2Ch, writerChannels.fileCloser.EvtWritten,
							writerChannels.fileCloser.CloseFiles, subRun, Trg2, path)
						s.metrics.h5FilesOpenedCounter.With(prometheus.Labels{"trigger": Trg1.String()}).Inc()
						s.metrics.h5FilesOpenedCounter.With(prometheus.Labels{"trigger": Trg2.String()}).Inc()
					} else {
						go decodedWriter(s, writerChannels.decodedTrg1Ch, writerChannels.fileCloser.EvtWritten,
							writerChannels.fileCloser.CloseFiles, subRun, Trg0, path)
						s.metrics.h5FilesOpenedCounter.With(prometheus.Labels{"trigger": Trg0.String()}).Inc()
					}
				}
				bytesInFile = 0
			}
			evtCount++
		}
	}
}

func mountEvent(s *server, eventData *map[int]map[int][]byte, ldcData *LDCData, writerChs WriterChannels,
	bytesInFile *int, gdcID int, nLDCs int, metrics chan int, decoderConfig duck.DecoderConfiguration,
	decoderJobsCh chan DecoderJob) {
	runNumber, _ := s.getContext().Value("runNumber").(int)
	writeOutputEnable, _ := s.getContext().Value("writeOutput").(bool)
	decode, _ := s.getContext().Value("decode").(bool)
	decoderConfig, ok := s.getContext().Value("decoderConfig").(duck.DecoderConfiguration)
	if !ok {
		message := fmt.Errorf("error getting decoder configuration from context")
		s.logger.Slog.Error(message.Error())
	}

	if _, exists := (*eventData)[ldcData.EventID]; !exists {
		(*eventData)[ldcData.EventID] = make(map[int][]byte)
		s.metrics.incompleteEventsCounter.Inc()
	}
	(*eventData)[ldcData.EventID][ldcData.ldcID] = ldcData.Data

	// Check whether we have data from all equipments
	nLDCsFound := len(maps.Keys((*eventData)[ldcData.EventID]))

	if nLDCsFound == nLDCs {
		gdcData := addGDCHeader((*eventData)[ldcData.EventID], ldcData.EventID, gdcID, runNumber)
		*bytesInFile += len(gdcData)

		// FileCloser is only used for the decoder writer.
		// if not using the decoder, this has to be skipped, otherwise it will saturate the channel
		if decoderConfig.WriteData {
			writerChs.fileCloser.EvtReceived <- ldcData.EventID
		}

		if writeOutputEnable {
			writerChs.binaryEvtCh <- gdcData
			s.metrics.evtsWaitingBinaryWriterCounter.Set(float64(len(writerChs.binaryEvtCh)))
		}

		if decode {
			s.metrics.evtsWaitingDecodeCounter.Set(float64(len(decoderJobsCh)))
			decoderJobsCh <- DecoderJob{
				DuckEventID:   ldcData.EventID,
				Data:          gdcData,
				WritersCh:     writerChs,
				DecoderConfig: decoderConfig,
			}
		}

		metrics <- len(gdcData)

		delete(*eventData, ldcData.EventID)
		s.metrics.incompleteEventsCounter.Dec()
	}
}
