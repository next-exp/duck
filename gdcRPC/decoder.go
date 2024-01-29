package main

import (
	"fmt"
	"os"
	"time"

	duck "github.com/jmbenlloch/next_duck/pkg"
	decoder "github.com/next-exp/decoder_go/pkg"
	"github.com/prometheus/client_golang/prometheus"
)

type DecoderLogger struct {
	logger duck.DuckLogger
}

func (l DecoderLogger) Info(message string, module string) {
	l.logger.Slog.Info(message)
	//fmt.Println(message)
}

func (l DecoderLogger) Error(message string) {
	l.logger.NonStoppingError(message)
	//fmt.Println(message)
}

func parseDecoderConfiguration(s *server) (decoder.Configuration, error) {
	decoderConfigDB, ok := s.getContext().Value("decoderConfig").(duck.DecoderConfiguration)
	if !ok {
		message := fmt.Errorf("error getting decoder configuration from context")
		s.logger.Slog.Error(message.Error())
		return decoder.Configuration{}, message
	}
	configuration := decoder.Configuration{}
	configuration.MaxEvents = 1000000000
	configuration.Verbosity = 1
	configuration.ExtTrigger = decoderConfigDB.ExtTrigger
	configuration.TrgCode1 = decoderConfigDB.TrgCode1
	configuration.TrgCode2 = decoderConfigDB.TrgCode2
	configuration.ReadPMTs = decoderConfigDB.ReadPMTs
	configuration.ReadSiPMs = decoderConfigDB.ReadSiPMs
	configuration.ReadTrigger = decoderConfigDB.ReadTrigger
	configuration.SplitTrg = decoderConfigDB.SplitTrigger
	configuration.NoDB = decoderConfigDB.NoDB
	configuration.Discard = decoderConfigDB.Discard
	configuration.Skip = 0
	configuration.Host = decoderConfigDB.Host
	configuration.User = decoderConfigDB.User
	configuration.Passwd = decoderConfigDB.Passwd
	configuration.DBName = decoderConfigDB.DBName
	configuration.NumWorkers = 1
	configuration.WriteData = decoderConfigDB.WriteData
	configuration.Parallel = false
	configuration.UseBlosc = decoderConfigDB.UseBlosc
	configuration.CompressionLevel = decoderConfigDB.CompressionLevel

	// Parse blosc algorithm
	var compressionAlgorithm decoder.BloscAlgorithm
	compressionAlgorithm.UnmarshalJSON([]byte(`"` + decoderConfigDB.BloscAlgorithm + `"`))
	configuration.BloscAlgorithm = compressionAlgorithm

	// Parse blosc algorithm
	var bitShuffle decoder.BloscShuffle
	bitShuffle.UnmarshalJSON([]byte(`"` + decoderConfigDB.BitShuffle + `"`))
	configuration.BloscShuffle = bitShuffle
	return configuration, nil
}

func setupDecoder(s *server, runNumber int) {
	decoderLogger := DecoderLogger{
		logger: s.logger,
	}
	decoder.SetLogger(decoderLogger)

	configuration, err := parseDecoderConfiguration(s)
	if err != nil {
		message := fmt.Errorf("error parsing decoder configuration: %w", err)
		s.logger.Slog.Error(message.Error())
		return
	}
	decoder.SetConfiguration(configuration)

	dbConn, err := decoder.ConnectToDatabase(configuration.User, configuration.Passwd, configuration.Host, configuration.DBName)
	if err != nil {
		message := fmt.Errorf("error connection to database: %w", err)
		s.logger.Slog.Error(message.Error())
		return
	}
	decoder.LoadDatabase(dbConn, runNumber)
	defer dbConn.Close()
}

func decodeEvent(s *server, eventID int, eventData []byte, writerChs WriterChannels, decoderConfig duck.DecoderConfiguration) {
	start := time.Now()
	header, payload, err := decoder.ReadEvent(eventData)
	//message := fmt.Sprintf("decoder: read event %d", header.EventId[0])
	//s.logger.Slog.Info(message)
	if err != nil {
		writerChs.fileCloser.EvtWithError <- eventID
		message := fmt.Errorf("error reading event header: %w", err)
		s.logger.Slog.Error(message.Error())
		return
	}

	defer func() {
		if r := recover(); r != nil {
			writerChs.fileCloser.EvtWithError <- eventID
			eventID := decoder.EventIdGetNbInRun(header.EventId)
			errMessage := fmt.Errorf("decoder recovered from panic on event %d: %v", eventID, r)
			s.logger.NonStoppingError(errMessage.Error())
			message := fmt.Sprintf("discarding event %d", eventID)
			s.logger.NonStoppingError(message)
			s.metrics.evtDecoderErrorCounter.Inc()
		}
	}()

	event, err := decoder.ReadGDC(payload, header)
	//message = fmt.Sprintf("Decoded event %d: %d, %d", header.EventId[0], len(event.PmtWaveforms), len(event.SipmWaveforms))
	//s.logger.Slog.Debug(message)
	if err != nil {
		writerChs.fileCloser.EvtWithError <- eventID
		message := fmt.Errorf("error reading GDC data: %w", err)
		s.logger.Slog.Error(message.Error())
		s.metrics.evtDecoderErrorCounter.Inc()
		return
	}
	if event.Error {
		writerChs.fileCloser.EvtWithError <- eventID
		message := fmt.Sprintf("discarding event %d, it has errors", event.EventID)
		s.logger.NonStoppingError(message)
		s.metrics.evtDecoderErrorCounter.Inc()
		return
	}

	// Count events per trigger type
	if decoderConfig.SplitTrigger {
		trgType := int(event.TriggerType)
		switch trgType {
		case decoderConfig.TrgCode1:
			s.metrics.evtsPerTriggerTypeCounter.With(prometheus.Labels{"trigger": Trg1.String()}).Inc()
		case decoderConfig.TrgCode2:
			s.metrics.evtsPerTriggerTypeCounter.With(prometheus.Labels{"trigger": Trg2.String()}).Inc()
		default:
			message := fmt.Sprintf("discarding event %d with unknown trigger type %d", event.EventID, trgType)
			s.logger.Slog.Error(message)
			s.metrics.evtDecoderErrorCounter.Inc()
			return
		}
	} else {
		s.metrics.evtsPerTriggerTypeCounter.With(prometheus.Labels{"trigger": Trg0.String()}).Inc()
	}

	if decoderConfig.WriteData {
		if decoderConfig.SplitTrigger {
			if int(event.TriggerType) == decoderConfig.TrgCode1 {
				writerChs.decodedTrg1Ch <- DecodedWriterEvent{
					DuckEventID: eventID,
					decodedEvt:  event,
				}
				pendingJobs := len(writerChs.decodedTrg1Ch)
				s.metrics.evtsWaitingForHdf5WriterCounter.With(prometheus.Labels{"trigger": Trg1.String()}).Set(float64(pendingJobs))
			} else if int(event.TriggerType) == decoderConfig.TrgCode2 {
				writerChs.decodedTrg2Ch <- DecodedWriterEvent{
					DuckEventID: eventID,
					decodedEvt:  event,
				}
				pendingJobs := len(writerChs.decodedTrg2Ch)
				s.metrics.evtsWaitingForHdf5WriterCounter.With(prometheus.Labels{"trigger": Trg2.String()}).Set(float64(pendingJobs))
			}
		} else {
			writerChs.decodedTrg1Ch <- DecodedWriterEvent{
				DuckEventID: eventID,
				decodedEvt:  event,
			}
			pendingJobs := len(writerChs.decodedTrg1Ch)
			s.metrics.evtsWaitingForHdf5WriterCounter.With(prometheus.Labels{"trigger": Trg0.String()}).Set(float64(pendingJobs))
		}
	}
	writerChs.fileCloser.EvtDecoded <- eventID

	duration := time.Since(start)
	//message = fmt.Sprint("decoder: finished event ", event.EventID, " in ", duration.Milliseconds(), "ms")
	//s.logger.Slog.Info(message)
	s.metrics.decodeTimeHistogram.Observe(float64(duration.Milliseconds()))
}

func decodedWriter(s *server, writerCh chan DecodedWriterEvent,
	writtenEventCh chan int, closeFileCh chan bool,
	filenumber int, trigger TriggerType, path string) {
	fname, _ := getDecodedOutputFilename(s, filenumber, int(trigger), path)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, 0755); err != nil {
		message := fmt.Errorf("failed to create output directory %s: %w", path, err)
		s.logger.Slog.Error(message.Error())
		return
	}

	writer, err := decoder.NewWriter(fname)
	retries := 0
	for err != nil && retries < 10 {
		message := fmt.Errorf("error creating writer for file %s: %w", fname, err)
		s.logger.NonStoppingError(message.Error())
		_ = os.Remove(fname)
		messageInfo := fmt.Sprintf("retrying opening file %s", fname)
		s.logger.Slog.Info(messageInfo)
		writer, err = decoder.NewWriter(fname)
		time.Sleep(1 * time.Second)
		retries++
	}
	if err != nil {
		message := fmt.Errorf("error creating writer for file %s after 10 retries: %w", fname, err)
		s.logger.Slog.Error(message.Error())
		return
	}

	loop := true
	for loop {
		select {
		case <-closeFileCh:
			loop = false
		case data := <-writerCh:
			start := time.Now()
			if data.decodedEvt.EventID == 0 {
				fmt.Println("decodedWriter: eventID is 0, skipping...")
				continue
			}
			writer.WriteEvent(&data.decodedEvt)
			writtenEventCh <- data.DuckEventID
			duration := time.Since(start)
			//fmt.Println("decodedWriter: wrote event ", data.decodedEvt.EventID, " in ", duration.Milliseconds(), "ms")
			s.metrics.h5WriteTimeHistogram.Observe(float64(duration.Milliseconds()))
		}
	}
	//fmt.Println("decodedWriter: closing file ", fname)
	err = writer.Close()
	if err != nil {
		message := fmt.Errorf("error closing file %s: %w", fname, err)
		s.logger.Slog.Error(message.Error())
	}
	s.metrics.h5FilesOpenedCounter.With(prometheus.Labels{"trigger": trigger.String()}).Dec()
}

func launchDecoderWorkers(s *server, workers int, jobsCh chan DecoderJob) {
	for i := 0; i < workers; i++ {
		go func() {
			message := fmt.Sprintf("Starting decoder worker %d", i)
			s.logger.Slog.Debug(message)
			for {
				select {
				case job, ok := <-jobsCh:
					if !ok {
						message := fmt.Sprintf("Stopping decoder worker %d", i)
						s.logger.Slog.Debug(message)
						return
					}
					decodeEvent(s, job.DuckEventID, job.Data, job.WritersCh, job.DecoderConfig)
				}
			}
		}()
	}
}
