//go:build nohdf5

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jmbenlloch/next_duck/gdcRPC/testhelpers"
	duck "github.com/jmbenlloch/next_duck/pkg"
)

// Helper function to create a test server for processEvents tests
func createTestServerForProcessEvents(runNumber int, gdcID int, nGDCs int) *server {
	gdcConfig := testhelpers.NewTestGDCConfiguration(gdcID, "gdc1")
	decoderConfig := duck.DecoderConfiguration{WriteData: false}
	ctx := testhelpers.NewMockContextBuilder().
		WithRunNumber(runNumber).
		WithGDCConfiguration(&gdcConfig).
		WithWriteOutput(true).
		WithDecode(false).
		WithNGDCs(nGDCs).
		WithDecoderConfig(decoderConfig).
		Build()

	return &server{
		ctx:            ctx,
		cancelCtx:      func() {},
		configFilename: "test_config.yml",
		serverName:     "test_server",
		state:          duck.INITIALIZED,
		logger:         duck.NewDuckLogger("", nil, 0),
		metrics:        NewMetricsRegistry(),
	}
}

// TestCountEnabledLDCs_AllEnabled tests counting when all LDCs are enabled with enabled equipments
func TestCountEnabledLDCs_AllEnabled(t *testing.T) {
	ldc1 := testhelpers.NewTestLDCConfiguration(1, "ldc1", 3)
	ldc2 := testhelpers.NewTestLDCConfiguration(2, "ldc2", 2)
	ldc3 := testhelpers.NewTestLDCConfiguration(3, "ldc3", 4)
	ldcs := []duck.LDCConfiguration{ldc1, ldc2, ldc3}

	count := countEnabledLDCs(ldcs)
	assert.Equal(t, 3, count, "All LDCs with enabled equipments should be counted")
}

// TestCountEnabledLDCs_SomeDisabled tests counting when some LDCs are disabled
func TestCountEnabledLDCs_SomeDisabled(t *testing.T) {
	ldc1 := testhelpers.NewTestLDCConfiguration(1, "ldc1", 3)
	ldc2 := testhelpers.NewTestLDCConfiguration(2, "ldc2", 2)
	ldc2.Enabled = false
	ldc3 := testhelpers.NewTestLDCConfiguration(3, "ldc3", 4)
	ldcs := []duck.LDCConfiguration{ldc1, ldc2, ldc3}

	count := countEnabledLDCs(ldcs)
	assert.Equal(t, 2, count, "Only enabled LDCs should be counted")
}

// TestCountEnabledLDCs_AllEquipmentsDisabled tests that LDCs with no enabled equipments are not counted
func TestCountEnabledLDCs_AllEquipmentsDisabled(t *testing.T) {
	ldc1 := testhelpers.NewTestLDCConfiguration(1, "ldc1", 3)
	ldc2 := testhelpers.NewTestLDCConfiguration(2, "ldc2", 2)
	ldc3 := testhelpers.NewTestLDCConfiguration(3, "ldc3", 4)

	// Disable all equipments in ldc2
	for i := range ldc2.Equipments {
		ldc2.Equipments[i].Enabled = false
	}

	ldcs := []duck.LDCConfiguration{ldc1, ldc2, ldc3}
	count := countEnabledLDCs(ldcs)
	assert.Equal(t, 2, count, "LDCs with no enabled equipments should not be counted")
}

// TestCountEnabledLDCs_NoEnabledEquipments tests counting when no LDCs have enabled equipments
func TestCountEnabledLDCs_NoEnabledEquipments(t *testing.T) {
	ldc1 := testhelpers.NewTestLDCConfiguration(1, "ldc1", 3)
	ldc2 := testhelpers.NewTestLDCConfiguration(2, "ldc2", 2)

	// Disable all equipments
	for _, ldc := range []duck.LDCConfiguration{ldc1, ldc2} {
		for i := range ldc.Equipments {
			ldc.Equipments[i].Enabled = false
		}
	}

	ldcs := []duck.LDCConfiguration{ldc1, ldc2}
	count := countEnabledLDCs(ldcs)
	assert.Equal(t, 0, count, "No LDCs should be counted when all equipments are disabled")
}

// TestCountEnabledLDCs_EmptyList tests counting with empty LDC list
func TestCountEnabledLDCs_EmptyList(t *testing.T) {
	ldcs := []duck.LDCConfiguration{}
	count := countEnabledLDCs(ldcs)
	assert.Equal(t, 0, count, "Empty list should return 0")
}

// TestCountEnabledLDCs_PartiallyEnabledEquipments tests LDC with some equipments enabled
func TestCountEnabledLDCs_PartiallyEnabledEquipments(t *testing.T) {
	ldc1 := testhelpers.NewTestLDCConfiguration(1, "ldc1", 5)
	ldc2 := testhelpers.NewTestLDCConfiguration(2, "ldc2", 3)

	// Disable some equipments in ldc2
	ldc2.Equipments[0].Enabled = false
	ldc2.Equipments[2].Enabled = false

	ldcs := []duck.LDCConfiguration{ldc1, ldc2}
	count := countEnabledLDCs(ldcs)
	assert.Equal(t, 2, count, "LDCs with any enabled equipment should be counted")
}

// TestMountEvent_SingleLDC_AssemblyComplete tests event assembly with single LDC
func TestMountEvent_SingleLDC_AssemblyComplete(t *testing.T) {
	s := createTestServerForProcessEvents(123, 1, 1)

	eventData := make(map[int]map[int][]byte)
	ldcData := LDCData{
		EventID: 100,
		ldcID:   1,
		Data:    testhelpers.CreateLDCEventData(100, 1, 200),
	}

	writerChs := WriterChannels{
		binaryEvtCh: make(chan []byte, 10),
		fileCloser: FileCloserChannels{
			EvtReceived: make(chan int, 10),
		},
	}
	bytesInFile := 0
	metrics := make(chan int, 10)
	decoderConfig := duck.DecoderConfiguration{WriteData: false}
	decoderJobsCh := make(chan DecoderJob, 10)

	mountEvent(s, &eventData, &ldcData, writerChs, &bytesInFile, 1, 1, metrics, decoderConfig, decoderJobsCh)

	select {
	case <-writerChs.binaryEvtCh:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for GDC data")
	}
}

// TestMountEvent_MultipleLDCs_AssemblyComplete tests event assembly with multiple LDCs
func TestMountEvent_MultipleLDCs_AssemblyComplete(t *testing.T) {
	s := createTestServerForProcessEvents(123, 1, 3)

	eventData := make(map[int]map[int][]byte)
	eventData[200] = make(map[int][]byte)
	eventData[200][1] = testhelpers.CreateLDCEventData(200, 1, 200)
	eventData[200][2] = testhelpers.CreateLDCEventData(200, 2, 200)

	ldcData := LDCData{
		EventID: 200,
		ldcID:   3,
		Data:    testhelpers.CreateLDCEventData(200, 3, 200),
	}

	writerChs := WriterChannels{
		binaryEvtCh: make(chan []byte, 10),
		fileCloser: FileCloserChannels{
			EvtReceived: make(chan int, 10),
		},
	}
	bytesInFile := 0
	metrics := make(chan int, 10)
	decoderConfig := duck.DecoderConfiguration{WriteData: false}
	decoderJobsCh := make(chan DecoderJob, 10)

	mountEvent(s, &eventData, &ldcData, writerChs, &bytesInFile, 1, 3, metrics, decoderConfig, decoderJobsCh)

	select {
	case <-writerChs.binaryEvtCh:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for GDC data")
	}
}

// TestWriterChannels_Structure tests WriterChannels structure
func TestWriterChannels_Structure(t *testing.T) {
	writerChs := WriterChannels{
		binaryEvtCh:   make(chan []byte, 5),
		decodedTrg1Ch: make(chan DecodedWriterEvent, 5),
		decodedTrg2Ch: make(chan DecodedWriterEvent, 5),
		fileCloser: FileCloserChannels{
			EvtReceived:  make(chan int, 5),
			EvtDecoded:   make(chan int, 5),
			EvtWritten:   make(chan int, 5),
			EvtWithError: make(chan int, 5),
			DataEnd:      make(chan bool, 5),
			CloseFiles:   make(chan bool, 5),
		},
	}

	assert.NotNil(t, writerChs.binaryEvtCh)
	assert.NotNil(t, writerChs.decodedTrg1Ch)
	assert.NotNil(t, writerChs.decodedTrg2Ch)
}

// TestLDCData_Structure tests LDCData structure
func TestLDCData_Structure(t *testing.T) {
	ldcData := LDCData{
		EventID: 123,
		ldcID:   2,
		Data:    []byte{0x01, 0x02, 0x03},
	}

	assert.Equal(t, 123, ldcData.EventID)
	assert.Equal(t, 2, ldcData.ldcID)
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, ldcData.Data)
}

// TestDecoderJob_Structure tests DecoderJob structure
func TestDecoderJob_Structure(t *testing.T) {
	job := DecoderJob{
		DuckEventID: 456,
		Data:        []byte{0xAA, 0xBB},
		WritersCh: WriterChannels{
			binaryEvtCh: make(chan []byte, 1),
		},
		DecoderConfig: duck.DecoderConfiguration{
			WriteData: true,
		},
	}

	assert.Equal(t, 456, job.DuckEventID)
	assert.Equal(t, []byte{0xAA, 0xBB}, job.Data)
	assert.True(t, job.DecoderConfig.WriteData)
}
