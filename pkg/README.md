# Shared Packages (pkg/)

Shared Go packages used across all components of the NEXT-100 DAQ system. These packages 
provide common functionality for configuration, logging, metrics, database access, and 
real-time messaging.

## Overview

The `pkg/` directory contains reusable packages that are imported by:
- API server (`api/`)
- Local Data Concentrators (`ldcRPC/`)
- Global Data Concentrators (`gdcRPC/`)
- Monitoring tools (`rateReader/`, `readLogs/`)
- Test utilities

## Packages

### Configuration (`configuration.go`)

Manages configuration loading from YAML files and database.

**Key Functions:**
- `ReadConfig()` - Load configuration from YAML file
- `ReadDBConfiguration()` - Load system configuration from database
- `GetDBConnection()` - Create database connection with retry logic

**Configuration Structure:**
```go
type Config struct {
    DBHost     string `yaml:"dbHost"`
    DBPort     int    `yaml:"dbPort"`
    DBUser     string `yaml:"dbUser"`
    DBPass     string `yaml:"dbPass"`
    DBName     string `yaml:"dbName"`
    Centrifugo CentrifugoConfig `yaml:"centrifugo"`
}

type CentrifugoConfig struct {
    Host       string `yaml:"host"`
    Port       int    `yaml:"port"`
    APIKey     string `yaml:"apiKey"`
    TokenHMACKey string `yaml:"tokenHMACKey"`
}
```

**Usage:**
```go
config, err := configuration.ReadConfig("config.yaml")
if err != nil {
    log.Fatal(err)
}
db, err := configuration.GetDBConnection(config)
```

### Logger (`logger.go`)

Custom logger that publishes log messages to Centrifuge WebSocket for real-time monitoring.

**DuckLogger:**
```go
type DuckLogger struct {
    logger     *log.Logger
    centrifuge *CentrifugeClient
    host       string
    runNumber  int
}

// Custom log levels
const (
    LOG      = "LOG"
    ERROR    = "ERROR"
    METRICS  = "METRICS"
    STATE    = "STATE"
    FILE     = "FILE"
    SUMMARY  = "SUMMARY"
)
```

**Key Methods:**
- `Log(message)` - Regular log message
- `Error(message)` - Error message
- `Metrics(metrics)` - Publish performance metrics
- `State(state)` - Publish state changes
- `File(filename, size)` - Log file operations
- `Summary(stats)` - Publish run summary

**Usage:**
```go
logger := logger.NewDuckLogger(centrifugeClient, "gdc1")
logger.Log("Starting data acquisition")
logger.Metrics(metrics)
logger.State(pkg.RUNNING)
```

### Metrics (`metrics.go`)

Tracks and calculates performance metrics for monitoring.

**Metrics Structure:**
```go
type Metrics struct {
    Events      int64   // Total event count
    Bytes       int64   // Total bytes processed
    Errors      int64   // Error count
    AvgRate     float64 // Average event rate (Hz)
    CurrentRate float64 // Current event rate (Hz)
    AvgByteRate float64 // Average byte rate (MB/s)
    CurrentByteRate float64 // Current byte rate (MB/s)
}

type MetricsCounter struct {
    mu           sync.RWMutex
    metrics      Metrics
    prevEvents   int64
    prevBytes    int64
    prevTime     time.Time
    period       time.Duration
}
```

**Key Methods:**
- `IncrementEvents(count)` - Increment event counter
- `IncrementBytes(count)` - Increment byte counter
- `IncrementErrors(count)` - Increment error counter
- `GetMetrics()` - Get current metrics with calculated rates

**Usage:**
```go
counter := metrics.NewMetricsCounter(5 * time.Second)
counter.IncrementEvents(1)
counter.IncrementBytes(1024)
currentMetrics := counter.GetMetrics()
```

### States (`states.go`)

Defines process states for the DAQ system.

**State Constants:**
```go
const (
    INITIALIZED State = "INITIALIZED"
    STARTING    State = "STARTING"
    RUNNING     State = "RUNNING"
    STOPPING    State = "STOPPING"
    STOPPED     State = "STOPPED"
    PINGING     State = "PINGING"
)
```

**Usage:**
```go
state := pkg.INITIALIZED
logger.State(state)
```

### Centrifuge Client (`centrifugal.go`)

WebSocket client for real-time pub/sub messaging via Centrifuge server.

**CentrifugeClient:**
```go
type CentrifugeClient struct {
    client    *centrifuge.Client
    connected bool
    mu        sync.RWMutex
}
```

**Key Methods:**
- `Connect()` - Connect to Centrifuge server
- `Subscribe(channel)` - Subscribe to a channel
- `Publish(channel, data)` - Publish message to channel
- `OnMessage(handler)` - Set message handler

**Usage:**
```go
client := centrifugal.NewClient("ws://localhost:8000/connection/websocket")
client.Connect()
client.Subscribe("duck")
client.Publish("duck", messageJSON)
```

**Message Structure:**
```go
type Message struct {
    Timestamp int64       `json:"timestamp"`
    Host      string      `json:"host"`
    Type      string      `json:"type"`  // LOG, ERROR, METRICS, STATE, etc.
    Value     interface{} `json:"value"`
    RunNumber int         `json:"runNumber"`
}
```

### Prometheus (`prometheus.go`, `metrics.go`)

Prometheus metrics integration for monitoring.

**Key Functions:**
- `RegisterCounter(name, help)` - Register counter metric
- `RegisterGauge(name, help)` - Register gauge metric
- `IncrementCounter(name)` - Increment counter
- `SetGauge(name, value)` - Set gauge value

**Usage:**
```go
pkg.RegisterCounter("events_total", "Total events processed")
pkg.RegisterGauge("events_rate_hz", "Current event rate")

pkg.IncrementCounter("events_total")
pkg.SetGauge("events_rate_hz", 42.5)
```

Access metrics at `http://localhost:<prometheus-port>/metrics`

### Detector Configuration (`detectorConfiguration.go`)

Configuration structures for detector components.

**Key Structures:**
```go
type DetectorConfig struct {
    LDCs       []LDCConfig
    GDCs       []GDCConfig
    Equipments []EquipmentConfig
    Params     DuckParams
}

type LDCConfig struct {
    ID             int
    Host           string
    IP             string
    Port           int
    GRPCPort       int
    PrometheusPort int
    Equipments     []EquipmentConfig
}

type GDCConfig struct {
    ID                 int
    Host               string
    IP                 string
    Port               int
    GRPCPort           int
    PrometheusPort     int
    DataPath           string
    WriteOutputEnable  bool
    DecodeEnable       bool
    WriteDataEnable    bool
}

type EquipmentConfig struct {
    ID       int
    Type     string  // "energy" or "tracking"
    DeviceIP string
    HostIP   string
    HostPort int
    LDCID    int
}
```

### Decoder Configuration (`decoderConfiguration.go`)

Configuration for event decoding and HDF5 writing.

**Key Structures:**
```go
type DecoderConfig struct {
    WriteData       bool
    NWorkers        int
    BufferSize      int
    Calibrations    string
    TriggerCode     int
    PMTFlag         bool
    SiPMFlag        bool
    CompressionLevel int
}
```

### DATE Format Utilities (`date.go`)

Utilities for working with DATE format binary data.

**Key Functions:**
- `ParseDATEHeader(data)` - Parse DATE event header
- `WriteDATEHeader(header)` - Write DATE event header
- `ValidateMagicNumber(data)` - Validate DATE magic number

### Statistics (`statistics.go`)

Run statistics structures.

**Key Structures:**
```go
type RunStatistics struct {
    RunNumber   int
    StartTime   time.Time
    StopTime    time.Time
    TotalEvents int64
    TotalBytes  int64
    TotalErrors int64
    Duration    time.Duration
}
```

### Database (`database/`)

Generated code from sqlc for type-safe database access.

**Key Files:**
- `models.go` - Database table models
- `querier.go` - Querier interface
- `*.sql.go` - Query implementations

**Usage:**
```go
queries := database.New(db)
gdcs, err := queries.ListGDCs(ctx)
ldc, err := queries.GetLDC(ctx, 1)
```

## Common Patterns

### Configuration Loading

```go
// Load YAML config
config, err := configuration.ReadConfig("config.yaml")

// Connect to database
db, err := configuration.GetDBConnection(config)

// Load system configuration from database
detectorConfig, err := configuration.ReadDBConfiguration(db)
```

### Logging and Metrics

```go
// Create logger
centrifuge := centrifugal.NewClient(config.Centrifuge.Host)
logger := logger.NewDuckLogger(centrifuge, "gdc1")

// Log messages
logger.Log("Processing started")
logger.Error("Connection failed")

// Track metrics
metricsCounter := metrics.NewMetricsCounter(5 * time.Second)
metricsCounter.IncrementEvents(1)

// Publish metrics periodically
ticker := time.NewTicker(5 * time.Second)
for range ticker.C {
    m := metricsCounter.GetMetrics()
    logger.Metrics(m)
}
```

### Real-time Messaging

```go
// Connect to Centrifuge
client := centrifugal.NewClient("ws://localhost:8000/connection/websocket")
client.Connect()

// Subscribe to channel
client.Subscribe("duck")

// Handle incoming messages
client.OnMessage(func(msg centrifuge.Message) {
    // Process message
})

// Publish message
message := pkg.Message{
    Timestamp: time.Now().Unix(),
    Host:      "gdc1",
    Type:      pkg.STATE,
    Value:     pkg.RUNNING,
}
client.Publish("duck", message)
```

## Testing

All packages have comprehensive tests:

```bash
# Test all packages
go test -v ./pkg/...

# Test specific package
go test -v ./pkg/logger/

# Test with coverage
go test -cover ./pkg/...
```

## Dependencies

- Go 1.24+
- `github.com/centrifugal/centrifuge-go` - WebSocket client
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/prometheus/client_golang` - Prometheus metrics
- `gopkg.in/yaml.v3` - YAML parsing

## Files

| File | Purpose |
|------|---------|
| `configuration.go` | Configuration loading from YAML/DB |
| `logger.go` | DuckLogger with Centrifuge publishing |
| `metrics.go` | Metrics tracking and rate calculation |
| `states.go` | Process state definitions |
| `centrifugal.go` | Centrifuge WebSocket client |
| `prometheus.go` | Prometheus metrics helpers |
| `detectorConfiguration.go` | Detector config structures |
| `decoderConfiguration.go` | Decoder config structures |
| `date.go` | DATE format utilities |
| `statistics.go` | Run statistics structures |
| `database/` | Generated sqlc code |

## Best Practices

1. **Error Handling**: Always check errors from configuration loading
2. **Connection Management**: Use connection pooling for database
3. **Metrics**: Update metrics atomically (thread-safe)
4. **Logging**: Use appropriate log levels (LOG, ERROR, METRICS, etc.)
5. **Configuration**: Validate configuration before use
6. **Testing**: Maintain >80% test coverage

## Related

- `api/` - Uses configuration and database packages
- `ldcRPC/` - Uses logger, metrics, states
- `gdcRPC/` - Uses logger, metrics, states, decoder config
- `database/` - Schema and queries
