# Device Simulator Testing Guide

## Overview

This directory contains a mock LDC tool for testing the device simulator locally without needing the full e2e environment.

## Prerequisites

1. Build the mock LDC:
```bash
cd /home/jmbenlloch/next/duck/deviceSimulator
go build -o mock_ldc mock_ldc.go
```

2. Ensure you have an RD file to test with (e.g., `../next100/run_14816.ldc2next.next-100.042.rd`)

## Running the Test

### Terminal 1: Start Mock LDC

```bash
cd /home/jmbenlloch/next/duck/deviceSimulator
./mock_ldc -listen 127.0.0.1:6000 -stats-interval 5
```

This will:
- Listen on UDP port 6000
- Print statistics every 5 seconds
- Validate sequence counters on all received packets
- Count events (marked by 0xfafafafa)

### Terminal 2: Run Simulator (with Database)

If you have the database running (e.g., via Docker):

```bash
cd /home/jmbenlloch/next/duck/deviceSimulator

# Make sure database is accessible
export DB_HOST="localhost"
export DB_PORT="3306"
export DB_USER="root"
export DB_PASS="duck"
export DB_NAME="duck"

# First insert test equipment into database:
# INSERT INTO equipments (id, type, device_ip, host_ip, host_port, enabled, ldcID)
# VALUES (999, 22, '0.0.0.0', '127.0.0.1', 6000, true, 1);
#
# INSERT INTO simulatorParams (equipmentID, filePath, rateHz, loopMode, maxEvents, packetSize)
# VALUES (999, '/path/to/file.rd', 10.0, false, 10, 7992);

# Start simulator in server mode
./deviceSimulator -id 999 -config config.yml

# In another terminal, trigger the run:
curl -X POST http://localhost:50000/daq.RunControl/StartRun \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Terminal 2 Alternative: Simpler Test (No Database)

For quick testing without database, you can modify the `runCLI` function or use this approach:

```bash
# Start the mock LDC in terminal 1 as above

# In terminal 2, use the go run command directly:
cd /home/jmbenlloch/next/duck/deviceSimulator

# This requires modifying config.go to provide a default equipment config
# Or manually test with the full e2e environment
```

## Expected Output

### Mock LDC Output (Success)

```
2025/11/03 10:00:00 Mock LDC listening on 127.0.0.1:6000
2025/11/03 10:00:01 [Event 1] Complete: 61 packets, 487432 bytes (from 127.0.0.1:xxxxx)
2025/11/03 10:00:02 [Event 2] Complete: 61 packets, 487432 bytes (from 127.0.0.1:xxxxx)
...
2025/11/03 10:00:05 [Progress] Received 200 packets, 3 events, 0 sequence errors

========================================
Mock LDC Statistics
========================================
Events received:      10
Packets received:     610
Bytes received:       4874320
Sequence errors:      0
Duration:             10.50 seconds
Event rate:           0.95 events/sec

✓ All packets received with correct sequence counters
========================================
```

### Mock LDC Output (Failure - Sequence Errors)

```
2025/11/03 10:00:01 [ERROR] Sequence mismatch! Expected 33, got 52 (packet size: 7992 bytes)
...
========================================
Mock LDC Statistics
========================================
Events received:      5
Packets received:     300
Bytes received:       2400000
Sequence errors:      15
...
⚠️  WARNING: 15 sequence errors detected!
========================================
```

## What the Fix Addresses

The fix in `sender.go:182-197` ensures that when the simulator stops:

1. A final end-of-event marker (0xfafafafa) is sent to ensure the LDC resets its sequence counter
2. A 100ms delay allows in-flight packets to be delivered
3. The UDP connection is properly closed

This prevents the sequence counter mismatch errors that occurred when stopping and restarting runs.

## Verification

To verify the fix works:

1. Start mock LDC
2. Start/stop the simulator multiple times
3. Check that mock LDC reports 0 sequence errors each time
4. Verify events are complete (no incomplete events warning)

## Notes

- Equipment ID 216 in the test file has approximately 61 packets per event
- Each packet is 7992 bytes (fragment size for NEXT-100)
- The test file contains data for 27 equipment IDs
- Sequence counters must be consecutive (0, 1, 2, ..., N) within each event
