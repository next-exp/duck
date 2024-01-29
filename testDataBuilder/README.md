# Test Data Builder Tool

Utility to extract and rebuild test data files from GDC raw data files, isolating data for 
a specific LDC for testing and debugging purposes.

## Overview

The test data builder reads a GDC raw data file (`.rd` format), extracts events belonging 
to a specific LDC, sorts them by event ID, detects missing events, and writes the filtered 
data to a new file. This creates isolated LDC test files from combined GDC output.

## Usage

```bash
./testDataBuilder -data /path/to/data -ldc <ldc_id>
```

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-data` | Required | Directory containing .rd files |
| `-ldc` | Required | LDC ID to extract (1, 2, 3, etc.) |

### Examples

```bash
# Extract LDC 1 data
./testDataBuilder -data /data/next100/runs -ldc 1

# Extract LDC 2 data
./testDataBuilder -data /data/next100/runs -ldc 2

# Extract from specific directory
./testDataBuilder -data ./test_data -ldc 3
```

## How It Works

### Input Processing

1. **Scan Directory**: Finds all `.rd` files in the data directory
2. **Read GDC Files**: Parses DATE-format binary files
3. **Filter by LDC**: Extracts events containing data from the specified LDC
4. **Sort Events**: Orders events by event ID
5. **Detect Gaps**: Identifies missing event IDs in sequence
6. **Write Output**: Creates new file `ldc<id>.raw`

### Data Flow

```
┌────────────────────────────────────────────────────────┐
│                  GDC Raw Data File (.rd)                │
│                                                        │
│  Event 1:                                              │
│    LDC 1: [Equipment 101, 102, 103, 104]              │
│    LDC 2: [Equipment 201, 202, 203, 204]              │
│    LDC 3: [Equipment 301, 302, 303, 304]              │
│                                                        │
│  Event 2:                                              │
│    LDC 1: [Equipment 101, 102, 103, 104]              │
│    LDC 2: [Equipment 201, 202, 203, 204]              │
│    LDC 3: [Equipment 301, 302, 303, 304]              │
│  ...                                                   │
└───────────────────┬────────────────────────────────────┘
                    │
                    │ Extract LDC 1
                    │
                    v
┌────────────────────────────────────────────────────────┐
│              LDC 1 Test File (ldc1.raw)                │
│                                                        │
│  Event 1: [Equipment 101, 102, 103, 104]              │
│  Event 2: [Equipment 101, 102, 103, 104]              │
│  Event 3: [Equipment 101, 102, 103, 104]              │
│  ...                                                   │
└────────────────────────────────────────────────────────┘
```

## Output

### File Creation

Creates file in current directory: `ldc<id>.raw`

Example:
```bash
$ ./testDataBuilder -data /data/run_12345 -ldc 1
Processing /data/run_12345/run_12345.gdc1next.next-100.0.rd...
Processing /data/run_12345/run_12345.gdc1next.next-100.1.rd...
Processing /data/run_12345/run_12345.gdc1next.next-100.2.rd...

Summary:
  Total events: 10000
  Events with LDC 1: 10000
  Missing event IDs: 2, 5, 8 (3 total)
  
Output: ldc1.raw (52.3 MB)
```

### Missing Event Detection

The tool reports gaps in event ID sequence:

```
Missing event IDs: 2, 5, 8, 15, 23
Total missing: 5
```

This helps identify:
- Network packet loss
- LDC processing errors
- Event filtering issues

## Use Cases

### Create Test Data for LDC Development

```bash
# Extract LDC 1 data from production run
./testDataBuilder -data /data/production/run_12345 -ldc 1

# Use extracted file for testing
./deviceSimulator -id 101 -file ldc1.raw -server=false
```

### Debug LDC-Specific Issues

```bash
# Extract problematic LDC data
./testDataBuilder -data /data/run_with_issues -ldc 2

# Analyze with eventDump
./eventDump -n 100 ldc2.raw
```

### Validate Event Sequencing

```bash
# Check for missing events
./testDataBuilder -data /data/run_12345 -ldc 1 2>&1 | grep "Missing"
```

### Create Minimal Test Sets

```bash
# Extract small subset for quick testing
# (Manually truncate GDC file first)
dd if=run.gdc1next.rd of=small.gdc1next.rd bs=1M count=10
./testDataBuilder -data . -ldc 1
```

## File Format

### Input: GDC Raw Data

DATE-format binary with:
- 80-byte GDC event header
- 80-byte LDC sub-event headers
- 28-byte equipment headers
- Equipment payload data

### Output: LDC Test Data

Same DATE format but containing only:
- Single LDC's sub-events
- Sorted by event ID
- Equipment data intact

## Performance

### Large Files

- **1 GB GDC file**: ~30 seconds to process
- **10,000 events**: ~5 seconds
- **Memory usage**: ~100 MB (streaming parser)

### Optimization

The tool uses streaming I/O:
- Reads events one at a time
- Only keeps LDC data in memory
- Writes output incrementally

## Example Workflow

### Prepare Test Data

```bash
# 1. Extract LDC data from production run
./testDataBuilder -data /data/production/run_12345 -ldc 1

# 2. Verify extracted data
./eventDump -n 10 ldc1.raw

# 3. Use with device simulator
./deviceSimulator -id 101 -file ldc1.raw -loop

# 4. Test LDC processing
./ldcRPC -config test_config.yaml
```

### Analyze Event Gaps

```bash
# Extract and check for gaps
./testDataBuilder -data /data/run_12345 -ldc 1 2>&1 | tee extract.log

# Count missing events
grep "Total missing" extract.log

# Find which events are missing
grep "Missing event IDs" extract.log
```

## Integration with Testing

### Unit Test Data Generation

```bash
# Create small test file
dd if=run.gdc1.rd of=test.gdc1.rd bs=1M count=1
./testDataBuilder -data . -ldc 1

# Use in tests
cp ldc1.raw testdata/sample_events.raw
```

### Performance Testing

```bash
# Extract large dataset
./testDataBuilder -data /data/large_run -ldc 1

# Benchmark LDC processing
time ./ldcRPC -config benchmark.yaml < ldc1.raw
```

## Troubleshooting

### "No .rd files found"

Check directory path:
```bash
ls -la /data/next100/runs/*.rd
```

### "LDC ID not found in file"

Verify LDC IDs present:
```bash
./eventDump -n 100 file.rd | grep "LDC ID"
```

### Large Output Files

Output file size equals LDC data portion of input:
- 1 GB GDC file with 7 LDCs → ~140 MB per LDC file
- This is expected behavior

## Files

| File | Purpose |
|------|---------|
| `main.go` | Main data extraction logic |
| `io.go` | Binary file writing utilities |
| `dateHeaders.go` | DATE format header structures |

## Dependencies

- Go 1.24+
- Standard library only

## Related Tools

- `eventDump/` - View extracted data
- `deviceSimulator/` - Replay extracted data
- `ldcRPC/` - Process extracted data
- `gdcRPC/` - Creates original GDC files

## Advanced Usage

### Extract Multiple LDCs

```bash
# Extract all LDCs from a run
for i in {1..7}; do
    ./testDataBuilder -data /data/run_12345 -ldc $i
done

# Result: ldc1.raw, ldc2.raw, ..., ldc7.raw
```

### Compare LDC Outputs

```bash
# Extract two LDCs
./testDataBuilder -data /data/run -ldc 1
./testDataBuilder -data /data/run -ldc 2

# Compare event counts
./eventDump -n -1 -h ldc1.raw | grep -c "^Event:"
./eventDump -n -1 -h ldc2.raw | grep -c "^Event:"
```

### Validate Round-Robin Distribution

```bash
# Extract LDC 1 data from multiple GDC files
# (LDC 1 data distributed across GDCs via round-robin)
./testDataBuilder -data /data/run_12345 -ldc 1

# Verify event IDs are consecutive (with gaps = nGDCs - 1)
./eventDump -n -1 -h ldc1.raw | grep "Event ID" | head -20
```
