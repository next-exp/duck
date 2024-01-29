# Protocol Buffer Definitions (rpc/)

Protocol buffer service definitions for gRPC/ConnectRPC communication between components 
of the NEXT-100 DAQ system.

## Overview

This directory defines two main service interfaces:

1. **RunControl** (`proto/control/control.proto`) - Internal service for LDC/GDC control
2. **DuckAPI** (`proto/api/api.proto`) - Web API for GUI communication

## Services

### RunControl Service (Internal)

**File:** `proto/control/control.proto`

**Purpose:** Communication between API server and data concentrators (LDCs/GDCs).

**Methods:**

| Method | Request | Response | Description |
|--------|---------|----------|-------------|
| `StartRun` | StartRunRequest | StartRunResponse | Start data acquisition |
| `StopRun` | StopRunRequest | StopRunResponse | Stop data acquisition |
| `GetState` | GetStateRequest | GetStateResponse | Get current process state |
| `PingDevices` | PingDevicesRequest | PingDevicesResponse | Health check for devices |
| `GetRunStatistics` | GetRunStatisticsRequest | GetRunStatisticsResponse | Get run metrics |

**Messages:**

```protobuf
message StartRunRequest {
    int32 run_number = 1;
    string host = 2;
    int32 ldc_id = 3;  // For LDCs
    int32 gdc_id = 3;  // For GDCs
}

message StartRunResponse {
    bool success = 1;
    string error = 2;
}

message GetStateResponse {
    string state = 1;
    int32 run_number = 2;
}

message GetRunStatisticsResponse {
    int64 events = 1;
    int64 bytes = 2;
    int64 errors = 3;
    double rate = 4;
    double byte_rate = 5;
}
```

**Usage:**

The API server calls these methods on LDCs and GDCs:

```go
// Start run on LDC
client := control.NewRunControlClient(conn)
resp, err := client.StartRun(ctx, &control.StartRunRequest{
    RunNumber: 12345,
    Host:      "ldc1next",
    LdcId:     1,
})

// Get LDC state
state, err := client.GetState(ctx, &control.GetStateRequest{})
```

### DuckAPI Service (Web API)

**File:** `proto/api/api.proto`

**Purpose:** HTTP/ConnectRPC API for web frontend communication.

**Methods:**

#### Run Control

| Method | Description |
|--------|-------------|
| `StartRun` | Start data acquisition on all processes |
| `StopRun` | Gracefully stop data acquisition |
| `ForceStopRun` | Force immediate stop |
| `GetProcessStates` | Get states of all LDCs/GDCs |
| `RestartServices` | Restart selected services |

#### GDC Management

| Method | Description |
|--------|-------------|
| `CreateGDC` | Add new GDC configuration |
| `GetGDCs` | List all GDCs |
| `GetGDC` | Get single GDC details |
| `UpdateGDC` | Update GDC configuration |
| `DeleteGDC` | Remove GDC |

#### LDC Management

| Method | Description |
|--------|-------------|
| `CreateLDC` | Add new LDC configuration |
| `GetLDCs` | List all LDCs |
| `GetLDC` | Get single LDC details |
| `UpdateLDC` | Update LDC configuration |
| `DeleteLDC` | Remove LDC |

#### Equipment Management

| Method | Description |
|--------|-------------|
| `CreateEquipment` | Register new equipment |
| `GetEquipments` | List all equipment |
| `GetEquipment` | Get single equipment details |
| `UpdateEquipment` | Update equipment configuration |
| `DeleteEquipment` | Remove equipment |

#### Decoder Configuration

| Method | Description |
|--------|-------------|
| `GetDecoderConfiguration` | Get decoder settings |
| `UpdateDecoderConfiguration` | Update decoder settings |

#### Run Information

| Method | Description |
|--------|-------------|
| `GetRunNumber` | Get current run number |
| `CheckDisabled` | Check if system is in maintenance mode |

#### Authentication

| Method | Description |
|--------|-------------|
| `GetToken` | Generate JWT token for Centrifuge |

#### Test Devices

| Method | Description |
|--------|-------------|
| `StartTestDevices` | Start test data generation |
| `StopTestDevices` | Stop test data generation |
| `GetTestDevicesStates` | Get test device states |
| `GetTestDeviceStatistics` | Get test device statistics |
| `UpdateTestDevice` | Update test device parameters |

**Messages:**

```protobuf
message StartRunRequest {
    // Empty - uses database configuration
}

message StartRunResponse {
    int32 run_number = 1;
    repeated ProcessState states = 2;
}

message ProcessState {
    string host = 1;
    string type = 2;  // "LDC" or "GDC"
    int32 id = 3;
    string state = 4;
    string error = 5;
}

message GDC {
    int32 id = 1;
    string host = 2;
    string ip = 3;
    int32 port = 4;
    int32 grpc_port = 5;
    int32 prometheus_port = 6;
    string data_path = 7;
    bool write_output_enable = 8;
    bool decode_enable = 9;
    bool write_data_enable = 10;
    bool enabled = 11;
}

message LDC {
    int32 id = 1;
    string host = 2;
    string ip = 3;
    int32 port = 4;
    int32 grpc_port = 5;
    int32 prometheus_port = 6;
    bool enabled = 7;
}

message Equipment {
    int32 id = 1;
    string type = 2;  // "energy" or "tracking"
    string device_ip = 3;
    string host_ip = 4;
    int32 host_port = 5;
    int32 ldc_id = 6;
    bool enabled = 7;
}
```

## Code Generation

Protocol buffer code is generated using [buf](https://buf.build/).

### Generate Go Code

```bash
# Install buf
go install github.com/bufbuild/buf/cmd/buf@latest

# Generate control service stubs
cd rpc/proto/control
buf generate

# Generate API service stubs (includes TypeScript)
cd ../api
buf generate
```

### Generate All

```bash
# Use project task
task gen:rpc  # Control service
task gen:api  # API service + TypeScript
task gen:all  # Everything
```

### Generated Files

**Control Service (`proto/control/`):**
- `control.pb.go` - Protocol buffer message definitions
- `control_connect.go` - ConnectRPC client/server stubs

**API Service (`proto/api/`):**
- `api.pb.go` - Protocol buffer message definitions
- `api_connect.go` - ConnectRPC Go client/server stubs
- `api_connect.ts` - TypeScript client stubs (for GUI)

## buf Configuration

### buf.yaml

```yaml
version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
```

### buf.gen.yaml (Control)

```yaml
version: v1
plugins:
  - plugin: go
    out: .
    opt:
      - paths=source_relative
  - plugin: connect-go
    out: .
    opt:
      - paths=source_relative
```

### buf.gen.yaml (API)

```yaml
version: v1
plugins:
  - plugin: go
    out: .
    opt:
      - paths=source_relative
  - plugin: connect-go
    out: .
    opt:
      - paths=source_relative
  - plugin: es
    out: ../../gui/src/generated
    opt:
      - target=ts
  - plugin: connect-es
    out: ../../gui/src/generated
    opt:
      - target=ts
```

## Usage Examples

### Go Server (API)

```go
package main

import (
    "connectrpc.com/connect"
    "github.com/jmbenlloch/next_duck/rpc/proto/api"
)

type DuckAPIServer struct {
    // Implementation
}

func (s *DuckAPIServer) StartRun(
    ctx context.Context,
    req *connect.Request[api.StartRunRequest],
) (*connect.Response[api.StartRunResponse], error) {
    // Implementation
    runNumber, states := s.control.StartRun(ctx)
    return connect.NewResponse(&api.StartRunResponse{
        RunNumber: runNumber,
        States:    states,
    }), nil
}

// Register handler
mux := http.NewServeMux()
mux.Handle(api.NewDuckAPIHandler(&DuckAPIServer{}))
```

### Go Client (Control)

```go
package main

import (
    "connectrpc.com/connect"
    "github.com/jmbenlloch/next_duck/rpc/proto/control"
)

func main() {
    client := control.NewRunControlClient(
        http.DefaultClient,
        "http://localhost:50051",
    )
    
    resp, err := client.StartRun(ctx, connect.NewRequest(&control.StartRunRequest{
        RunNumber: 12345,
        Host:      "ldc1next",
        LdcId:     1,
    }))
}
```

### TypeScript Client (GUI)

```typescript
import { createPromiseClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { DuckAPI } from "./generated/api_connect";

const transport = createConnectTransport({
  baseUrl: "http://localhost:1323",
});

const client = createPromiseClient(DuckAPI, transport);

// Start run
const response = await client.startRun({});
console.log(`Started run ${response.runNumber}`);

// Get GDCs
const gdcs = await client.getGDCs({});
gdcs.gdcs.forEach(gdc => {
  console.log(`GDC ${gdc.id}: ${gdc.host}`);
});
```

## Service Architecture

```
┌─────────────────┐
│   Web Frontend  │
│    (Vue.js)     │
└────────┬────────┘
         │ ConnectRPC (HTTP/2)
         │ DuckAPI Service
         │
┌────────▼────────┐
│   API Server    │
│    (api/)       │
└────┬─────┬──────┘
     │     │
     │     └─────────────┐
     │                   │
     │ gRPC              │ gRPC
     │ RunControl        │ RunControl
     │                   │
┌────▼────┐        ┌────▼────┐
│   LDC   │        │   GDC   │
│ (ldc/)  │        │ (gdc/)  │
└─────────┘        └─────────┘
```

## Protocol Design Principles

1. **Backward Compatibility**: New fields can be added without breaking old clients
2. **Clear Naming**: Descriptive field and method names
3. **Error Handling**: Every response includes error field
4. **Pagination**: List methods support pagination (future)
5. **Validation**: Required fields marked clearly

## Testing

```bash
# Lint protobuf files
cd rpc/proto/control && buf lint
cd ../api && buf lint

# Format protobuf files
buf format -w

# Check breaking changes
buf breaking --against '.git#branch=main'
```

## Files

```
rpc/
├── proto/
│   ├── control/
│   │   ├── control.proto        # RunControl service definition
│   │   ├── control.pb.go        # Generated Go messages
│   │   ├── control_connect.go   # Generated ConnectRPC stubs
│   │   ├── buf.yaml             # buf configuration
│   │   └── buf.gen.yaml         # Code generation config
│   └── api/
│       ├── api.proto            # DuckAPI service definition
│       ├── api.pb.go            # Generated Go messages
│       ├── api_connect.go       # Generated ConnectRPC Go stubs
│       ├── buf.yaml             # buf configuration
│       └── buf.gen.yaml         # Code generation config
└── README.md
```

## Dependencies

- Go 1.24+
- `connectrpc.com/connect` - ConnectRPC for Go
- `google.golang.org/protobuf` - Protocol buffers for Go
- `buf` - Protocol buffer tooling
- TypeScript (for GUI):
  - `@connectrpc/connect`
  - `@connectrpc/connect-web`
  - `@bufbuild/protobuf`

## Related

- `api/` - Implements DuckAPI service
- `ldcRPC/` - Implements RunControl service (LDC)
- `gdcRPC/` - Implements RunControl service (GDC)
- `gui/` - Uses DuckAPI TypeScript client
