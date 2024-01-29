# Running Playwright Tests in Docker (Fedora)

This document explains how to run Playwright E2E tests on Fedora using Docker, bypassing the OS compatibility issues.

## Why Docker?

Playwright has limited support for Fedora Linux due to:
- Different library paths (glibc versions)
- Missing system dependencies
- RPM package manager differences

Docker solves this by running tests in a controlled Ubuntu environment.

## Quick Start

### 1. Build the Docker Image

```bash
cd /home/jmbenlloch/next/duck-e2e/gui
docker build -f Dockerfile.playwright -t duck-playwright:latest .
```

This builds a Docker image with:
- Playwright browsers installed
- All Node.js dependencies
- Built GUI application

### 2. Run Tests

#### Run all mocked E2E tests (fast)

```bash
npm run docker:test
```

#### Run specific browser

```bash
npm run docker:test:chromium
npm run docker:test:firefox
npm run docker:test:webkit
```

#### Run with UI mode (for debugging)

```bash
npm run docker:test:ui
```

Then open http://localhost:9323 in your browser.

#### Run with browser visible

```bash
npm run docker:test:headed
```

#### Run integration tests (with real backend)

```bash
npm run docker:test:integration
```

This starts:
- MySQL database
- Centrifugo WebSocket server
- API backend
- Playwright tests

### 3. Interactive Shell

To enter the container for debugging:

```bash
npm run docker:shell
```

Inside the container:
```bash
npx playwright test --debug
npx playwright test --ui
```

## Docker Compose Options

### View test results

Test results are mounted to the host:

```bash
# View HTML report
firefox playwright-report/index.html

# Or use the Playwright report viewer
npx playwright show-report playwright-report
```

### Run with custom Playwright options

```bash
docker compose -f docker-compose.playwright.yml run --rm playwright \
  npx playwright test e2e/decoder-form.spec.ts --reporter=line
```

### Run specific test file

```bash
docker compose -f docker-compose.playwright.yml run --rm playwright \
  npx playwright test e2e/decoder-form.spec.ts
```

## Environment Variables

Available in `docker-compose.playwright.yml`:

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_SERVER` | `http://api:1323/daq` | API URL (for integration tests) |
| `VITE_WS_SERVER` | `ws://centrifugo:8000` | WebSocket URL |
| `CI` | `true` | CI mode (retries, etc.) |

## Troubleshooting

### Docker daemon not running

```bash
sudo systemctl start docker
sudo systemctl enable docker
```

### Permission denied accessing Docker

Add your user to the docker group:

```bash
sudo usermod -aG docker $USER
newgrp docker
```

Or use sudo:
```bash
sudo docker compose -f docker-compose.playwright.yml run --rm playwright
```

### Port already in use

Check what's using the port:
```bash
sudo lsof -i :9323  # UI mode port
```

Or use different ports in `docker-compose.playwright.yml`.

### Build fails with "no matching manifest"

The Playwright image uses Ubuntu jammy (22.04). Ensure your Docker supports multi-platform:

```bash
docker buildx build --platform linux/amd64 -f Dockerfile.playwright -t duck-playwright:latest .
```

### Tests fail with "ECONNREFUSED"

For integration tests, ensure backend services are healthy:

```bash
docker compose -f docker-compose.playwright.yml --profile integration ps
```

All services should show "(healthy)" status.

## CI/CD Integration

For CI/CD systems (GitHub Actions, GitLab CI, etc.):

```yaml
- name: Run E2E tests in Docker
  run: |
    cd gui
    docker compose -f docker-compose.playwright.yml run --rm playwright
```

For integration tests:

```yaml
- name: Run integration tests
  run: |
    cd gui
    docker compose -f docker-compose.playwright.yml --profile integration up --abort-on-container-exit
    docker compose -f docker-compose.playwright.yml --profile integration down -v
```

## Performance Tips

### Use Docker BuildKit (faster builds)

```bash
export DOCKER_BUILDKIT=1
docker build -f Dockerfile.playwright -t duck-playwright:latest .
```

### Reuse built image

The Dockerfile caches layers. Rebuild only when dependencies change:

```bash
# Only rebuild if package.json changed
docker build -f Dockerfile.playwright -t duck-playwright:latest .
```

### Parallel execution

For faster test runs, use Docker Compose scale:

```bash
docker compose -f docker-compose.playwright.yml run --rm playwright \
  npx playwright test --workers=4
```

## Comparison: Native vs Docker

| Aspect | Native (Fedora) | Docker |
|--------|-----------------|--------|
| Setup | ❌ Not supported | ✅ Works |
| Speed | N/A | ~2-3min (first run) |
| Consistency | N/A | ✅ Identical to CI |
| Debugging | N/A | Requires port mapping |
| Resource usage | N/A | Higher |

## Advanced: Custom Dockerfile

For custom configurations, modify `Dockerfile.playwright`:

```dockerfile
FROM mcr.microsoft.com/playwright:v1.48.2-jammy

# Install additional tools
RUN apt-get update && apt-get install -y \
    vim \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
# ... rest of Dockerfile
```
