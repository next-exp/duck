# NEXT-100 DAQ Web Interface

Vue.js 3 web interface for controlling and monitoring the NEXT-100 Data Acquisition System.

## Overview

The GUI provides a real-time interface for:
- Starting and stopping data acquisition runs
- Configuring LDCs, GDCs, and equipment
- Monitoring system status and metrics
- Viewing run statistics and plots
- Managing test devices and simulators

## Technology Stack

- **Vue.js 3** with Composition API
- **TypeScript** for type safety
- **Vite** for fast development and building
- **Tailwind CSS** + **DaisyUI** for styling
- **ConnectRPC Web** for API communication
- **Centrifuge JS** for real-time WebSocket updates
- **Vitest** + **Playwright** for testing

## Development

### Prerequisites

- Node.js 18+
- npm 9+

### Setup

```bash
# Install dependencies
npm install

# Start development server
npm run dev
```

The GUI will be available at http://localhost:5173 with hot module replacement.

### Build for Production

```bash
# Type-check and build
npm run build

# Preview production build
npm run preview
```

## Project Structure

```
gui/
├── src/
│   ├── components/          # Vue components
│   │   ├── ControlPanel.vue    # Run control interface
│   │   ├── EquipmentTable.vue  # Equipment management
│   │   ├── MetricsPlot.vue     # Real-time plots
│   │   └── ...
│   ├── generated/           # Generated TypeScript from protobuf
│   │   ├── api_connect.ts      # ConnectRPC client
│   │   └── api_pb.ts           # Protocol messages
│   ├── stores/              # Pinia stores (state management)
│   ├── composables/         # Vue composition functions
│   ├── router/              # Vue Router configuration
│   ├── App.vue              # Root component
│   └── main.ts              # Application entry point
├── e2e/                     # Playwright E2E tests
├── e2e-integration/         # Integration tests with real backend
├── public/                  # Static assets
├── index.html               # HTML template
├── vite.config.ts           # Vite configuration
├── tsconfig.json            # TypeScript configuration
└── package.json             # Dependencies and scripts
```

## Features

### Run Control

- Start/Stop data acquisition
- Monitor run status in real-time
- View current run number
- Emergency stop (force stop)

### Configuration Management

- **GDC Management**: Add, edit, delete GDC configurations
- **LDC Management**: Add, edit, delete LDC configurations
- **Equipment Management**: Configure ATCA boards
- **Decoder Settings**: Adjust decoder parameters

### Real-time Monitoring

- Live event rate plots
- Data rate graphs
- System state indicators
- Error notifications
- Process health status

### Test Devices

- Start/stop test data generation
- Configure test device parameters
- Monitor test device statistics

## API Integration

### ConnectRPC Client

The GUI uses ConnectRPC to communicate with the API server:

```typescript
import { createPromiseClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { DuckAPI } from "./generated/api_connect";

const transport = createConnectTransport({
  baseUrl: "http://localhost:1323",
});

const client = createPromiseClient(DuckAPI, transport);

// Start a run
const response = await client.startRun({});
console.log(`Started run ${response.runNumber}`);

// Get GDCs
const gdcs = await client.getGDCs({});
```

### Centrifuge WebSocket

Real-time updates via Centrifuge:

```typescript
import Centrifuge from "centrifuge";

const centrifuge = new Centrifuge("ws://localhost:8000/connection/websocket", {
  token: "your-jwt-token",
});

centrifuge.subscribe("duck", (message) => {
  console.log("Received:", message.data);
  // Update UI based on message type
});

centrifuge.connect();
```

### Message Types

The GUI handles these message types from Centrifuge:

- **LOG**: General log messages
- **ERROR**: Error notifications
- **STATE**: State transitions (INITIALIZED → RUNNING)
- **METRICS**: Performance metrics (event rate, data rate)
- **FILE**: File operations
- **SUMMARY**: Run summaries

## Components

### ControlPanel.vue

Main run control interface with:
- Start/Stop buttons
- Current run number display
- System state indicator
- Process status grid

### EquipmentTable.vue

Equipment management with:
- Sortable table
- Inline editing
- Enable/disable toggles
- Add/delete operations

### MetricsPlot.vue

Real-time metrics visualization:
- Event rate plot (using Chart.js or D3)
- Data rate plot
- Auto-updating every 5 seconds
- Configurable time window

### ConfigurationForms

For LDCs, GDCs, Equipment:
- Form validation
- Dynamic fields
- Save/cancel operations

## Testing

### Unit Tests (Vitest)

```bash
# Run unit tests
npm run test:unit

# Run with coverage
npm run test:coverage

# Watch mode
npm run test:unit:watch
```

### E2E Tests (Playwright)

```bash
# Run E2E tests in Docker
npm run docker:test

# Run specific browser
npm run docker:test:chromium
npm run docker:test:firefox

# Run with UI
npm run docker:test:ui
```

### Integration Tests

```bash
# Run against full stack
npm run docker:test:integration:full
```

## Configuration

### Environment Variables

```bash
# API server URL
VITE_API_URL=http://localhost:1323

# Centrifuge WebSocket URL
VITE_CENTRIFUGE_URL=ws://localhost:8000/connection/websocket
```

### Vite Configuration

```typescript
// vite.config.ts
export default defineConfig({
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:1323',
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
});
```

## Docker

### Development

```bash
# Run with full E2E stack
cd ..
task e2e:up:dev

# GUI available at http://localhost:5173
```

### Production Build

```bash
# Build Docker image
docker build -f gui/Dockerfile -t next-duck-gui:latest gui/

# Run container
docker run -p 80:80 next-duck-gui:latest
```

## Styling

### Tailwind CSS + DaisyUI

Components use DaisyUI for consistent styling:

```vue
<button class="btn btn-primary">Start Run</button>
<table class="table table-zebra">
  <!-- ... -->
</table>
<div class="alert alert-error">Error message</div>
```

### Custom Themes

Themes configured in `tailwind.config.js`:

```javascript
module.exports = {
  content: ["./src/**/*.{vue,js,ts}"],
  theme: {
    extend: {
      colors: {
        'next-primary': '#...',
        'next-secondary': '#...',
      },
    },
  },
  plugins: [require("daisyui")],
};
```

## Performance

- **Bundle size**: ~500 KB (gzipped)
- **First contentful paint**: <1s
- **Time to interactive**: <2s
- **Update latency**: <100ms (WebSocket)

## Browser Support

- Chrome/Chromium 90+
- Firefox 90+
- Safari 14+
- Edge 90+

## IDE Setup

### VSCode (Recommended)

1. Install [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar)
2. Disable Vetur
3. Install [TypeScript Vue Plugin (Volar)](https://marketplace.visualstudio.com/items?itemName=Vue.vscode-typescript-vue-plugin)

### TypeScript Support

For `.vue` imports in TS:

```json
// tsconfig.json
{
  "compilerOptions": {
    "types": ["vite/client"]
  }
}
```

## Development Workflow

### Typical Session

```bash
# 1. Start full E2E stack
task e2e:up:dev

# 2. Open GUI
open http://localhost:5173

# 3. Make changes to Vue components
# Hot reload automatically updates UI

# 4. Check console for errors
# Check network tab for API calls

# 5. Run tests
npm run test:unit
npm run docker:test
```

### Debugging

```typescript
// Use Vue DevTools
// Available in Chrome/Firefox extensions

// Debug in component
<script setup lang="ts">
const state = ref('INITIALIZED');
console.log('Current state:', state.value);
debugger; // Browser will pause here
</script>
```

## Scripts

| Script | Description |
|--------|-------------|
| `dev` | Start development server |
| `build` | Build for production |
| `preview` | Preview production build |
| `test:unit` | Run unit tests |
| `test:coverage` | Run tests with coverage |
| `type-check` | Run TypeScript type checking |
| `lint` | Run ESLint |
| `docker:build` | Build Playwright Docker image |
| `docker:test` | Run E2E tests in Docker |

## Dependencies

### Production

- `vue` - Vue.js 3 framework
- `vue-router` - Routing
- `pinia` - State management
- `@connectrpc/connect` - ConnectRPC client
- `@connectrpc/connect-web` - Web transport
- `centrifuge` - WebSocket client
- `chart.js` or `d3` - Plotting

### Development

- `typescript` - TypeScript compiler
- `vite` - Build tool
- `vitest` - Unit testing
- `@playwright/test` - E2E testing
- `tailwindcss` - CSS framework
- `daisyui` - UI components
- `eslint` - Linting
- `prettier` - Code formatting

## Troubleshooting

### "Failed to connect to API"

- Check API server is running: `curl http://localhost:1323`
- Verify VITE_API_URL in `.env`
- Check CORS settings in API server

### "WebSocket connection failed"

- Verify Centrifuge is running: `docker ps | grep centrifugo`
- Check JWT token is valid
- Verify VITE_CENTRIFUGE_URL

### "Type errors in generated code"

```bash
# Regenerate TypeScript from protobuf
cd ..
task gen:api
```

### Hot Reload Not Working

```bash
# Clear Vite cache
rm -rf node_modules/.vite

# Restart dev server
npm run dev
```

## Contributing

1. Follow existing component patterns
2. Use TypeScript for all new code
3. Write unit tests for utilities
4. Write E2E tests for user flows
5. Follow Vue.js style guide
6. Use Prettier for formatting

## Related

- `api/` - Backend API server
- `rpc/proto/api/` - API protocol definitions
- `pkg/centrifugal` - Centrifuge client
- `docker/e2e/` - E2E testing setup

## Resources

- [Vue.js 3 Documentation](https://vuejs.org/)
- [Vite Documentation](https://vitejs.dev/)
- [ConnectRPC Web](https://connectrpc.com/docs/web)
- [Centrifuge JS](https://github.com/centrifugal/centrifuge-js)
- [DaisyUI Components](https://daisyui.com/components/)
- [Vitest](https://vitest.dev/)
- [Playwright](https://playwright.dev/)
