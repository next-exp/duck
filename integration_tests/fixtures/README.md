# Integration Test Configurations

This directory contains test configuration files used by the integration test suite.

## Configuration Files

### `ldc_test.yml`
**Purpose**: LDC (Local Data Collector) integration tests
**Used by**: `integration_tests/ldc/`
**Components**:
- Database connection
- Single LDC configuration
- Single equipment configuration
- Empty GDC array (tests create dummy GDC listeners)

**Tests**:
- UDP packet receiving and processing
- State transitions (INITIALIZED ↔ RUNNING)
- Statistics tracking
- Event processing

### `gdc_test.yml`
**Purpose**: GDC (Global Data Collector) integration tests
**Used by**: `integration_tests/gdc/`
**Components**:
- Database connection
- Single GDC configuration
- Empty LDC array

**Tests**:
- GDC lifecycle (start/stop)
- ConnectRPC interface
- Statistics retrieval
- State transitions

### `e2e_test.yml`
**Purpose**: End-to-end system integration tests
**Used by**: `integration_tests/e2e/`
**Components**:
- Database connection
- Complete LDC configuration
- Complete GDC configuration
- Equipment configuration

**Tests**:
- Complete data flow: Equipment → LDC → GDC → File
- LDC-GDC communication
- State synchronization
- Error propagation

## Database vs YAML Configuration

The test configurations use a hybrid approach:

1. **YAML files** provide:
   - Database connection parameters
   - Service (LDC/GDC) network settings
   - Equipment definitions
   - Enable/disable flags

2. **Database seed data** (`database/integration_seed.sql`) provides:
   - Initial LDC configurations
   - GDC configurations
   - Equipment assignments
   - Channel mappings
   - Huffman codes

### Runtime Patching

During test execution, the testhelpers automatically patch configurations:

```go
// In testhelpers/services.go:
func ensureMySQLForNetworkTest(t *testing.T, configFilename string) {
    // 1. Start MySQL container
    pool, resource := SetupMySQLContainer(t)

    // 2. Patch YAML config with container's mapped port
    cfg.Database.Port = resource.GetPort("3306/tcp")

    // 3. Update LDC hostnames in database to match test host
    _, err = db.Exec("UPDATE ldcs SET hostname=?", host)

    // 4. Disable extra equipments
    _, err = db.Exec("UPDATE equipments SET enabled=false WHERE ldcID=? AND id<>?", 1, 1)

    // 5. Start dummy GDC listeners for enabled GDCs
    for each enabled GDC:
        start TCP listener on configured port
}
```

This allows tests to:
- Use fresh Docker containers for each test
- Avoid port conflicts
- Isolate test environments
- Test with real database operations

## Test Isolation Strategy

Each test gets:
1. **Fresh Docker containers** - MySQL, Centrifugo (if needed)
2. **Clean database state** - Schema + seed data loaded per test
3. **Dynamic port assignment** - No port conflicts between parallel tests
4. **Automatic cleanup** - Containers removed on test completion

## Configuration Best Practices

When adding new test configurations:

1. **Use descriptive names**: `ldc_test.yml`, `gdc_test.yml`, `e2e_test.yml`
2. **Document purpose**: Add comments explaining what the config tests
3. **Keep it minimal**: Only include components needed for that test suite
4. **Use empty arrays**: Explicitly show what's NOT included (e.g., `ldcs: []`)
5. **Match test scope**: LDC tests don't need real GDCs, use dummy listeners

## Example: Creating a New Test Config

```yaml
# my_feature_test.yml - Tests new feature
database:
  hostname: localhost
  port: 3306
  username: root
  password: password
  database: duck

# Only components needed for this test
ldcs:
  - name: ldc_test
    hostname: localhost
    grpc_port: 18000
    enabled: true
    equipments: [...]  # Minimal equipment config

# Explicitly show what's not needed
gdcs: []  # Feature doesn't use GDC
```

## Port Ranges

To avoid conflicts, test configurations use specific port ranges:

| Component | Port Range | Usage |
|-----------|------------|-------|
| MySQL | 3306, 33092+ | Database (Docker mapped) |
| LDC gRPC | 17002-17999 | LDC ConnectRPC |
| GDC gRPC | 17001-17999 | GDC ConnectRPC |
| GDC TCP | 17000-17999 | LDC→GDC data |
| Equipment UDP | 16000-16999 | Equipment→LDC data |
| Centrifugo | 8000+ | Message broker |

**Note**: Test configurations avoid port conflicts by using non-overlapping port ranges.

## Related Files

- `integration_tests/fixtures/database/integration_seed.sql` - Database seed data (USED by testhelpers/docker.go)
- `integration_tests/testhelpers/docker.go` - Docker container management
- `integration_tests/testhelpers/services.go` - Service lifecycle management
- `integration_tests/README.md` - Integration test overview
