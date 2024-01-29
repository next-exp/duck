# Integration E2E Tests

These are **slow** integration tests that run against a **real backend** using Docker Compose.

## Purpose

Integration tests validate:
- **API contracts** - Real HTTP requests/responses with the actual Go backend
- **WebSocket protocol** - Real Centrifugo WebSocket message handling
- **Database constraints** - Actual database schema and constraints
- **End-to-end workflows** - Complete user journeys through the system

## When to Run

| Situation | Command |
|-----------|---------|
| Nightly/CI pipeline | `npm run test:e2e:integration` |
| Before major release | `npm run test:e2e:integration` |
| After API contract changes | `npm run test:e2e:integration` |
| Local development | Rarely (use `npm run test:e2e` instead) |

## Prerequisites

Integration tests run entirely in Docker. Ensure you have the necessary Docker images built.

### 1. Build Docker Images

```bash
cd /path/to/duck  # Go to repository root
docker build -t duck-e2e:latest -f docker/e2e/Dockerfile .
cd gui
docker build -t duck-playwright:latest -f Dockerfile.playwright .
```

## Running Tests

### Run All Integration Tests (Docker-based)

```bash
# Run everything in Docker - builds GUI, starts backends, runs tests, cleans up
npm run docker:test:integration:full
```

### Run Tests with More Control

```bash
# Start all services (backends + test runner)
docker compose -f docker-compose.playwright.yml --profile integration up --abort-on-container-exit

# Clean up afterwards
docker compose -f docker-compose.playwright.yml --profile integration down -v
```

### Run Specific Browser

```bash
# Run integration tests with Chromium
docker compose -f docker-compose.playwright.yml --profile integration run --rm playwright npx playwright test --config=playwright.integration.config.ts --project=chromium
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `E2E_API_URL` | `http://localhost:1323/daq` | Backend API URL |
| `E2E_WS_URL` | `ws://localhost:8000` | WebSocket server URL |

## Test Structure

```
e2e-integration/
├── integration.spec.ts    # Main integration tests
└── README.md              # This file
```

### Test Categories

1. **API Contract Validation** (5 tests)
   - Fetch GDCs from real backend
   - Fetch LDCs from real backend
   - Fetch decoder configuration
   - Handle API errors gracefully

2. **Real WebSocket/Centrifuge Tests** (3 tests)
   - Connect to real Centrifugo server
   - Receive real state messages
   - Display run statistics from backend

3. **Database Constraints** (2 tests)
   - Enforce unique GDC names
   - Validate port number ranges

4. **End-to-End Workflows** (2 tests)
   - GDC create-update-delete cycle
   - Run control with real backend

5. **Health & Connectivity** (2 tests)
   - Verify API health endpoint
   - Verify WebSocket reachability

**Total: 14 integration tests**

## Troubleshooting

### Tests fail with "ECONNREFUSED"

Backend services may not be running. Ensure Docker services are up:
```bash
docker compose -f docker-compose.playwright.yml --profile integration ps
```

### Tests timeout waiting for data

Services might not be healthy yet. Check with:
```bash
docker compose -f docker-compose.playwright.yml --profile integration ps
```

### Database connection errors

MySQL might still be initializing. Check logs:
```bash
docker logs duck-e2e-mysql
```

### WebSocket connection errors

Centrifugo might not be ready. Check logs:
```bash
docker logs duck-e2e-centrifugo
```

## CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Integration Tests

on:
  schedule:
    - cron: '0 2 * * *'  # Run at 2 AM daily
  workflow_dispatch:      # Allow manual trigger

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    timeout-minutes: 30

    steps:
      - uses: actions/checkout@v3

      - name: Build Docker images
        run: |
          docker build -t duck-e2e:latest -f docker/e2e/Dockerfile .
          cd gui
          docker build -t duck-playwright:latest -f Dockerfile.playwright .

      - name: Run integration tests
        run: |
          cd gui
          docker compose -f docker-compose.playwright.yml --profile integration up --abort-on-container-exit

      - name: Stop services
        if: always()
        run: |
          cd gui
          docker compose -f docker-compose.playwright.yml --profile integration down -v

      - name: Upload test report
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: integration-test-report
          path: gui/playwright-report/
```

## Comparison: Mocked vs Integration

| Aspect | Mocked E2E | Integration E2E |
|--------|------------|-----------------|
| **Backend** | MSW (mocked) | Real Docker |
| **Speed** | ~2 minutes | ~10-15 minutes |
| **Flakiness** | Low | Medium |
| **Infrastructure** | None | Docker required |
| **Run Frequency** | Every PR | Nightly/Merge |
| **Validates** | UI/UX | API contracts + DB |

## Best Practices

1. **Keep integration tests minimal** - Only test what requires a real backend
2. **Use mocked tests for UI logic** - Faster and more reliable
3. **Run integration tests in CI** - Not locally (too slow)
4. **Clean up after tests** - Always stop Docker containers
5. **Monitor test duration** - If >15min, consider splitting
