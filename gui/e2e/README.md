# E2E Test Documentation

This directory contains Playwright E2E tests for the DUCK GUI application with **full backend mocking** support.

## Two-Tier Testing Strategy

We use a **hybrid approach** to balance speed, reliability, and integration coverage:

```
┌─────────────────────────────────────────────────────┐
│                    CI/CD Pipeline                    │
├─────────────────────────────────────────────────────┤
│  PR Check (Fast - <3 min)                            │
│  ├─ Unit Tests (Vitest)           - 30s              │
│  ├─ E2E Mocked (Playwright + MSW)  - 2min            │
│  └─ Total: Fast feedback ✅                             │
├─────────────────────────────────────────────────────┤
│  Nightly/Pre-merge (Slow - 10-15min)                  │
│  ├─ E2E Integration (Docker)         - Real backend   │
│  └─ Runs on: main branch, release candidates         │
└─────────────────────────────────────────────────────┘
```

### Test Tiers

| Tier | Command | Stack | When to Run | Duration |
|------|---------|-------|-------------|----------|
| **Unit** | `npm run test:unit` | Vitest + jsdom | Every PR | ~30s |
| **E2E Mocked** | `npm run test:e2e:mocked` | Playwright + MSW + Fake WS | Every PR | ~2min |
| **E2E Integration** | `npm run test:e2e:integration` | Docker + Real Backend | Nightly/Merge | ~10min |

## Test Files

### Mocked Tests (`e2e/`)

These run **without any backend** using MSW for API mocking and a fake WebSocket for Centrifuge.

| File | Description |
|------|-------------|
| `basic.spec.ts` | Basic page loading, navigation, accessibility |
| `decoder-form.spec.ts` | Decoder configuration form validation |
| `gdc-form.spec.ts` | GDC management form tests |
| `ldc-equipment-forms.spec.ts` | LDC and Equipment configuration |
| `run-control-statistics.spec.ts` | Run control and statistics display |
| `example-with-fixtures.spec.ts` | Example showing new fixture usage |

### Integration Tests (`e2e-integration/`)

These run against a **real backend** using Docker Compose to validate API contracts and real WebSocket behavior.

| Purpose | Tests |
|---------|-------|
| Contract validation | API schema changes |
| WebSocket protocol | Real Centrifugo message handling |
| Database constraints | Actual data persistence |

## New Fixtures API

The `fixtures/index.ts` provides enhanced test utilities:

### MSW Fixture

```typescript
import { test, expect } from './fixtures'

test('my test', async ({ page, msw }) => {
  // MSW is automatically active with all handlers from src/mocks/handlers.ts

  // Override specific handler for this test
  msw.overrideHandler(
    http.get('http://localhost:1323/gdc', () => {
      return HttpResponse.json({ gdcs: [] })
    })
  )
})
```

### Centrifuge Fixture

```typescript
test('run control test', async ({ page, centrifuge }) => {
  // Simulate server state changes
  await centrifuge.setState('gdc-01', 'INITIALIZED')
  await centrifuge.setState('ldc-01', 'INITIALIZED')

  // Common scenarios
  await centrifuge.setupInitializedState(['gdc-01', 'ldc-01'])
  await centrifuge.startRunScenario(['gdc-01', 'ldc-01', 'ldc-02'])

  // Simulate metrics
  await centrifuge.setMetrics('gdc-01', {
    EventCounter: 50000,
    ByteCounter: 100000000,
    CurrentDataRate: 50000000,
    CurrentTrgRate: 25.5,
    AvgDataRate: 48000000,
    AvgTrgRate: 24.2
  })
})
```

## Running Tests

### Development (Fast Feedback)

```bash
# Run mocked E2E tests (no backend needed)
npm run test:e2e:mocked

# Run with UI mode for debugging
npm run test:e2e:ui

# Run single test file
npx playwright test e2e/decoder-form.spec.ts

# Run with browser visible
npm run test:e2e:headed
```

### Integration (Real Backend)

```bash
# Run integration tests against real Docker stack
npm run test:e2e:integration
```

### Full Test Suite

```bash
# Run unit + mocked E2E (fast - for PRs)
npm run test

# Run everything including integration (slow - for nightlies)
npm run test:all
```

## Test Structure

### Using the Fixtures

```typescript
import { test, expect } from './fixtures'

test.describe('Feature Tests', () => {
  test('should work with mocked backend', async ({ page, msw, centrifuge }) => {
    await page.goto('/daq/')

    // MSW automatically handles API calls
    // Centrifuge simulates WebSocket messages

    await centrifuge.setState('gdc-01', 'RUNNING')

    // Use data-testids for robust selectors
    await expect(page.locator('[data-testid="gdc-name-input-1"]')).toBeVisible()
  })
})
```

## Best Practices

### ✅ Do

- Use **data-testids** instead of CSS selectors: `[data-testid="submit-button"]`
- Wait for **specific conditions**: `await expect(locator).toBeVisible()`
- Use **Centrifuge fixture** for state changes
- Use **MSW handlers** for API responses
- Test **user flows**, not implementation details

### ❌ Don't

- Use `force: true` on clicks - fix the actual issue
- Use `waitForTimeout` - wait for specific state instead
- Use brittle selectors like `input[id*="name"]`
- Test framework functionality (404, basic navigation)

## Coverage

Current E2E test coverage:

| Feature | Tests | Status |
|---------|-------|--------|
| Decoder Configuration | 8 tests | ✅ Mocked |
| GDC Management | 7 tests | ✅ Mocked |
| LDC Management | 7 tests | ✅ Mocked |
| Equipment Management | 7 tests | ✅ Mocked |
| Run Control | 10 tests | ✅ Mocked |
| Run Statistics | 8 tests | ✅ Mocked |
| Basic/Accessibility | 11 tests | ✅ Mocked |

**Total: ~58 tests**

## Why Mocking Works

### MSW (Mock Service Worker)

- Intercepts HTTP requests at the network level
- Works with ConnectRPC (gRPC-Web) protocol
- Handlers defined in `src/mocks/handlers.ts`
- Shared between unit and E2E tests

### Fake WebSocket

- Injects mock `WebSocket` class via `page.addInitScript()`
- Simulates Centrifuge protocol messages
- No external server needed
- Full control over message timing and content

## Maintenance

### Adding New Tests

1. Use the fixtures: `import { test, expect } from './fixtures'`
2. Add data-testids to components if needed
3. Update MSW handlers for new endpoints
4. Add Centrifuge helpers for new message types

### Updating API Handlers

Edit `src/mocks/handlers.ts` - changes apply to both unit and E2E tests.

### Debugging

```bash
# Run with Playwright Inspector
npm run test:e2e:debug

# Run with UI mode
npm run test:e2e:ui

# Run with browser visible
npm run test:e2e:headed
```

## Troubleshooting

### Tests fail with "Cannot find module"

Run `npm install` to ensure MSW and other dependencies are installed.

### WebSocket messages not received

Ensure you're using the `websocket` fixture from `./fixtures` which mocks WebSocket connections.

### API calls not mocked

Check that MSW handlers are defined in `src/mocks/handlers.ts` for the endpoint.
