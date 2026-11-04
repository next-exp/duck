# API Server

Central HTTP/ConnectRPC server that provides the web API for controlling the NEXT-100 data 
acquisition system. Acts as the main control plane for managing runs, GDCs, LDCs, equipment, 
and system configuration.

## Overview

The API server is the primary interface between the web GUI and the DAQ components. It:

- Serves a REST-like API over HTTP/2 using ConnectRPC
- Manages configuration in MySQL database
- Controls LDCs and GDCs via gRPC
- Broadcasts real-time updates via Centrifuge WebSocket
- Handles authentication and authorization

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        API Server                             │
│                         (port 1323)                           │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐    │
│  │              HTTP/2 + ConnectRPC Handlers             │    │
│  │  - CORS middleware                                    │    │
│  │  - Logging middleware                                 │    │
│  │  - Authentication (token-based)                       │    │
│  └───────────┬──────────────────────────┬───────────────┘    │
│              │                          │                     │
│  ┌───────────▼──────────┐    ┌──────────▼────────────┐       │
│  │   Run Control        │    │   Configuration       │       │
│  │   - StartRun         │    │   - GDC CRUD          │       │
│  │   - StopRun          │    │   - LDC CRUD          │       │
│  │   - ForceStopRun     │    │   - Equipment CRUD    │       │
│  │   - RestartServices  │    │   - Decoder config    │       │
│  └───────────┬──────────┘    └──────────┬────────────┘       │
│              │                          │                     │
│  ┌───────────▼──────────────────────────▼────────────┐       │
│  │              Control Logic (control.go)            │       │
│  │  - Coordinates LDC/GDC startup sequence            │       │
│  │  - Manages state transitions                       │       │
│  │  - Error handling and recovery                     │       │
│  └───────────┬──────────────────────────┬────────────┘       │
│              │                          │                     │
│  ┌───────────▼──────────┐    ┌──────────▼────────────┐       │
│  │   gRPC Clients       │    │   MySQL Database      │       │
│  │   - LDC control      │    │   - Configuration     │       │
│  │   - GDC control      │    │   - Run metadata      │       │
│  └──────────────────────┘    └───────────────────────┘       │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐    │
│  │          Centrifuge WebSocket Publisher               │    │
│  │  - Real-time state updates                           │    │
│  │  - Metrics broadcasting                              │    │
│  │  - Error notifications                               │    │
│  └──────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────┘
```

## API Handlers

### Prometheus metrics

The API serves `GET /metrics` on its HTTP port (1323 by default). Scrape the API
directly; the `/daq/` frontend proxy path is not required. Metrics include:

| Metric | Meaning |
|--------|---------|
| `duck_api_run_active` | 1 when the latest database run has a start time and no stop time; otherwise 0 |
| `duck_api_run_number` | Latest database run number |
| `duck_api_run_state_read_success` | 1 when the database lookup succeeded; 0 when it failed |
| `duck_api_run_transition_state{state="idle|starting|stopping|error"}` | One-hot state of the API's start/stop operation |

On a database lookup failure, `duck_api_run_active` and `duck_api_run_number`
are omitted from that scrape. An active run means run control recorded it as
started; use GDC/LDC event metrics to confirm that data is flowing. For alerts,
check `duck_api_run_state_read_success == 1` and `up == 1` alongside run status.

### Run Control (`apiHandlerControl.go`)

| Method | Description |
|--------|-------------|
| `StartRun` | Starts data acquisition on all enabled GDCs and LDCs |
| `StopRun` | Gracefully stops data acquisition |
| `ForceStopRun` | Forces immediate stop (emergency) |
| `GetProcessStates` | Returns current state of all processes |
| `RestartServices` | Restarts selected services |

**Start Sequence:**
1. Validates configuration
2. Increments run number in database
3. Sends `StartRun` to all GDCs (parallel)
4. Waits for GDCs to initialize
5. Sends `StartRun` to all LDCs (parallel)
6. Waits for LDCs to initialize
7. Broadcasts RUNNING state

**Stop Sequence:**
1. Sends `StopRun` to all LDCs (parallel)
2. Waits for LDCs to stop
3. Sends `StopRun` to all GDCs (parallel)
4. Waits for GDCs to stop
5. Updates run end time in database
6. Broadcasts INITIALIZED state

### GDC Management (`apiHandlerGDC.go`)

| Method | Description |
|--------|-------------|
| `CreateGDC` | Add new GDC to system |
| `GetGDCs` | List all GDCs (with filtering) |
| `GetGDC` | Get single GDC details |
| `UpdateGDC` | Modify GDC configuration |
| `DeleteGDC` | Remove GDC from system |

### LDC Management (`apiHandlerLDC.go`)

| Method | Description |
|--------|-------------|
| `CreateLDC` | Add new LDC to system |
| `GetLDCs` | List all LDCs (with filtering) |
| `GetLDC` | Get single LDC details |
| `UpdateLDC` | Modify LDC configuration |
| `DeleteLDC` | Remove LDC from system |

### Equipment Management (`apiHandlerEquipment.go`)

| Method | Description |
|--------|-------------|
| `CreateEquipment` | Register new equipment (ATCA board) |
| `GetEquipments` | List all equipment (with filtering) |
| `GetEquipment` | Get single equipment details |
| `UpdateEquipment` | Modify equipment configuration |
| `DeleteEquipment` | Remove equipment from system |

### Decoder Configuration (`apiHandlerDecoder.go`)

| Method | Description |
|--------|-------------|
| `GetDecoderConfiguration` | Get current decoder settings |
| `UpdateDecoderConfiguration` | Modify decoder parameters |

### Run Information (`apiHandlerRunInfo.go`)

| Method | Description |
|--------|-------------|
| `GetRunNumber` | Get current run number |
| `CheckDisabled` | Check if system is disabled for maintenance |

### Authentication (`apiHandlerAuth.go`)

| Method | Description |
|--------|-------------|
| `GetToken` | Generate JWT token for Centrifuge WebSocket |

### Test Devices (`apiHandlerTestDevice.go`)

| Method | Description |
|--------|-------------|
| `StartTestDevices` | Start test data generation |
| `StopTestDevices` | Stop test data generation |
| `GetTestDevicesStates` | Get test device states |
| `GetTestDeviceStatistics` | Get test device statistics |
| `UpdateTestDevice` | Update test device parameters |

## Configuration

### Environment Variables

```bash
# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASS=duck
DB_NAME=duck

# Server
API_PORT=1323

# Centrifuge
CENTRIFUGO_HOST=localhost
CENTRIFUGO_PORT=8000
CENTRIFUGO_API_KEY=your-api-key
```

### YAML Configuration

```yaml
database:
  host: localhost
  port: 3306
  user: root
  password: duck
  name: duck

centrifugo:
  host: localhost
  port: 8000
  apiKey: your-api-key
  tokenHMACKey: your-hmac-key

api:
  port: 1323
  tokenDuration: 24h
```

## Usage

### Start Server

```bash
# With config file
./api -config config.yaml

# With custom port
./api -config config.yaml -port 8080
```

### API Endpoints

The server exposes ConnectRPC endpoints:

```bash
# Base URL
http://localhost:1323

# Example: Start a run
curl -X POST http://localhost:1323/duck.v1.DuckAPI/StartRun \
  -H "Content-Type: application/json" \
  -d '{}'

# Example: Get GDCs
curl -X POST http://localhost:1323/duck.v1.DuckAPI/GetGDCs \
  -H "Content-Type: application/json" \
  -d '{}'
```

## State Management

### Process States

- `INITIALIZED` - Ready to start
- `STARTING` - Initializing run
- `RUNNING` - Actively acquiring data
- `STOPPING` - Shutting down
- `PINGING` - Health check in progress

### State Transitions

```
INITIALIZED ──StartRun──> STARTING ──> RUNNING
     ▲                                           │
     │                                           │
     └───────StopRun─────────────────────────────┘
```

## Error Handling

- **Configuration Errors**: Validated before starting run
- **RPC Errors**: Retries with exponential backoff
- **Database Errors**: Logged and reported via Centrifuge
- **Network Errors**: Timeout-based failure detection

## Monitoring

- **Prometheus Metrics**: Request counts, latencies, error rates
- **Structured Logging**: JSON logs with request tracing
- **Centrifuge Broadcasting**: Real-time state and metrics updates

## Testing

```bash
# Run API tests
go test -v ./api/

# Run with coverage
go test -cover ./api/

# Run specific handler tests
go test -v ./api/ -run TestStartRun
```

## Files

| File | Purpose |
|------|---------|
| `server.go` | HTTP server setup and routing |
| `apiHandler.go` | API server struct definition |
| `apiHandlerControl.go` | Run control operations |
| `apiHandlerGDC.go` | GDC CRUD operations |
| `apiHandlerLDC.go` | LDC CRUD operations |
| `apiHandlerEquipment.go` | Equipment CRUD operations |
| `apiHandlerDecoder.go` | Decoder configuration |
| `apiHandlerRunInfo.go` | Run information |
| `apiHandlerAuth.go` | Authentication |
| `apiHandlerTestDevice.go` | Test device management |
| `control.go` | Core run control logic |
| `controlRoutines.go` | Helper routines for control |
| `restart.go` | Service restart logic |
| `rpc.go` | gRPC client setup |
| `rpcClient.go` | RPC client functions |

## Dependencies

- Go 1.24+
- ConnectRPC
- MySQL driver
- Centrifuge client
- pkg/ packages (config, database, logging, metrics)

## Security

- **CORS**: Configured for allowed origins
- **JWT Tokens**: For Centrifuge WebSocket authentication
- **Input Validation**: All inputs sanitized
- **SQL Injection**: Prevented via parameterized queries (sqlc)

## Performance

- **Concurrent Requests**: Handles hundreds of concurrent requests
- **Non-blocking**: Async RPC calls to LDCs/GDCs
- **Connection Pooling**: Database connection reuse
- **Response Time**: <100ms for most operations

## Historical trigger and byte rates

`POST /apiService.DuckAPI/GetRateHistory` returns bounded, counter-derived rates
summed across the GDCs recorded for a single run. Apply the `rateSamples`
migration and deploy the history-enabled `rateReader` before collecting history.
This endpoint is read-only; it does not start acquisition or change TOPI.
The generated Go and TypeScript clients expose `getRateHistory`/`GetRateHistory`.

### Request and browser modes

Whole-run auto mode:

```json
{"run": 16191, "maxPoints": 1000}
```

Fixed-window manual/zoom mode:

```json
{
  "run": 16191,
  "from": "2026-11-04T12:00:00Z",
  "to": "2026-11-04T12:30:00Z",
  "maxPoints": 1000
}
```

Omitting `from` selects the run start (or earliest sample if start is unknown).
Omitting `to` selects the latest recorded sample, limited by the recorded stop
when present. Explicit bounds are clamped to that range, and windows entirely
outside it return no bins. `from` is inclusive and `to` is exclusive; a sample
at `to` can provide the counter endpoint for the preceding interval.

`maxPoints=0` defaults to 1000; values from 1 through 2000 are accepted. The API
selects a bin width from 5, 10, 15, 30 seconds, 1, 2, 5, 10, 15, 30 minutes,
1, 2, 6, 12 hours, 1 day, then wider daily multiples as necessary. Bins are
anchored to the run start. Both clipped edge bins count against the point
budget, so an unaligned zoom does not produce an extra point. The minimum
resolution is five seconds; zooming further cannot recover finer measurements.

The same RPC supports both browser modes; no persistent server-side mode is
needed. Auto mode can poll every five seconds, replacing the series and adapting
the time axis and bin width. Manual mode holds its selected window, cancels
superseded zoom requests and requests the same point budget over the smaller
window. A browser can return to auto by omitting the bounds again. The frontend
plot itself is not implemented by this API change.

### Response and aggregation

The response includes the selected `from`/`to`, `runStart`/`runStop` when known,
`latestSample`, `binWidthSeconds`, `gdcs`, `coverageBasis`, `hasData`,
`maxGapSeconds`, and at most `maxPoints` ordered `bins`.

Each bin contains `start`, `end`, `coverage`, `complete`, `partial`, and optional
`triggerRateHz` and `byteRate` (bytes/second). `partial` means the selected window
clips the bin width; it is independent of missing telemetry. Rate fields are
present only when all the bin's GDC-time is covered. A true zero rate remains
present as zero; an incomplete bin has absent rate fields. Plot these as gaps,
not as zero and not as a line bridged through the gap.

For each GDC, the API differences consecutive unsigned event/byte counters before
converting their deltas to floating point. It assumes a constant rate between
each pair of measurements and distributes the deltas to overlapping bins in
proportion to their time overlap. It then sums the GDC contributions and divides
by the actual bin duration. This produces time-weighted interval averages;
current sliding-window rate values and whole-run averages are not used.

Zoom queries include one predecessor and one successor per GDC so window-edge
intervals can be calculated without snapping the requested window to samples.
Counter decreases and `counter_reset=true` invalidate the interval ending at the
reset sample; the next interval may recover normally. Intervals exceeding
`maxGapSeconds` are unknown, not interpolated. The default is 15 seconds for the
five-second writer; an explicit value from 1 to 3600 seconds supports different
collector periods or stricter gap policies.

`coverage` is covered GDC-seconds divided by bin duration times roster size.
For now the roster consists of distinct GDCs recorded anywhere in the run,
including outside a manual window. `coverageBasis="observed_run_history"`
explicitly identifies this limitation: a GDC that never reported is unknown.
The endpoint does not substitute today's enabled-GDC configuration for a
historical configuration snapshot. Completed bins are complete relative to the
recorded roster, not a guarantee that every configured GDC reported. Persisting
an expected per-run roster can strengthen this in a later migration.

### Database access and limits

Queries use one repeatable-read, read-only transaction, so the roster, bounds and
counter streams share a snapshot while acquisition continues. The roster query
uses the `(run,gdc,sample_time)` primary key. Per-GDC window reads and the scalar
predecessor/successor lookups use the same key. The secondary `(run,sample_time)`
index remains available for other single-run time queries.

Rows are streamed into at most 2000 in-memory bin accumulators, rather than
loading all raw samples into a Go slice or sending them to the browser. Memory
therefore does not grow with raw sample count. Work still scales with the
selected window's raw samples: indexing avoids other runs, but it does not make
a full long-run scan constant-time. A ten-second query deadline and a roster
limit of 256 bound each request. Measured demand can justify caching or stored
rollups without changing this contract. A corrupt/implausible span exceeding
100 years is rejected before bin arithmetic.

An existing run with no history returns `hasData=false`, no bins and run metadata
when available. One sample can establish history bounds but cannot establish a
rate. Unknown runs return `not_found`; invalid arguments return
`invalid_argument`; a missing history table returns `failed_precondition`.
Canceling a request cancels its database work.

### History API validation

Run `go test -race ./api -run '^TestRateHistory' -timeout 60s`.
For the real MySQL/ConnectRPC test, set `RATE_HISTORY_TEST_DSN` to a disposable
schema and run the same command. The test creates and drops `runs` and
`rateSamples`; never point it at production. It verifies zoom boundary samples,
aggregate counter rates, gaps, reset recovery, run isolation, native resolution,
point limits and an indexed query plan.

`go test ./api -run '^$' -bench '^BenchmarkRateHistoryRebin$' -benchtime=1x -benchmem`
measures only the Go aggregation step for seven GDCs with 120000 intervals each;
it is not a database or end-to-end latency measurement.
