# Local Data Concentrator (LDC)

The LDC is the first-stage data concentrator in the NEXT-100 DAQ system. It receives UDP packets 
from front-end electronics (ATCA blades), assembles event fragments, and distributes them to 
Global Data Concentrators (GDCs) via TCP using round-robin load balancing.

## Overview

Each LDC server handles multiple equipment connections (ATCA boards). For NEXT-100:
- **7 LDC servers** for the Energy Plane (PMTs)
- **6 LDC servers** for the Tracking Plane (SiPMs)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         LDC Server                          │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Equipment 1  │  │ Equipment 2  │  │ Equipment N  │     │
│  │   (UDP)      │  │   (UDP)      │  │   (UDP)      │     │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘     │
│         │                 │                 │              │
│         v                 v                 v              │
│  ┌──────────────────────────────────────────────────┐     │
│  │           Ring Buffer (per equipment)             │     │
│  │    Producer/Consumer with reference counting      │     │
│  └───────────────────┬──────────────────────────────┘     │
│                      │                                     │
│                      v                                     │
│  ┌──────────────────────────────────────────────────┐     │
│  │         Event Assembler (equipmentData)           │     │
│  │   - Sequence counter validation                   │     │
│  │   - Missing datagram detection                    │     │
│  │   - Event boundary detection (0xfafafafa)        │     │
│  └───────────────────┬──────────────────────────────┘     │
│                      │                                     │
│                      v                                     │
│  ┌──────────────────────────────────────────────────┐     │
│  │          Sub-Event Builder (ldc.go)               │     │
│  │   Collects data from all equipments per event     │     │
│  └───────────────────┬──────────────────────────────┘     │
│                      │                                     │
│                      v                                     │
│  ┌──────────────────────────────────────────────────┐     │
│  │      Round-Robin GDC Distribution (gdcs.go)       │     │
│  │   currentGDC = (currentGDC + 1) % nGDCs          │     │
│  └───────────────────┬──────────────────────────────┘     │
│                      │                                     │
└──────────────────────┼─────────────────────────────────────┘
                       │
                       v TCP
              ┌────────┴────────┐
              │                 │
         ┌────▼────┐      ┌────▼────┐
         │  GDC 1  │ ...  │  GDC N  │
         └─────────┘      └─────────┘
```

## Key Components

### UDP Packet Reception (`readEquipment.go`)

- **`openEquipmentSocket()`**: Opens UDP socket with `SO_REUSEADDR` for quick rebind after restart
- **`readUDPSocket()`**: Reads UDP packets with deadline and backpressure handling
- **`listenEquipment()`**: Main UDP listening loop, spawns one goroutine per equipment

### Ring Buffer (`ringBuffer.go`)

Lock-free ring buffer for packet buffering:
- **`RingBufferData`**: Holds packet data and position metadata
- **`RingBufferTracking`**: Coordinates producer/consumer with reference counting
- **`TryAcquirePosition()`**: Blocks until buffer position is free (backpressure mechanism)
- **`MarkPositionInUse()`**: Signals data is ready after writing
- **`ReleasePosition()`**: Makes position available for reuse

### Sequence Counter Validation (`equipmentData.go`)

Ensures data integrity:

```go
// Each UDP packet starts with a sequence counter: [0,0,0,1], [0,0,0,2], ...
dataCounter := binary.BigEndian.Uint32(data[0:4])
if dataCounter != uint32(*expectedSequenceCounter) {
    // Missing datagram detected - mark event as corrupted
    *expectedSequenceCounter = dataCounter + 1
    return error
}
*expectedSequenceCounter++
```

Event boundary detection:
- Events end with `0xfafafafa` marker packet
- Sequence counter resets to 0 for each new event
- Corrupted events are skipped during assembly

### Event Assembly (`ldc.go`)

- **`readSubEvents()`**: Main event distribution loop
- **`mountSubEvent()`**: Assembles complete sub-events from all equipment data
- **`sendDataToGDC()`**: Sends assembled data via TCP

### GDC Connection Pool (`gdcs.go`)

- **`gdcGDCConnetionPool()`**: Creates TCP connections to all enabled GDCs
- **`gdcConnection()`**: Establishes single TCP connection with 5-second timeout
- Round-robin distribution ensures even load across GDCs

## Data Format

### Incoming UDP Packet Structure

```
┌────────────────┬────────────────┬─────────────────┐
│ Sequence Counter│   Event Data   │  End Marker?   │
│   (4 bytes)     │   (N bytes)    │  (optional)    │
└────────────────┴────────────────┴─────────────────┘
```

### Outgoing TCP Message Structure

```
┌────────────────────────────────────────────────────┐
│              LDC Header (headers.go)               │
│  - Magic number                                    │
│  - Event ID                                        │
│  - Number of equipments                            │
│  - Data size per equipment                         │
├────────────────────────────────────────────────────┤
│          Equipment 1 Data (from UDP packets)       │
├────────────────────────────────────────────────────┤
│          Equipment 2 Data                          │
├────────────────────────────────────────────────────┤
│                     ...                            │
├────────────────────────────────────────────────────┤
│          Equipment N Data                          │
└────────────────────────────────────────────────────┘
```

## Error Handling

### Missing Datagrams

When sequence counter mismatch detected:
1. Event marked with error flag
2. Sequence counter updated to expected value
3. Event skipped during assembly
4. Error logged and reported to monitoring

### Connection Failures

- TCP connections to GDCs have 5-second timeout
- On connection failure, LDC retries connection
- Round-robin counter still advances to prevent blocking

### Backpressure

Ring buffer prevents memory exhaustion:
- UDP receiver blocks if buffer is full
- Consumer releases positions after processing
- Configurable buffer size based on memory constraints

## Configuration

Configuration loaded from database via API server:
- Equipment IP addresses and ports
- GDC server addresses
- Buffer sizes
- Timeout values

## Monitoring

- **Prometheus metrics**: Packet rates, event rates, buffer utilization
- **Error tracking**: Missing datagrams, connection failures
- **Performance**: Throughput, latency statistics

## Testing

```bash
# Run all LDC tests
task ldcrpc:test:all

# Interactive development shell
task ldcrpc:shell

# Race detector
task ldcrpc:test:race

# Coverage report
task ldcrpc:test:coverage
```

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point, server initialization |
| `readEquipment.go` | UDP socket handling |
| `ringBuffer.go` | Lock-free packet buffer |
| `equipmentData.go` | Sequence validation, event assembly |
| `ldc.go` | Sub-event building, GDC distribution |
| `gdcs.go` | GDC connection pool management |
| `headers.go` | LDC header construction |
| `monitor.go` | Prometheus metrics |

## Performance

- **Throughput**: Designed for >100 MB/s per LDC
- **Latency**: Sub-millisecond event assembly
- **Buffer**: Configurable ring buffer size (default: 1000 packets)
- **Connections**: Handles multiple equipments (typically 4 per LDC)

## Dependencies

- Go 1.24+
- ConnectRPC (for API communication)
- Prometheus client (for metrics)
- pkg/ packages (config, database, logging)
