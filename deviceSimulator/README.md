# Device Simulator

The Device Simulator replays real detector data from binary dump files (.rd format) through the DAQ system, simulating actual detector devices by sending UDP packets to LDCs.

## Overview

Unlike the test device in `deviceRPC/` which generates random data, the Device Simulator reads real data from dump files and replays it with realistic timing, making it ideal for:

- System integration testing with real data patterns
- Performance testing with actual data volumes
- Validating data processing with known good data
- Debugging issues with specific event sequences

## Architecture

```
.rd File → Parser → Replayer → UDP Sender → LDC
                        ↓
                  Rate Control
                        ↓
                    Metrics
```

### Components

- **Parser** (`parser.go`): Reads and parses binary .rd files (GDC → LDC → Equipment hierarchy)
- **Replayer** (`replay.go`): Orchestrates replay with timing control and loop support
- **UDP Sender** (`sender.go`): Fragments data into UDP packets with proper protocol
- **Config** (`config.go`): Loads equipment configuration from database
- **Main** (`main.go`): Provides CLI and ConnectRPC server interfaces

## File Format

The simulator reads DATE format binary files (.rd or .dump):

```
GDC Event (80-byte header)
├── LDC Sub-event 1 (80-byte header)
│   ├── Equipment 1 (28-byte header + data)
│   ├── Equipment 2 (28-byte header + data)
│   └── ...
├── LDC Sub-event 2
│   └── ...
```

All multi-byte values are little-endian uint32.

## UDP Packet Protocol

The simulator sends data using the same protocol as real devices:

```
Packet format: [4-byte seq# (BE)][4-byte eventID (LE)][payload]
End marker: 0xfafafafa (4 bytes, LE)
```

Each event's data is fragmented into packets of configurable size (default 500 bytes), sent sequentially with sequence numbers starting at 0.

## Usage

### CLI Mode

Run standalone for quick testing:

```bash
# Basic usage
./deviceSimulator -id 1 -file /path/to/run.rd -server=false

# With options
./deviceSimulator \
  -id 101 \
  -file /path/to/run_13318.ldc1next.next-100.000.rd \
  -rate 2.0 \
  -loop \
  -max-events 100 \
  -server=false
```

**CLI Flags:**
- `-id`: Equipment ID (required)
- `-file`: Path to .rd file (required for CLI mode)
- `-rate`: Replay rate multiplier (default: 1.0, higher=faster)
- `-loop`: Loop replay continuously (default: false)
- `-max-events`: Maximum events to replay, 0=all (default: 0)
- `-server`: Run as ConnectRPC server (default: true)

### Server Mode

Run as ConnectRPC service (default mode):

```bash
# Start server
./deviceSimulator -id 1 -config config.yaml

# Or specify ports
./deviceSimulator \
  -id 1 \
  -grpc-port 50070 \
  -prometheus-port 12130
```

Control via ConnectRPC (same interface as test device):
- `StartRun()` - Begin replay
- `StopRun()` - Stop replay
- `GetState()` - Get current state
- `GetRunStatistics()` - Get event/packet/error counts
- `PingDevices()` - Health check

### Configuration

#### Database Configuration

The simulator requires two database tables:

1. **equipments** - Standard equipment configuration:
```sql
SELECT id, type, device_ip, host_ip, host_port, ldcID, enabled
FROM equipments WHERE id = ?
```

2. **simulatorParams** - Simulator-specific settings:
```sql
CREATE TABLE simulatorParams (
    equipmentID INT PRIMARY KEY,
    filePath VARCHAR(512),
    replayRate FLOAT DEFAULT 1.0,
    loopMode BOOLEAN DEFAULT false,
    maxEvents INT DEFAULT 0,
    packetSize INT DEFAULT 500
);
```

Example:
```sql
INSERT INTO simulatorParams
VALUES (101, '/data/run_13318.ldc1next.next-100.000.rd', 1.0, true, 0, 500);
```

#### Environment Variables

Database connection (or use `-config` YAML):
```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASS=duck
export DB_NAME=duck
```

### Examples

#### Replay Specific Equipment from File

```bash
# Replay equipment 102 from ldc1next file
./deviceSimulator \
  -id 102 \
  -file next100/run_13318.ldc1next.next-100.000.rd \
  -rate 1.0 \
  -server=false
```

#### Speed Testing (2x Speed)

```bash
./deviceSimulator -id 103 -file run.rd -rate 2.0 -server=false
```

#### Continuous Loop for Long Tests

```bash
./deviceSimulator -id 101 -file run.rd -loop -server=false
```

#### Integration with DAQ System

1. Ensure LDC is running and listening on configured port
2. Equipment configuration in database matches file equipment IDs
3. Start simulator:
```bash
./deviceSimulator -id 102 -config production.yaml
```
4. Control via ConnectRPC or GUI

## File Naming Convention

Files follow the pattern: `run_<runNum>.<ldc>.<experiment>.<fileNum>.rd`

Examples:
- `run_13318.ldc1next.next-100.000.rd` - LDC 1 (NEXT-100)
- `run_13319.ldc2next.next-100.000.rd` - LDC 2 (NEXT-100)
- `run_1.gdc-1.DEV.00000.rd` - GDC 1 (development)

Each file contains data for equipments belonging to that specific LDC.

## Testing

### Test Parser

Verify file parsing:
```bash
cd deviceSimulator
go run test_parser.go parser.go ../next100/run_13318.ldc1next.next-100.000.rd
```

Output shows:
- Total events parsed
- Equipment IDs found
- Data sizes per equipment
- Timestamps

### Build

```bash
go build -o bin/deviceSimulator ./deviceSimulator
```

## Metrics (Prometheus)

Available at `http://localhost:<prometheus-port>/metrics`:

- `simulator_event_count` - Total events replayed
- `simulator_packet_count` - Total UDP packets sent
- `simulator_data_bytes` - Total bytes transmitted
- `simulator_error_count` - Errors encountered
- `simulator_replay_rate` - Current replay rate multiplier

Default ports: 12130 + equipment_id - 1

## Troubleshooting

### Parser Errors

**"invalid magic number"**
- File is corrupted or not a valid .rd file
- Check with eventDump: `./eventDump file.rd`

**"equipment payload extends beyond boundary"**
- File truncated or malformed
- Verify file integrity

### UDP Sending Errors

**"connection refused"**
- LDC not running or wrong port
- Check equipment configuration in database
- Verify LDC is listening: `netstat -an | grep <port>`

**"no route to host"**
- Network connectivity issue
- Check IP configuration in equipments table
- Ping target: `ping <host_ip>`

### No Events for Equipment

**Some events have 0 equipments**
- Normal - control events (start/stop run) don't have equipment data
- Only physics events (type 7) contain equipment data

**Equipment ID not found in file**
- File is from different LDC
- Check file naming: equipment 102 in ldc1next file, not ldc2next
- Verify equipment mapping with test_parser

### Performance Issues

**Replay too slow**
- Increase rate: `-rate 2.0`
- Check CPU usage and network bandwidth
- Monitor with Prometheus metrics

**Events dropped by LDC**
- Rate too high for LDC to process
- Decrease rate: `-rate 0.5`
- Check LDC error logs

## Advanced Usage

### Multi-Equipment Simulation

Run multiple simulators for different equipments:

```bash
# Terminal 1 - Equipment 101
./deviceSimulator -id 101 -file ldc1.rd &

# Terminal 2 - Equipment 102
./deviceSimulator -id 102 -file ldc1.rd &

# Terminal 3 - Equipment 103
./deviceSimulator -id 103 -file ldc1.rd &
```

Each needs unique equipment ID and ports.

### Custom Packet Sizes

Match real hardware:
```sql
UPDATE simulatorParams SET packetSize = 1500 WHERE equipmentID = 101;
```

### Event Limiting for Tests

Replay first 100 events only:
```bash
./deviceSimulator -id 102 -file run.rd -max-events 100 -server=false
```

## Architecture Details

### Replay Timing

The simulator maintains realistic timing by:
1. Extracting timestamps from event headers
2. Calculating time deltas between consecutive events
3. Adjusting delays by rate multiplier
4. Sleeping between sends to maintain cadence

Example: If events are 50ms apart and rate=2.0, simulator sends every 25ms.

### Loop Mode

When enabled:
1. Parser resets to file beginning after last event
2. Timing continues from last event (no gap)
3. Runs indefinitely until stopped

### State Management

States (matches deviceRPC):
- `INITIALIZED` - Ready to start
- `STARTING` - Initializing replay
- `RUNNING` - Actively replaying
- `STOPPING` - Shutting down
- `STOPPED` - Replay complete (internal)

### Thread Safety

- Replayer runs in background goroutine
- State protected by RWMutex
- Stop/Pause use channels for coordination
- Metrics updated periodically (1s interval)

## Comparison: Simulator vs Test Device

| Feature | Device Simulator | Test Device (deviceRPC) |
|---------|------------------|------------------------|
| Data Source | Real .rd files | Random generation |
| Timing | Original or scaled | Fixed rate |
| Loop Support | Yes | N/A (continuous) |
| Packet Protocol | Identical | Identical |
| Control Interface | ConnectRPC + CLI | ConnectRPC only |
| Use Case | Integration testing | Stress testing |
| Equipment per file | Multiple | One |

## Related Tools

- **eventDump** - View .rd file contents
- **deviceRPC** - Test device with random data
- **ldcRPC** - LDC receiver process
- **gdcRPC** - GDC collector process

## Contributing

When modifying the simulator:

1. Update parser if file format changes
2. Maintain UDP protocol compatibility
3. Add tests for new features
4. Update this README
5. Test with real data files

## Support

For issues or questions:
1. Check logs for error messages
2. Verify file format with eventDump
3. Test parser independently with test_parser.go
4. Check network connectivity to LDC
5. Review database configuration
6. Monitor Prometheus metrics
