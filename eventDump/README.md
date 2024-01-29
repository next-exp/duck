# Event Dump Tool

Command-line utility to read and display binary event data files (.rd format) in human-readable 
format for debugging and data inspection.

## Overview

The event dump tool parses DATE-format binary files containing event data from the NEXT-100 
detector. It displays event headers, equipment data, and payload information in a readable format.

## Usage

```bash
./eventDump [options] <filename.raw>

Options:
  -n int
        Maximum number of events to dump (-1 for all) (default 10)
  -h    Headers only mode (skip hex dump of payload data)
  -s    Sort equipment by ID within each event
```

### Examples

```bash
# Dump first 10 events
./eventDump run_12345.ldc1next.next-100.000.rd

# Dump all events (be careful with large files!)
./eventDump -n -1 run_12345.ldc1next.next-100.000.rd

# Dump first 100 events with headers only
./eventDump -n 100 -h run_12345.ldc1next.next-100.000.rd

# Dump events with sorted equipment
./eventDump -s run_12345.ldc1next.next-100.000.rd
```

## Output Format

### Event Header

Each event displays its header information:

```
Event: 1
  Size: 1024 bytes
  Version: 1
  Type: 7 (PHYSICS_EVENT)
  Run: 12345
  Event ID: 1
  LDC ID: 1
  GDC ID: 0
  Timestamp: 2025-03-06 12:34:56.789
```

### Equipment Data

For each equipment in the event:

```
  Equipment: 101 (energy)
    Size: 512 bytes
    Version: 1
    Type: 1
    Event ID: 1
    Equipment ID: 101
    Timestamp: 2025-03-06 12:34:56.789
    
    Payload (512 bytes):
    00000000  00 01 02 03 04 05 06 07  08 09 0a 0b 0c 0d 0e 0f  ................
    00000010  10 11 12 13 14 15 16 17  18 19 1a 1b 1c 1d 1e 1f  ................
    ...
```

### Event Types

| Type | Name | Description |
|------|------|-------------|
| 0 | START_OF_RUN | Run start marker |
| 1 | END_OF_RUN | Run end marker |
| 2 | PHYSICS_EVENT | Physics data |
| 3 | CALIBRATION_EVENT | Calibration data |
| 4 | START_OF_BURST | Burst start (for future use) |
| 5 | END_OF_BURST | Burst end (for future use) |
| 7 | CONTROL_EVENT | Control/monitoring event |

## File Format

The tool reads DATE-format binary files with the following structure:

```
GDC Event (80-byte header)
├── Event Header
│   ├── Size (4 bytes)
│   ├── Version (4 bytes)
│   ├── Type (4 bytes)
│   ├── Run Number (4 bytes)
│   ├── Event ID (8 bytes)
│   ├── LDC ID (4 bytes)
│   ├── GDC ID (4 bytes)
│   └── Timestamp (48 bytes)
├── LDC Sub-event 1
│   ├── LDC Header (80 bytes)
│   ├── Equipment 1
│   │   ├── Equipment Header (28 bytes)
│   │   └── Payload Data
│   ├── Equipment 2
│   └── ...
├── LDC Sub-event 2
└── ...
```

All multi-byte values are little-endian.

## Headers Only Mode

Use `-h` flag to skip the hex dump of payload data:

```bash
./eventDump -h run_12345.ldc1next.next-100.000.rd
```

This is useful for:
- Quick inspection of event structure
- Verifying event ordering
- Checking timestamps
- Identifying missing events

## Sorting Equipment

Use `-s` flag to sort equipment by ID within each event:

```bash
./eventDump -s run_12345.ldc1next.next-100.000.rd
```

This ensures consistent ordering regardless of how equipment data was written.

## Use Cases

### Debug Data Issues

```bash
# Check if a file is corrupted
./eventDump -n 100 run_12345.ldc1next.next-100.000.rd

# Look for missing events
./eventDump -n -1 -h run.rd | grep "Event ID"
```

### Verify Data Format

```bash
# Check event types in a run
./eventDump -n -1 -h run.rd | grep "Type:"

# Verify equipment mapping
./eventDump -n 10 run.rd | grep "Equipment:"
```

### Extract Event Information

```bash
# Get event timestamps
./eventDump -n -1 -h run.rd | grep "Timestamp:"

# Count events
./eventDump -n -1 -h run.rd | grep -c "^Event:"
```

## Performance

- **Small files (<100 MB)**: Near-instantaneous
- **Large files (>1 GB)**: Use `-n` to limit output
- **Very large files**: Use `-h` to skip payload dumps

## Building

```bash
# Build binary
go build -o bin/eventDump ./eventDump

# Or use the project build task
task build:eventDump
```

## Output Redirection

```bash
# Save to file
./eventDump -n 100 run.rd > events.txt

# Search with grep
./eventDump -n -1 -h run.rd | grep "PHYSICS_EVENT"

# Count event types
./eventDump -n -1 -h run.rd | grep "Type:" | sort | uniq -c
```

## Common Patterns

### Check for Missing Event IDs

```bash
./eventDump -n -1 -h run.rd | grep "Event ID" | awk '{print $3}' > event_ids.txt
# Then check for gaps in sequence
```

### Extract Specific Equipment

```bash
./eventDump -n 100 run.rd | grep -A 20 "Equipment: 102"
```

### Verify Run Numbers

```bash
./eventDump -n -1 -h run.rd | grep "Run:" | sort | uniq
```

## Troubleshooting

### "Invalid magic number"

The file is not in DATE format or is corrupted. Verify:
- File extension is `.rd` or `.raw`
- File was created by GDC
- File is not truncated

### "Unexpected EOF"

File is truncated or corrupted. Check:
- File size matches expected
- File transfer completed successfully
- Disk is not full

### Too Much Output

Use `-n` to limit number of events:
```bash
./eventDump -n 10 large_file.rd
```

## Files

| File | Purpose |
|------|---------|
| `main.go` | Event dump tool implementation |

## Dependencies

- Go 1.24+
- Standard library only

## Related Tools

- `deviceSimulator/` - Replays .rd files through DAQ
- `testDataBuilder/` - Extracts LDC-specific data from .rd files
- `gdcRPC/` - Writes .rd files
