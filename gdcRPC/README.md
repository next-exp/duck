# Global Data Concentrator (GDC)

The GDC is the second-stage data concentrator in the NEXT-100 DAQ system. It receives sub-events 
from multiple Local Data Concentrators (LDCs) via TCP, assembles complete detector events, and 
writes them to disk in either binary or HDF5 format with optional online decoding.

## Overview

Each GDC server receives events from all LDCs. For NEXT-100:
- **7 GDC servers** for the Energy Plane (PMTs)  
- **7 GDC servers** for the Tracking Plane (SiPMs)

Events are distributed to GDCs via round-robin from LDCs, so each GDC receives ~1/N of total events.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                            GDC Server                           │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │  LDC 1   │  │  LDC 2   │  │  LDC 3   │  │  LDC N   │       │
│  │  (TCP)   │  │  (TCP)   │  │  (TCP)   │  │  (TCP)   │       │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
│       │             │             │             │               │
│       v             v             v             v               │
│  ┌────────────────────────────────────────────────────┐        │
│  │          LDC Data Receiver (ldcData.go)            │        │
│  │  - Magic number validation                          │        │
│  │  - Event ID tracking (increment by nGDCs)          │        │
│  │  - Missing event detection                          │        │
│  └────────────────────┬───────────────────────────────┘        │
│                       │                                         │
│                       v                                         │
│  ┌────────────────────────────────────────────────────┐        │
│  │        Event Builder (processEvents.go)             │        │
│  │  - Collects data from all LDCs per event            │        │
│  │  - Assembles complete detector events               │        │
│  └────┬────────────────┬────────────────┬──────────────┘        │
│       │                │                │                        │
│       v                v                v                        │
│  ┌─────────┐    ┌────────────┐    ┌────────────┐              │
│  │ Binary  │    │  Decoder   │    │  Decoder   │              │
│  │ Writer  │    │  (Trg 1)   │    │  (Trg 2)   │              │
│  │         │    │  Workers   │    │  Workers   │              │
│  └────┬────┘    └─────┬──────┘    └─────┬──────┘              │
│       │               │                  │                      │
│       v               v                  v                      │
│  ┌─────────┐    ┌──────────┐      ┌──────────┐                │
│  │ .rd     │    │ HDF5     │      │ HDF5     │                │
│  │ (raw)   │    │ (Trg 1)  │      │ (Trg 2)  │                │
│  └─────────┘    └──────────┘      └──────────┘                │
│                                                                 │
│  ┌────────────────────────────────────────────────────┐        │
│  │       File Closer Coordinator (fileCloser.go)       │        │
│  │  - Ensures all events written before closing        │        │
│  │  - Tracks: received, decoded, written, errored      │        │
│  └────────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

## Key Components

### LDC Data Reception (`ldcData.go`)

- **`handleConnection()`**: Handles TCP connection from LDC
- **`readLDCData()`**: Reads TCP data in chunks
- **`processData()`**: Assembles complete events with:
  - Magic number validation (lines 99-109)
  - Event ID tracking for detecting missing events (lines 127-137)
  - Expected event ID increments by `nGDCs` due to round-robin distribution (line 151)

### Event Processing (`processEvents.go`)

- **`readData()`**: Main event assembly and distribution
  - Starts binary writer if `writeOutputEnable` enabled
  - Starts HDF5 writers if `decoderConfig.WriteData` enabled
  - Launches decoder workers if `decode` enabled
- **`mountEvent()`**: Assembles complete events from LDC fragments

### Binary Writer (`binaryWriter.go`)

Writes raw event data in DATE-compatible format:

```go
// File naming: run_{run}.{host}.{experiment}.{subrun}.rd
// Example: run_12345.gdc1next.next100.0.rd
```

- Minimal CPU overhead
- Maximum throughput for raw data
- Suitable for offline processing

### HDF5 Writer with Decoder (`decoder.go`)

Online decoding and HDF5 writing:

#### Decoder Setup

```go
// Configuration from context
type DecoderConfig struct {
    WriteData    bool   // Enable HDF5 writing
    NWorkers     int    // Number of decoder workers per trigger type
    BufferSize   int    // Event buffer size
    Calibrations string // Calibration data source
}

// Connects to database and loads calibration data
setupDecoder()
```

#### Event Decoding (`decodeEvent()`)

1. Reads event header
2. Parses GDC payload
3. Routes to appropriate HDF5 writer channel based on trigger type:
   - Type 1 (Calibration): `decodedTrg1Ch`
   - Type 2 (Physics): `decodedTrg2Ch`

#### HDF5 Writing (`decodedWriter()`)

```go
// File naming: run_{run}_{subrun}_ldc{id}_trg{trigger}.waveforms.h5
// Example: run_12345_0_ldc1_trg1.waveforms.h5

writer := decoder.NewWriter(filename)
writer.WriteEvent(event)  // Writes waveforms with Blosc compression
```

#### Worker Pool (`launchDecoderWorkers()`)

- Configurable number of workers per trigger type
- Parallel processing for high throughput
- Separate pools for Type 1 and Type 2 events

### File Closer Coordination (`fileCloser.go`)

Ensures data integrity during file closing:

```go
type FileCloserChannels struct {
    Received chan int    // Events received from LDCs
    Decoded  chan int    // Events decoded successfully
    Written  chan int    // Events written to disk
    Errored  chan int    // Events with errors
}
```

Algorithm:
1. Tracks all events in processing pipeline
2. Waits for all channels to report completion
3. Closes files only when all events accounted for
4. Timeout fallback for error recovery

### I/O Operations (`io.go`)

- **`openFile()`**: Creates output file with proper naming
- **`writeBinaryData()`**: Writes bytes and records Prometheus metrics
- **`openHDF5File()`**: Opens HDF5 file for decoded data

## Data Format

### Incoming LDC Message Structure

```
┌────────────────────────────────────────────────────┐
│                   LDC Header                       │
│  - Magic number (validation)                       │
│  - Event ID                                        │
│  - Number of equipments                            │
│  - Data size per equipment                         │
├────────────────────────────────────────────────────┤
│          Equipment 1 Data                          │
├────────────────────────────────────────────────────┤
│          Equipment 2 Data                          │
├────────────────────────────────────────────────────┤
│                     ...                            │
└────────────────────────────────────────────────────┘
```

### GDC Event Header

```
┌────────────────────────────────────────────────────┐
│                   GDC Header                       │
│  - Magic number                                    │
│  - Event ID (from round-robin distribution)        │
│  - Trigger type (1 or 2)                           │
│  - Number of LDCs                                  │
│  - Total data size                                 │
│  - Timestamps                                      │
├────────────────────────────────────────────────────┤
│          LDC 1 Header + Data                       │
├────────────────────────────────────────────────────┤
│          LDC 2 Header + Data                       │
├────────────────────────────────────────────────────┤
│                     ...                            │
└────────────────────────────────────────────────────┘
```

### Output File Formats

#### Binary (.rd)

Raw event data in DATE-compatible format:
- No decoding overhead
- Maximum write speed
- Processed offline later

#### HDF5 (.h5)

Decoded waveforms with Blosc compression:
- PMT waveforms: Energy Plane signals
- SiPM waveforms: Tracking Plane signals
- Event metadata: Timestamps, trigger info
- ~80% compression ratio

## Error Handling

### Missing Events

Event ID tracking detects missing events:
- Expected ID = previous ID + nGDCs (due to round-robin)
- Missing events logged and reported
- Processing continues with next valid event

### Write Failures

- File closer tracks errored events
- Timeout mechanism prevents hangs
- Partial files cleaned up on error

### Decoder Errors

- Worker pool continues processing other events
- Failed events reported to file closer
- HDF5 file integrity maintained

## Configuration

Loaded from database via API server:

```yaml
writeOutputEnable: true          # Enable binary writing
decoder:
  writeData: true                # Enable HDF5 writing
  decode: true                   # Enable online decoding
  nWorkers: 10                   # Decoder workers per trigger
  bufferSize: 1000               # Event buffer size
```

## Monitoring

- **Prometheus metrics**: 
  - Events received/built/written
  - Write throughput (MB/s)
  - Decoder performance
  - Error rates
- **Real-time status**: Via Centrifuge to GUI
- **Logging**: Structured logs with levels

## Testing

```bash
# Run all GDC tests
task gdcrpc:test:all

# Interactive shell with HDF5 support
task gdcrpc:shell

# Race detector
task gdcrpc:test:race

# Coverage report
task gdcrpc:test:coverage

# Benchmarks
task gdcrpc:test:bench
```

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point, server initialization |
| `ldcData.go` | TCP connection handling from LDCs |
| `processEvents.go` | Event building and distribution |
| `binaryWriter.go` | Raw binary file writer |
| `decoder.go` | Online decoder integration |
| `decodedWriter.go` | HDF5 writer for decoded data |
| `fileCloser.go` | File closing coordination |
| `io.go` | File I/O operations |
| `monitor.go` | Prometheus metrics |

## Performance

- **Throughput**: Up to 125 MB/s per GDC
- **Event Rate**: Designed for >100 Hz
- **Latency**: Sub-second event building
- **Compression**: ~80% with Blosc (HDF5 mode)
- **Workers**: Configurable decoder worker pools

## Dependencies

- Go 1.24+
- HDF5 library (CGO)
- Blosc compression library
- [decoder_go](https://github.com/next-exp/decoder_go) - Event decoder
- ConnectRPC (for API communication)
- Prometheus client (for metrics)
- pkg/ packages (config, database, logging)

## Use Cases

### Binary Mode (Raw Data)

```bash
# Configuration
writeOutputEnable: true
decoder:
  writeData: false
  decode: false
```

Use when:
- Maximum throughput needed
- Offline processing planned
- Limited CPU resources

### HDF5 Mode (Decoded Waveforms)

```bash
# Configuration
writeOutputEnable: false
decoder:
  writeData: true
  decode: true
  nWorkers: 10
```

Use when:
- Real-time monitoring needed
- Quick analysis required
- Storage efficiency important
