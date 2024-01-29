# Integration Tests

This directory contains integration tests for the Duck DAQ system, testing real interactions between components with actual dependencies like MySQL and messaging systems.

## Overview

Integration tests fill the critical gap between unit tests and production by testing:

- **Real Database Operations**: Actual MySQL with schema and data
- **Component Communication**: gRPC, TCP, UDP protocols
- **Data Flow**: End-to-end from equipment to files
- **Failure Scenarios**: Network issues, service failures
- **Configuration Loading**: From database, not files

## Test Structure

```
integration_tests/
├── database/          # Database integration tests (P0 - CRITICAL)
│   └── config_test.go
├── messaging/         # Centrifuge messaging tests (P1 - HIGH)
├── network/           # UDP/TCP communication tests (P0 - CRITICAL)
├── grpc/              # gRPC orchestration tests (P1 - HIGH)
├── e2e/               # End-to-end flow tests (P0 - CRITICAL)
└── testhelpers/       # Test utilities and Docker management
    ├── docker.go      # Docker container helpers
    ├── data.go        # Test data generation
    └── services.go    # Service lifecycle management
```

## Quick Start

### Prerequisites

1. **Docker**: Required for running test dependencies
   ```bash
   # Verify Docker is available
   docker --version
   ```

2. **Go Dependencies**: Integration test dependencies
   ```bash
   go mod download
   ```

### Running Tests

#### Run All Integration Tests
```bash
# Run all integration tests (requires build tag)
go test -v -tags=integration ./integration_tests/...

# Run with timeout (recommended)
go test -v -tags=integration -timeout=30m ./integration_tests/...
```

#### Run Specific Test Suites
```bash
# Database integration tests
go test -v -tags=integration ./integration_tests/database/...

# Skip short mode checks
go test -v -tags=integration ./integration_tests/database/... -short
```

#### Run in Short Mode (for CI)
```bash
# Skip integration tests in short mode
go test -short ./...
```

## Test Suites

### 1. Database Integration (✅ IMPLEMENTED)

**Priority**: P0 - Critical
**Location**: `integration_tests/database/`

Tests real database operations with MySQL:
- Configuration loading from database
- CRUD operations persistence
- Run control database operations
- Foreign key constraint enforcement

**Tests**:
- `TestReadConfiguration_RealDatabase` - Load config from MySQL
- `TestReadConfiguration_EnabledFiltering` - Verify filtering works
- `TestReadConfiguration_EquipmentAssignment` - Verify equipment mapping

### 2. Network Communication (🚧 TODO)

**Priority**: P0 - Critical
**Location**: `integration_tests/ldc/`

Tests actual data flow between components:
- LDC receives and assembles UDP events
- LDC sends events to GDC via TCP
- GDC receives from multiple LDCs
- Network failure recovery

### 3. gRPC Communication (🚧 TODO)

**Priority**: P1 - High
**Location**: `integration_tests/grpc/`

Tests service orchestration:
- API starts LDC via gRPC
- API stops GDC via gRPC
- API pings devices
- API fetches statistics

### 4. Centrifuge Messaging (🚧 TODO)

**Priority**: P1 - High
**Location**: `integration_tests/messaging/`

Tests pub/sub messaging:
- Metrics published to Centrifuge
- State changes propagated
- Error messages with stop flag
- Reconnection after disconnect

### 5. End-to-End Data Flow (🚧 TODO)

**Priority**: P0 - Critical
**Location**: `integration_tests/e2e/`

Tests complete system workflows:
- Single event flows through system
- High throughput (10,000 events)
- File rotation during run
- Graceful lifecycle management

### 6. Failure Scenarios (🚧 TODO)

**Priority**: P1 - High
**Location**: `integration_tests/failure/`

Tests system resilience:
- GDC disconnects during run
- Database connection lost
- Centrifuge unavailable
- Disk full on GDC

## Docker Infrastructure

### Docker Compose Files

- `docker-compose.integration.yml` - Main integration test infrastructure
- `docker-compose.e2e.yml` - End-to-end test environment

### Services

- **MySQL 8.0**: Database with schema and seed data
- **Centrifuge**: Pub/sub messaging server
- **Test Services**: LDC, GDC, API containers (TODO)

## Test Data

### Database Fixtures

- `integration_tests/fixtures/database/integration_seed.sql` - Test data for integration tests
- `database/db.sql` - Main database schema

### Configuration Files

- `integration_tests/fixtures/configs/integration_test.yml` - Test configuration

## Test Helpers

### Docker Management

```go
// Start MySQL with schema and seed data
pool, resource := testhelpers.SetupMySQLContainer(t)
defer testhelpers.CleanupDocker(pool, resource)

// Get connection string for tests
connStr := testhelpers.GetMySQLConnectionString(resource)
```

### Data Generation

```go
// Generate test UDP packets for events
packets := testhelpers.GenerateTestEvent(eventID, numPackets)

// Create UDP sender for testing
sender := testhelpers.CreateUDPSender("127.0.0.1", 16006)
```

## Implementation Notes

### Build Tags

Integration tests use build tags to avoid running in normal unit test suites:

```go
//go:build integration
// +build integration

package database_test
```

### Database Schema

The integration tests:
1. Create fresh MySQL container
2. Load schema (`database/schema.sql`)
3. Load seed data (`integration_tests/fixtures/database/integration_seed.sql`)
4. Run tests against real data
5. Clean up containers

### Test Isolation

Each test gets:
- Fresh Docker containers
- Clean database state
- Isolated network environment
- Automatic cleanup

## Troubleshooting

### Docker Issues

**Problem**: `MySQL did not become ready in time`
```bash
# Check Docker daemon
docker info

# Check for port conflicts
docker ps | grep mysql
```

**Problem**: Container startup timeouts
```bash
# Increase timeout in test helpers
pool.MaxWait = 120 * time.Second  # Default: 60 seconds
```

### Database Issues

**Problem**: Table doesn't exist errors
```bash
# Manually verify schema
docker exec -it <container> mysql -u root -ptestpass duck_test -e "SHOW TABLES;"
```

**Problem**: Connection refused
```bash
# Check if MySQL is listening
docker exec -it <container> netstat -tlnp | grep 3306
```

### Permission Issues

**Problem**: Permission denied on Unix socket
- Ensure test user has permissions
- Check MySQL container user mapping

## Performance

### Test Execution Times

- **Database Tests**: ~15 seconds each
- **Network Tests**: ~30 seconds estimated
- **E2E Tests**: ~2 minutes estimated
- **Full Suite**: ~5-10 minutes estimated

### Resource Usage

- **Memory**: ~512MB per MySQL container
- **Disk**: ~100MB for database files
- **CPU**: Minimal during execution

## CI/CD Integration

### GitHub Actions

See `.github/workflows/integration-tests.yml` for:
- Docker setup
- Service dependencies
- Test execution
- Artifact collection

### Local Development

```bash
# Run integration tests locally
task integration-test

# Run specific suite
task integration-test-component

# Run with coverage
go test -v -tags=integration -cover ./integration_tests/...
```

## Contributing

### Adding New Tests

1. Create test file in appropriate directory
2. Use build tags: `//go:build integration`
3. Follow naming pattern: `Test*Integration`
4. Use testhelpers for Docker management
5. Add cleanup with `defer`

### Example Test Structure

```go
//go:build integration
// +build integration

package database_test

import (
    "testing"
    "github.com/stretchr/testify/require"
    "github.com/jmbenlloch/next_duck/integration_tests/testhelpers"
)

func TestNewFeature_Integration(t *testing.T) {
    // Setup dependencies
    pool, resource := testhelpers.SetupMySQLContainer(t)
    defer testhelpers.CleanupDocker(pool, resource)

    // Test implementation
    // ...

    // Assertions
    require.NoError(t, err)
    require.Equal(t, expected, actual)
}
```

## Current Status

### ✅ Completed

- [x] Database integration infrastructure
- [x] Docker container management helpers
- [x] SQL schema and seed data loading
- [x] Database integration tests (3/3 passing)
- [x] Test configuration files
- [x] Build tag system

### 🚧 In Progress

- [ ] Network communication tests
- [ ] gRPC orchestration tests
- [ ] Centrifuge messaging tests

### 📋 Planned

- [ ] End-to-end data flow tests
- [ ] Failure scenario tests
- [ ] Performance benchmarks
- [ ] CI/CD pipeline integration

## References

- [Integration Testing Strategy](INTEGRATION_TESTS_STRATEGY.md) - Overall strategy and design
- [Implementation Progress](INTEGRATION_TESTS_PROGRESS.md) - Detailed progress tracking
- [Docker Compose Integration](docker-compose.integration.yml) - Infrastructure definition
- [Database Schema](../../database/schema.sql) - Main database schema
- [Test Configuration](testing/fixtures/configs/integration_test.yml) - Test configuration