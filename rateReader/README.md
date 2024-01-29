# Rate Reader Service

Background service that aggregates metrics from all GDCs and writes rate monitoring data 
to the database for real-time display in the web interface.

## Overview

The rate reader subscribes to the Centrifuge WebSocket to receive real-time metrics from 
all running GDCs, aggregates the data, and periodically writes summary statistics to the 
database `rates` table.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Rate Reader Service                     │
│                                                              │
│  ┌────────────────────────────────────────────────────┐     │
│  │         Centrifuge WebSocket Subscriber            │     │
│  │  - Subscribes to "duck" channel                    │     │
│  │  - Receives METRICS messages from all GDCs         │     │
│  └───────────────────┬────────────────────────────────┘     │
│                      │                                       │
│                      v                                       │
│  ┌────────────────────────────────────────────────────┐     │
│  │            Metrics Aggregator                       │     │
│  │  - Sum total events across all GDCs                │     │
│  │  - Sum total bytes across all GDCs                 │     │
│  │  - Calculate combined trigger rate (Hz)            │     │
│  │  - Calculate combined byte rate (MB/s)             │     │
│  └───────────────────┬────────────────────────────────┘     │
│                      │                                       │
│                      v                                       │
│  ┌────────────────────────────────────────────────────┐     │
│  │         Periodic Database Writer                    │     │
│  │  - Writes to 'rates' table every N seconds         │     │
│  │  - Configurable period (default: 5 seconds)        │     │
│  └───────────────────┬────────────────────────────────┘     │
│                      │                                       │
└──────────────────────┼───────────────────────────────────────┘
                       │
                       v
              ┌────────────────┐
              │  MySQL Database │
              │   rates table   │
              └────────────────┘
```

## Usage

```bash
# Basic usage
./rateReader -config config.yaml

# With custom update period
./rateReader -config config.yaml -period 10

# With verbose logging
./rateReader -config config.yaml -verbose
```

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | Required | Path to configuration YAML file |
| `-period` | 5 | Update period in seconds |
| `-verbose` | false | Enable verbose logging |

## Configuration

### YAML Configuration

```yaml
dbHost: localhost
dbPort: 3306
dbUser: root
dbPass: duck
dbName: duck

centrifugo:
  host: localhost
  port: 8000
  apiKey: your-api-key
  tokenHMACKey: your-hmac-key
```

### Environment Variables

Alternatively, use environment variables:

```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASS=duck
export DB_NAME=duck
export CENTRIFUGO_HOST=localhost
export CENTRIFUGO_PORT=8000
export CENTRIFUGO_API_KEY=your-api-key
```

## Database Schema

The service writes to the `rates` table:

```sql
CREATE TABLE rates (
    timestamp TIMESTAMP PRIMARY KEY,
    totalBytes BIGINT NOT NULL,
    totalEvents BIGINT NOT NULL,
    byteRate FLOAT NOT NULL,
    triggerRate FLOAT NOT NULL
);
```

## Metrics Aggregation

### Message Format

GDCs publish METRICS messages to Centrifuge:

```json
{
  "timestamp": 1709727296,
  "host": "gdc1next",
  "type": "METRICS",
  "value": {
    "events": 1234,
    "bytes": 567890,
    "errors": 0,
    "avgRate": 12.5,
    "currentRate": 13.2,
    "avgByteRate": 0.57,
    "currentByteRate": 0.61
  },
  "runNumber": 12345
}
```

### Aggregation Logic

The rate reader:

1. **Subscribes** to Centrifuge "duck" channel
2. **Filters** METRICS messages from GDCs
3. **Aggregates** data from all GDCs:
   - `totalEvents` = sum of all GDC event counts
   - `totalBytes` = sum of all GDC byte counts
   - `triggerRate` = sum of all GDC current rates
   - `byteRate` = sum of all GDC byte rates
4. **Writes** aggregated data to database every N seconds

## Example Output

### Database Records

```sql
SELECT * FROM rates ORDER BY timestamp DESC LIMIT 10;

+---------------------+------------+-------------+-----------+-------------+
| timestamp           | totalBytes | totalEvents | byteRate  | triggerRate |
+---------------------+------------+-------------+-----------+-------------+
| 2025-03-06 12:34:55 | 125678900  | 245678      | 25.3      | 48.2        |
| 2025-03-06 12:34:50 | 125543200  | 245321      | 24.8      | 47.9        |
| 2025-03-06 12:34:45 | 125407500  | 244965      | 25.1      | 48.1        |
+---------------------+------------+-------------+-----------+-------------+
```

### Log Output (Verbose Mode)

```
2025-03-06 12:34:50 [INFO] Connected to Centrifuge at ws://localhost:8000
2025-03-06 12:34:50 [INFO] Subscribed to channel: duck
2025-03-06 12:34:55 [INFO] Aggregated metrics: events=245678, bytes=125678900
2025-03-06 12:34:55 [INFO] Wrote rates to database: rate=48.2 Hz, byteRate=25.3 MB/s
```

## Monitoring

The rate reader itself doesn't expose Prometheus metrics, but you can monitor:

- **Database writes**: Check `rates` table for recent timestamps
- **Centrifuge connection**: Check logs for connection status
- **Update frequency**: Verify timestamps are ~5 seconds apart

### SQL Queries

```sql
-- Check latest rates
SELECT * FROM rates ORDER BY timestamp DESC LIMIT 1;

-- Get average rate over last hour
SELECT 
    AVG(triggerRate) as avg_trigger_rate,
    AVG(byteRate) as avg_byte_rate
FROM rates
WHERE timestamp > NOW() - INTERVAL 1 HOUR;

-- Get rate history for plotting
SELECT 
    timestamp,
    triggerRate,
    byteRate
FROM rates
WHERE timestamp > NOW() - INTERVAL 10 MINUTE
ORDER BY timestamp;
```

## Use Cases

### Real-time Monitoring Dashboard

The web GUI queries the `rates` table to display:
- Live trigger rate plot
- Data rate plot
- Total event count
- Total data volume

### Performance Analysis

```sql
-- Peak rates during a run
SELECT MAX(triggerRate), MAX(byteRate) FROM rates 
WHERE timestamp BETWEEN '2025-03-06 10:00:00' AND '2025-03-06 11:00:00';
```

### System Health

```sql
-- Check if rate reader is active
SELECT COUNT(*) FROM rates WHERE timestamp > NOW() - INTERVAL 1 MINUTE;
-- Should return 12+ rows (5-second intervals)
```

## Error Handling

- **Centrifuge disconnect**: Automatically reconnects with exponential backoff
- **Database write failure**: Logs error and continues (doesn't crash)
- **Missing GDC metrics**: Aggregates available data, logs warning

## Performance

- **CPU**: <1% (minimal processing)
- **Memory**: ~10 MB (message buffering)
- **Network**: ~1 KB/s (WebSocket messages)
- **Database**: 1 write every 5 seconds

## Building

```bash
# Build binary
go build -o bin/rateReader ./rateReader

# Or use project build system
task build:rateReader
```

## Deployment

### Docker

```bash
docker run -d \
  --name rate-reader \
  -e DB_HOST=mysql \
  -e CENTRIFUGO_HOST=centrifugo \
  next-duck/rate-reader:latest \
  -period 5 -verbose
```

### Systemd

```ini
[Unit]
Description=NEXT-100 Rate Reader
After=network.target mysql.service

[Service]
Type=simple
User=next
ExecStart=/opt/next/bin/rateReader -config /opt/next/config/rateReader.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## Files

| File | Purpose |
|------|---------|
| `main.go` | Main service implementation |

## Dependencies

- Go 1.24+
- `github.com/centrifugal/centrifuge-go` - WebSocket client
- `github.com/go-sql-driver/mysql` - MySQL driver
- `pkg/` - Shared packages (config, centrifuge, logging)

## Related

- `gdcRPC/` - Publishes METRICS messages
- `api/` - Web API serves rate data
- `gui/` - Displays rate plots
- `database/` - Schema includes rates table
