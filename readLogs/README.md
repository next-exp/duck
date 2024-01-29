# Log Reader Tool

Simple utility to read and display log messages from the Centrifuge WebSocket in real-time 
for debugging and monitoring system activity.

## Overview

The log reader subscribes to the Centrifuge WebSocket "duck" channel and prints all log 
messages to stdout, except METRIC type messages which are filtered out. This provides a 
real-time view of system activity during development and debugging.

## Usage

```bash
./readLogs -config config.yaml
```

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | Required | Path to configuration YAML file |

## Configuration

### YAML Configuration

```yaml
centrifugo:
  host: localhost
  port: 8000
  apiKey: your-api-key
  tokenHMACKey: your-hmac-key
```

The configuration file only needs Centrifuge connection settings.

## Output

The tool prints log messages in real-time:

```
2025-03-06 12:34:56.789 [gdc1next] LOG: Starting data acquisition
2025-03-06 12:34:56.790 [gdc1next] STATE: RUNNING
2025-03-06 12:34:57.123 [ldc1next] LOG: UDP receiver started on port 16001
2025-03-06 12:34:57.456 [ldc1next] LOG: Connected to GDC at 192.168.1.10:50061
2025-03-06 12:35:00.001 [gdc1next] FILE: run_12345.gdc1next.next-100.0.rd (102400 bytes)
2025-03-06 12:35:05.123 [ldc1next] ERROR: Missing datagram detected (expected 100, got 102)
2025-03-06 12:35:10.000 [api] LOG: Run 12345 started successfully
```

### Message Types

| Type | Description | Displayed |
|------|-------------|-----------|
| `LOG` | General log messages | ✓ |
| `ERROR` | Error messages | ✓ |
| `STATE` | State transitions | ✓ |
| `FILE` | File operations | ✓ |
| `SUMMARY` | Run summaries | ✓ |
| `METRICS` | Performance metrics | ✗ (filtered) |

### Output Format

```
{timestamp} [{host}] {type}: {message}
```

Where:
- `timestamp`: Unix timestamp converted to human-readable format
- `host`: Process hostname (gdc1next, ldc1next, api, etc.)
- `type`: Message type (LOG, ERROR, STATE, FILE, SUMMARY)
- `message`: Actual log message content

## Use Cases

### Real-time Monitoring

Monitor system activity during a run:

```bash
./readLogs -config config.yaml

# In another terminal, start a run
curl -X POST http://localhost:1323/duck.v1.DuckAPI/StartRun

# Watch logs appear in real-time
```

### Debugging

Debug issues during development:

```bash
# Start log reader
./readLogs -config config.yaml > debug.log 2>&1 &

# Run tests
task test

# Check debug.log for errors
grep ERROR debug.log
```

### Error Tracking

Monitor for errors in real-time:

```bash
./readLogs -config config.yaml | grep --line-buffered ERROR
```

### State Monitoring

Watch state transitions:

```bash
./readLogs -config config.yaml | grep --line-buffered STATE
```

### Run Summaries

Capture run summaries:

```bash
./readLogs -config config.yaml | grep --line-buffered SUMMARY > run_summaries.log
```

## Output Redirection

### Save to File

```bash
# Save all logs
./readLogs -config config.yaml > logs.txt 2>&1

# Append to file
./readLogs -config config.yaml >> logs.txt 2>&1
```

### Filter with Grep

```bash
# Show only errors
./readLogs -config config.yaml | grep ERROR

# Show errors and state changes
./readLogs -config config.yaml | grep -E "(ERROR|STATE)"

# Exclude specific host
./readLogs -config config.yaml | grep -v "\[api\]"

# Show specific run
./readLogs -config config.yaml | grep "run 12345"
```

### Pipe to Other Tools

```bash
# Count messages by type
./readLogs -config config.yaml | awk '{print $3}' | sort | uniq -c

# Monitor specific equipment
./readLogs -config config.yaml | grep "equipment 101"
```

## Example Session

```bash
$ ./readLogs -config config.yaml
2025-03-06 12:34:56.789 [api] LOG: API server started on port 1323
2025-03-06 12:34:57.123 [ldc1next] LOG: Initializing LDC server
2025-03-06 12:34:57.124 [ldc1next] STATE: INITIALIZED
2025-03-06 12:34:58.456 [gdc1next] LOG: Initializing GDC server
2025-03-06 12:34:58.457 [gdc1next] STATE: INITIALIZED
2025-03-06 12:35:00.001 [api] LOG: Starting run 12345
2025-03-06 12:35:00.050 [gdc1next] STATE: STARTING
2025-03-06 12:35:00.051 [ldc1next] STATE: STARTING
2025-03-06 12:35:00.123 [ldc1next] LOG: UDP receiver started
2025-03-06 12:35:00.456 [gdc1next] LOG: HDF5 writer initialized
2025-03-06 12:35:00.500 [gdc1next] STATE: RUNNING
2025-03-06 12:35:00.501 [ldc1next] STATE: RUNNING
2025-03-06 12:35:00.502 [api] LOG: Run 12345 started successfully
2025-03-06 12:35:05.123 [gdc1next] FILE: run_12345.gdc1next.next-100.0.rd (1048576 bytes)
2025-03-06 12:35:10.456 [gdc1next] FILE: run_12345.gdc1next.next-100.1.rd (1048576 bytes)
^C
```

## Integration with Development Workflow

### Start Full Stack with Logging

```bash
# Terminal 1: Start DAQ system
task e2e:up:dev

# Terminal 2: Watch logs
./readLogs -config docker/e2e/config.yaml

# Terminal 3: Use GUI
open http://localhost:5173
```

### Debug Specific Component

```bash
# Watch logs from specific component
./readLogs -config config.yaml | grep "\[ldc1next\]"
```

## Performance

- **CPU**: <0.1% (just message formatting)
- **Memory**: ~5 MB
- **Network**: ~1 KB/s (depends on log volume)

## Building

```bash
# Build binary
go build -o bin/readLogs ./readLogs

# Or use project build system
task build:readLogs
```

## Troubleshooting

### "Connection refused"

Centrifuge server not running:
```bash
# Check if Centrifuge is running
docker ps | grep centrifugo

# Start if needed
docker-compose up -d centrifugo
```

### "Authentication failed"

Invalid API key or HMAC key:
- Check configuration file
- Verify keys match Centrifuge server configuration

### No Messages Appear

System may be idle:
- Start a run to generate activity
- Check if LDCs/GDCs are running
- Verify Centrifuge subscription in logs

## Files

| File | Purpose |
|------|---------|
| `main.go` | Main log reader implementation |
| `logger.go` | Logger helper functions |

## Dependencies

- Go 1.24+
- `github.com/centrifugal/centrifuge-go` - WebSocket client
- `pkg/centrifugal` - Centrifuge client wrapper
- `pkg/configuration` - Configuration loading

## Related

- `pkg/logger` - DuckLogger that publishes messages
- `rateReader/` - Reads METRICS messages for monitoring
- `api/` - Publishes control messages
- `ldcRPC/`, `gdcRPC/` - Publish operational logs
