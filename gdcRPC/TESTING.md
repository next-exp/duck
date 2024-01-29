# gdcRPC Testing Guide

This guide covers testing practices for the gdcRPC package, which implements the Global Data Collector (GDC) for the NEXT DAQ system.

## Quick Start

### Run Unit Tests Locally (Fast)
```bash
# Using Taskfile
task test-gdcrpc-unit

# Using go directly
go test -v -tags=nohdf5 ./gdcRPC/
```

### Run All Tests (Including Docker-based)
```bash
# Using Taskfile
task test-gdcrpc-all

# Or separately
task test-gdcrpc-unit          # Local tests
task test-gdcrpc-integration   # Docker tests
```

### Generate Coverage Report
```bash
task test-gdcrpc-coverage
```

## Test Structure

### Test Files

| Test File | Description | Build Tag | Tests |
|-----------|-------------|-----------|-------|
| `io_test.go` | File operations, filename generation | `nohdf5` | 15+ |
| `processEvents_test.go` | Event assembly, mountEvent, readData | `nohdf5` | 30+ |
| `ldcData_test.go` | TCP packet handling, fragmentation | `nohdf5` | 20+ |
| `main_test.go` | gRPC server, state management, concurrency | `nohdf5` | 30+ |
| `gdc_test.go` | TCP listener, connection monitoring | `nohdf5` | 10+ |
| `binaryWriter_test.go` | Binary file writer goroutine | `nohdf5` | 10+ |
| `headers_test.go` | GDC header creation | None | 8 |
| `fileCloser_test.go` | File synchronization | None | 9 |
| `decoder_test.go` | Decoder integration (requires Docker) | None | TBD |

**Total: 140+ tests**

### Test Helpers

Located in `testhelpers/`:
- `fixtures.go` - Test data creation (GDC/LDC configurations, event data)
- `mocks.go` - Mock implementations (TCPConn, FileCloser, context builder)

## Build Tags Explained

### `//go:build nohdf5`
Tests with this tag can run **without HDF5/Blosc dependencies**. These test pure Go code:
- File I/O operations
- Event assembly logic
- TCP packet handling
- State management
- Connection monitoring

### No Build Tag
Tests without build tags require Docker to build, as they depend on:
- HDF5 library
- Blosc compression
- External decoder package

## Running Specific Tests

### By Test File
```bash
# Test only io functions
go test -v -tags=nohdf5 -run TestGetDecodedOutputFilename ./gdcRPC/

# Test only processEvents
go test -v -tags=nohdf5 -run TestMountEvent ./gdcRPC/
```

### By Pattern
```bash
# Test all count functions
go test -v -tags=nohdf5 -run TestCount ./gdcRPC/

# Test concurrent access
go test -v -tags=nohdf5 -run Concurrent ./gdcRPC/
```

### With Race Detector
```bash
task test-gdcrpc-race
```

### With Benchmarks
```bash
task test-gdcrpc-bench
```

## Test Patterns

### Table-Driven Tests
For pure functions with multiple input combinations:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {"valid input", validInput, expectedOutput, false},
        {"invalid input", invalidInput, nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.want, got)
            }
        })
    }
}
```

### Channel-Based Tests
For goroutines and concurrent operations:

```go
func TestGoroutine(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    dataCh := make(chan DataType, 10)
    done := make(chan bool, 1)

    go func() {
        FunctionUnderTest(dataCh, ctx)
        done <- true
    }()

    dataCh <- testData

    select {
    case <-done:
        // Success
    case <-ctx.Done():
        t.Fatal("Timeout")
    }
}
```

### TempDir Usage
For file operations requiring cleanup:

```go
func TestFileOperation(t *testing.T) {
    tempDir := t.TempDir()  // Automatically cleaned up

    // Create files in tempDir
    // ...
}
```

## Writing New Tests

### 1. Choose Build Tag
- Use `//go:build nohdf5` for tests that don't require HDF5
- Omit build tag only for decoder/integration tests

### 2. Use Test Helpers
```go
import "github.com/jmbenlloch/next_duck/gdcRPC/testhelpers"

// Create test context
ctx := testhelpers.NewMockContextBuilder().
    WithRunNumber(123).
    WithGDCConfiguration(&gdcConfig).
    WithWriteOutput(true).
    Build()

// Create test configuration
gdcConfig := testhelpers.NewTestGDCConfiguration(1, "gdc1")

// Create test event data
eventData := testhelpers.CreateLDCEventData(100, 1, 200)
```

### 3. Follow Naming Conventions
- Test functions: `Test<FunctionName>_<Scenario>`
- Subtests: Use `t.Run()` for readability
- Table-driven tests: Name each case descriptively

### 4. Handle Contexts Properly
```go
// Always cancel contexts
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// For cancellable contexts in tests
baseCtx, cancel := context.WithCancel(context.Background())
ctx = context.WithValue(baseCtx, "cancelCtx", cancel)
```

### 5. Use Appropriate Assertions
```go
import "github.com/stretchr/testify/assert"
import "github.com/stretchr/testify/require"

// require - fails test immediately
require.NoError(t, err)
require.NotNil(t, result)

// assert - records failure but continues
assert.Equal(t, expected, actual)
assert.True(t, condition)
```

## Test Categories

### Unit Tests (Pure Go)
Run locally without Docker:
- `TestGetDecodedOutputFilename*` - Filename generation
- `TestOpenFile*` - File creation
- `TestWriteBinaryData*` - Binary write operations
- `TestCountEnabledLDCs*` - Count enabled LDCs
- `TestCountEnabledGDCs*` - Count enabled GDCs
- `TestMountEvent*` - Event assembly logic
- `TestProcessData*` - Packet reassembly
- `TestAcceptTCPConnections*` - TCP acceptance
- `TestLDCConnectionMonitor*` - Connection tracking

### Concurrency Tests (Race Condition Detection)
Run with `-race` flag to detect data races:
- `TestServer_StartRun_ConcurrentAccess` - Verifies only one goroutine succeeds when multiple concurrent StartRun calls are made
- `TestServer_StopRun_ConcurrentAccess` - Concurrent StopRun calls
- `TestServer_StartStop_RaceCondition` - Rapid start/stop cycles
- `TestChangeStateOnStop_ConcurrentCalls` - Concurrent state change monitoring
- `TestServer_ThreadSafeStateAccess` - Thread-safe state accessor methods

### Other Concurrency Tests
Test thread-safety and race conditions:
- `TestMountEvent_ConcurrentAccess`
- `TestProcessData_ConcurrentProcessing`
- `TestLDCConnectionMonitor_ConcurrentAccess`
- `TestBinaryWriter_ConcurrentWrites`

### Metrics Verification Tests
Verify that metrics are properly managed:
- `TestServer_StartRun_ResetsMetrics` - Verifies all metrics reset to zero on StartRun
- `TestServer_StartRun_MetricsResetAcrossCycles` - Verifies metrics reset across multiple start/stop cycles
- `TestServer_GetRunStatistics_ReturnsCounters` - Verifies GetRunStatistics returns correct metrics

### Context Propagation Tests
Verify context setup and cancellation:
- `TestServer_StartRun_ContextSetup` - Verifies context values (cancelCtx, server) are set
- `TestServer_StartRun_ContextCancellation` - Verifies context can be cancelled
- `TestServer_StartRun_ChangeStateOnStopGoroutine` - Verifies changeStateOnStop goroutine behavior
- `TestServer_MultipleContextCreations` - Verifies new contexts are created on each StartRun

### Integration Tests (Docker)
Require HDF5/Blosc:
- `TestDecodeEvent*` - Full decoder pipeline
- `TestDecodedWriter*` - HDF5 file writing
- `TestGDC_CompleteDataFlow` - End-to-end data flow

## Common Issues

### Missing Build Tag
**Error**: `cannot find package`
**Solution**: Add `-tags=nohdf5` to go test command

### HDF5 Dependency
**Error**: `blosc_filter.h: No such file`
**Solution**: Use `-tags=nohdf5` or run in Docker with `task test-gdcrpc-integration`

### Context Not Canceled
**Error**: Test timeout
**Solution**: Always call `defer cancel()` or use `t.Cleanup()`

### File Not Cleaned Up
**Error**: Temp files remain after test
**Solution**: Use `t.TempDir()` instead of `os.MkdirTemp()`

## Coverage Goals

| Phase | Target Coverage | Status |
|-------|----------------|--------|
| Phase 2: Pure Go Functions | 60-70% | ✅ Complete |
| Phase 3: HDF5 Integration | 75-80% | ⏳ Pending |
| Phase 4: Integration Tests | 85%+ | ⏳ Pending |

## CI/CD Integration

### GitHub Actions
Tests run automatically on:
- Pull requests
- Push to main branch
- Release tags

### Pre-commit Hook
Optional: Add pre-commit hook to run tests before pushing:
```bash
# .git/hooks/pre-commit
#!/bin/sh
go test -tags=nohdf5 ./gdcRPC/
```

## Debugging Tests

### Verbose Output
```bash
go test -v -tags=nohdf5 ./gdcRPC/
```

### Run Single Test
```bash
go test -v -tags=nohdf5 -run TestMountEvent_SingleLDC ./gdcRPC/
```

### Print Debug Output
Tests use t.Logf for debug output:
```go
t.Logf("Event data: %+v", eventData)
```

### Race Detection
```bash
go test -race -tags=nohdf5 ./gdcRPC/
```

## Resources

- [Go Testing Guide](https://golang.org/doc/tutorial/add-a-test)
- [Testify Assertions](https://github.com/stretchr/testify)
- [TableDrivenTests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Context Package](https://golang.org/pkg/context/)

## Contributing

When adding new features to gdcRPC:
1. Write tests first (TDD)
2. Ensure 80%+ coverage for new code
3. Test concurrent access if goroutines are used
4. Add integration tests for external dependencies
5. Update this guide if new patterns are introduced

## Test Execution Examples

```bash
# Quick check during development
go test -tags=nohdf5 ./gdcRPC/

# Full test suite with coverage
task test-gdcrpc-coverage

# Debug specific test
go test -v -tags=nohdf5 -run TestProcessData_MultipleEventsInOneRead ./gdcRPC/

# Run with race detector
go test -race -tags=nohdf5 ./gdcRPC/

# Docker-based integration tests
task test-gdcrpc-integration
```
