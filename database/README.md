# Database Schema

MySQL/MariaDB database schema and queries for the NEXT-100 DAQ system configuration and 
run metadata management.

## Overview

The database stores:
- System configuration (LDCs, GDCs, equipment)
- Run metadata (run numbers, timestamps, statistics)
- Real-time monitoring data (rates)
- Decoder and test device parameters

## Schema

### Configuration Tables

#### `ldcs` - Local Data Concentrators

```sql
CREATE TABLE ldcs (
    id INT PRIMARY KEY AUTO_INCREMENT,
    host VARCHAR(255) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    port INT NOT NULL,
    grpcPort INT NOT NULL,
    prometheusPort INT NOT NULL,
    enabled BOOLEAN DEFAULT true
);
```

Manages LDC server configurations.

#### `gdcs` - Global Data Concentrators

```sql
CREATE TABLE gdcs (
    id INT PRIMARY KEY AUTO_INCREMENT,
    host VARCHAR(255) NOT NULL,
    ip VARCHAR(45) NOT NULL,
    port INT NOT NULL,
    grpcPort INT NOT NULL,
    prometheusPort INT NOT NULL,
    dataPath VARCHAR(512) NOT NULL,
    writeOutputEnable BOOLEAN DEFAULT true,
    decodeEnable BOOLEAN DEFAULT false,
    writeDataEnable BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true
);
```

Manages GDC server configurations with data writing options.

#### `equipments` - Equipment/Detector Boards

```sql
CREATE TABLE equipments (
    id INT PRIMARY KEY,
    type ENUM('energy', 'tracking') NOT NULL,
    deviceIP VARCHAR(45) NOT NULL,
    hostIP VARCHAR(45) NOT NULL,
    hostPort INT NOT NULL,
    ldcID INT NOT NULL,
    enabled BOOLEAN DEFAULT true,
    FOREIGN KEY (ldcID) REFERENCES ldcs(id)
);
```

Maps ATCA boards (equipment) to LDCs.

#### `duckParams` - DAQ System Parameters

```sql
CREATE TABLE duckParams (
    id INT PRIMARY KEY AUTO_INCREMENT,
    bufferSize INT NOT NULL,
    channelSize INT NOT NULL,
    -- Additional DAQ parameters
);
```

System-wide DAQ configuration parameters.

#### `decoderParams` - Event Decoder Settings

```sql
CREATE TABLE decoderParams (
    id INT PRIMARY KEY AUTO_INCREMENT,
    triggerCode INT NOT NULL,
    pmtFlag BOOLEAN DEFAULT true,
    sipmFlag BOOLEAN DEFAULT true,
    compressionLevel INT DEFAULT 5,
    -- Additional decoder parameters
);
```

Configuration for the event decoder.

#### `testDeviceParams` / `simulatorParams` - Test Device Settings

```sql
CREATE TABLE testDeviceParams (
    equipmentID INT PRIMARY KEY,
    eventRate FLOAT DEFAULT 10.0,
    packetSize INT DEFAULT 500,
    FOREIGN KEY (equipmentID) REFERENCES equipments(id)
);

CREATE TABLE simulatorParams (
    equipmentID INT PRIMARY KEY,
    filePath VARCHAR(512),
    replayRate FLOAT DEFAULT 1.0,
    loopMode BOOLEAN DEFAULT false,
    maxEvents INT DEFAULT 0,
    packetSize INT DEFAULT 500,
    FOREIGN KEY (equipmentID) REFERENCES equipments(id)
);
```

Test device and simulator-specific settings.

### Run Data Tables

#### `runs` - Run Numbers and Metadata

```sql
CREATE TABLE runs (
    run INT PRIMARY KEY,
    startTime TIMESTAMP NOT NULL,
    stopTime TIMESTAMP NULL,
    disabled BOOLEAN DEFAULT false
);
```

Tracks run numbers with start/stop timestamps.

#### `events` - Event Counts

```sql
CREATE TABLE events (
    run INT NOT NULL,
    gdcID INT NOT NULL,
    ldcID INT NOT NULL,
    eventCount BIGINT NOT NULL,
    PRIMARY KEY (run, gdcID, ldcID),
    FOREIGN KEY (run) REFERENCES runs(run)
);
```

Event counts per run/GDC/LDC combination.

#### `data` - Data Volumes

```sql
CREATE TABLE data (
    run INT NOT NULL,
    gdcID INT NOT NULL,
    ldcID INT NOT NULL,
    byteCount BIGINT NOT NULL,
    PRIMARY KEY (run, gdcID, ldcID),
    FOREIGN KEY (run) REFERENCES runs(run)
);
```

Byte counts per run/GDC/LDC for monitoring data volumes.

#### `errors` - Error Tracking

```sql
CREATE TABLE errors (
    run INT NOT NULL,
    gdcID INT NOT NULL,
    ldcID INT NOT NULL,
    errorCount INT NOT NULL,
    PRIMARY KEY (run, gdcID, ldcID),
    FOREIGN KEY (run) REFERENCES runs(run)
);
```

Error counts per run/GDC/LDC for monitoring system health.

#### `rates` - Real-time Rate Monitoring

```sql
CREATE TABLE rates (
    timestamp TIMESTAMP PRIMARY KEY,
    totalBytes BIGINT NOT NULL,
    totalEvents BIGINT NOT NULL,
    byteRate FLOAT NOT NULL,
    triggerRate FLOAT NOT NULL
);
```

Real-time aggregated rates from all GDCs (updated by rateReader service).

## Queries

Queries are defined in `queries/*.sql` using sqlc format for type-safe Go code.

### Configuration Queries

#### `ldcs.sql`

```sql
-- name: GetLDC :one
SELECT * FROM ldcs WHERE id = ?;

-- name: ListLDCs :many
SELECT * FROM ldcs ORDER BY id;

-- name: CreateLDC :exec
INSERT INTO ldcs (host, ip, port, grpcPort, prometheusPort, enabled)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateLDC :exec
UPDATE ldcs
SET host = ?, ip = ?, port = ?, grpcPort = ?, prometheusPort = ?, enabled = ?
WHERE id = ?;

-- name: DeleteLDC :exec
DELETE FROM ldcs WHERE id = ?;
```

#### `gdcs.sql`

```sql
-- name: GetGDC :one
SELECT * FROM gdcs WHERE id = ?;

-- name: ListGDCs :many
SELECT * FROM gdcs ORDER BY id;

-- name: CreateGDC :exec
INSERT INTO gdcs (host, ip, port, grpcPort, prometheusPort, dataPath,
                  writeOutputEnable, decodeEnable, writeDataEnable, enabled)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateGDC :exec
UPDATE gdcs
SET host = ?, ip = ?, port = ?, grpcPort = ?, prometheusPort = ?,
    dataPath = ?, writeOutputEnable = ?, decodeEnable = ?,
    writeDataEnable = ?, enabled = ?
WHERE id = ?;

-- name: DeleteGDC :exec
DELETE FROM gdcs WHERE id = ?;
```

#### `equipments.sql`

```sql
-- name: GetEquipment :one
SELECT * FROM equipments WHERE id = ?;

-- name: ListEquipments :many
SELECT * FROM equipments ORDER BY id;

-- name: ListEquipmentsByLDC :many
SELECT * FROM equipments WHERE ldcID = ? ORDER BY id;

-- name: CreateEquipment :exec
INSERT INTO equipments (id, type, deviceIP, hostIP, hostPort, ldcID, enabled)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateEquipment :exec
UPDATE equipments
SET type = ?, deviceIP = ?, hostIP = ?, hostPort = ?, ldcID = ?, enabled = ?
WHERE id = ?;

-- name: DeleteEquipment :exec
DELETE FROM equipments WHERE id = ?;
```

### Run Management Queries

#### `config.sql`

```sql
-- name: GetRunNumber :one
SELECT MAX(run) as run FROM runs;

-- name: CreateRun :exec
INSERT INTO runs (run, startTime, disabled)
VALUES (?, NOW(), ?);

-- name: UpdateRunStopTime :exec
UPDATE runs SET stopTime = NOW() WHERE run = ?;

-- name: GetConfiguration :many
SELECT
    l.id as ldcID, l.host as ldcHost, l.ip as ldcIP, l.port as ldcPort,
    l.grpcPort as ldcGRPCPort, l.prometheusPort as ldcPrometheusPort,
    g.id as gdcID, g.host as gdcHost, g.ip as gdcIP, g.port as gdcPort,
    g.grpcPort as gdcGRPCPort, g.prometheusPort as gdcPrometheusPort,
    e.id as equipmentID, e.type as equipmentType, e.deviceIP,
    e.hostIP, e.hostPort
FROM ldcs l
JOIN equipments e ON l.id = e.ldcID
JOIN gdcs g ON g.enabled = true
WHERE l.enabled = true AND e.enabled = true
ORDER BY l.id, e.id;
```

## Code Generation

This directory uses [sqlc](https://sqlc.dev/) to generate type-safe Go code from SQL queries.

### Generate Code

```bash
# Install sqlc
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Generate code
sqlc generate

# Or use the project task
task gen:sql
```

### Generated Files

After generation, `pkg/database/` contains:
- `models.go` - Go structs for database tables
- `querier.go` - Interface for all queries
- `*.sql.go` - Implementations of each query

## Setup

### Create Database

```bash
# Connect to MySQL
mysql -u root -p

# Create database
CREATE DATABASE duck CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

# Create user
CREATE USER 'duck'@'localhost' IDENTIFIED BY 'duck_password';
GRANT ALL PRIVILEGES ON duck.* TO 'duck'@'localhost';
FLUSH PRIVILEGES;
```

### Load Schema

```bash
# Load schema
mysql -u duck -p duck < database/schema.sql

# Load seed data (optional)
mysql -u duck -p duck < database/seed.sql
```

### Environment Variables

```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=duck
export DB_PASS=duck_password
export DB_NAME=duck
```

## Migrations

For schema changes:

1. **Modify schema.sql**
2. **Update affected queries in queries/*.sql**
3. **Regenerate code**: `sqlc generate`
4. **Test changes**: `go test ./pkg/database/...`
5. **Apply to production**: `mysql < database/schema.sql`

## Backup

```bash
# Backup database
mysqldump -u duck -p duck > backup_$(date +%Y%m%d).sql

# Restore database
mysql -u duck -p duck < backup_20250306.sql
```

## Performance

- **Indexes**: Primary keys on all tables
- **Foreign Keys**: Enforce referential integrity
- **Connection Pooling**: Handled by Go driver
- **Query Optimization**: Use EXPLAIN for slow queries

## Files

| File | Purpose |
|------|---------|
| `schema.sql` | Complete database schema |
| `queries/ldcs.sql` | LDC CRUD queries |
| `queries/gdcs.sql` | GDC CRUD queries |
| `queries/equipments.sql` | Equipment CRUD queries |
| `queries/config.sql` | Configuration and run queries |
| `queries/simulators.sql` | Simulator parameter queries |
| `.env.example` | Environment variable template |

## Related

- `pkg/database/` - Generated Go code
- `api/` - Uses generated queries
- Configuration loading in `pkg/configuration.go`
