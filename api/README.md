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
