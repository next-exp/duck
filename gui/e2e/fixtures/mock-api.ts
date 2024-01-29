import { test as base, Route, Page, APIRequestContext } from '@playwright/test'

const mockGDCs = [
  { id: 1, name: 'gdc1', hostname: 'gdc1-host', ip: '192.168.1.10', port: 6005, datapath: '/data/gdc1', enabled: true, writeOutput: true, decode: false, grpcPort: 50051, prometheusPort: 9090 },
  { id: 2, name: 'gdc2', hostname: 'gdc2-host', ip: '192.168.1.11', port: 6006, datapath: '/data/gdc2', enabled: false, writeOutput: true, decode: true, grpcPort: 50052, prometheusPort: 9091 }
]

const mockLDCs = [
  { id: 1, name: 'ldc1', hostname: 'ldc1-host', ip: '192.168.1.20', enabled: true, equipments: [], grpcPort: 50053, prometheusPort: 9092 },
  { id: 2, name: 'ldc2', hostname: 'ldc2-host', ip: '192.168.1.21', enabled: false, equipments: [], grpcPort: 50054, prometheusPort: 9093 }
]

const mockEquipments = [
  { id: 1, type: 22, deviceIp: '192.168.1.100', hostIp: '192.168.1.20', hostPort: 6006, ldcId: 1, enabled: true },
  { id: 2, type: 23, deviceIp: '192.168.1.101', hostIp: '192.168.1.20', hostPort: 6007, ldcId: 1, enabled: false }
]

const mockDecoderConfig = {
  extTrigger: 0,
  trgCode1: 0,
  trgCode2: 0,
  readPmts: false,
  readSipms: false,
  readTrigger: false,
  splitTrigger: false,
  noDb: false,
  discard: false,
  host: '',
  user: '',
  password: '',
  dbName: '',
  writeData: false,
  useBlosc: false,
  bloscAlgorithm: 'lz4',
  compressionLevel: 5,
  bitShuffle: 'no-shuffle'
}

async function fulfillJson(route: Route, data: object) {
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(data)
  })
}

const webSocketMockScript = `
  globalThis.MockWebSocketClass = class extends EventTarget {
    CONNECTING = 0;
    OPEN = 1;
    CLOSING = 2;
    CLOSED = 3;
    readyState = 0;
    onopen = null;
    onmessage = null;
    onerror = null;
    onclose = null;
    constructor(url) {
      super();
      setTimeout(() => {
        this.readyState = 1;
        this.onopen?.(new Event('open'));
        this.dispatchEvent(new Event('open'));
      }, 10);
    }
    send(data) {}
    close() {
      this.readyState = 3;
      this.onclose?.(new CloseEvent('close'));
      this.dispatchEvent(new CloseEvent('close'));
    }
    _receive(data) {
      this.onmessage?.(new MessageEvent('message', { data }));
      this.dispatchEvent(new MessageEvent('message', { data }));
    }
  };
  globalThis.WebSocket = globalThis.MockWebSocketClass;
  globalThis._mockWebSockets = [];
  const OriginalWS = globalThis.MockWebSocketClass;
  globalThis.MockWebSocketClass = class extends OriginalWS {
    constructor(url) {
      super(url);
      globalThis._mockWebSockets.push(this);
    }
  };
`

export class WebSocketHelper {
  constructor(private page: Page) {}

  async emitMessage(message: any): Promise<void> {
    await this.page.evaluate((msg) => {
      const sockets = (globalThis as any)._mockWebSockets || []
      sockets.forEach((ws: any) => ws._receive(JSON.stringify(msg)))
    }, message)
  }

  async emitStateChange(host: string, state: string): Promise<void> {
    await this.emitMessage({ type: 'state', host, value: JSON.stringify({ state }) })
  }

  async emitMetrics(host: string, metrics: { EventCounter: number; ByteCounter: number; CurrentDataRate?: number; CurrentTrgRate?: number; AvgDataRate?: number; AvgTrgRate?: number }): Promise<void> {
    await this.emitMessage({ type: 'metric', host, value: JSON.stringify({ CurrentDataRate: 0, CurrentTrgRate: 0, AvgDataRate: 0, AvgTrgRate: 0, ...metrics }) })
  }

  async emitSummary(host: string, summary: { events: number; bytes: number; errors?: number }): Promise<void> {
    await this.emitMessage({ type: 'summary', host, value: JSON.stringify({ errors: 0, ...summary }) })
  }

  async emitOutputFile(host: string, subrun: number): Promise<void> {
    await this.emitMessage({ type: 'output', host, value: JSON.stringify({ server: host, subrun }) })
  }
}

export const test = base.extend<{
  mockApi: { overrideRoute: (url: string | RegExp, handler: (route: Route) => Promise<void>) => Promise<void> }
  websocket: WebSocketHelper
}>({
  mockApi: async ({ page }, use) => {
    await page.addInitScript(webSocketMockScript)

    // ConnectRPC endpoints - URLs are /apiService.DuckAPI/MethodName
    await page.route('**/apiService.DuckAPI/GetGDCs', async route => {
      await fulfillJson(route, { gdcs: mockGDCs })
    })

    await page.route('**/apiService.DuckAPI/GetLDCs', async route => {
      await fulfillJson(route, { ldcs: mockLDCs })
    })

    await page.route('**/apiService.DuckAPI/GetEquipments', async route => {
      await fulfillJson(route, { equipments: mockEquipments })
    })

    await page.route('**/apiService.DuckAPI/GetDecoderConfiguration', async route => {
      await fulfillJson(route, { configuration: mockDecoderConfig })
    })

    await page.route('**/apiService.DuckAPI/GetRunNumber', async route => {
      await fulfillJson(route, { runNumber: '12345' })
    })

    await page.route('**/apiService.DuckAPI/GetProcessStates', async route => {
      await fulfillJson(route, {})
    })

    await page.route('**/apiService.DuckAPI/CheckDisabled', async route => {
      await fulfillJson(route, { gdcs: [], ldcs: [], equipments: [], writing: [] })
    })

    await page.route('**/apiService.DuckAPI/GetToken', async route => {
      await fulfillJson(route, { token: 'mock-token' })
    })

    await page.route('**/apiService.DuckAPI/StartRun', async route => {
      await fulfillJson(route, {})
    })

    await page.route('**/apiService.DuckAPI/StopRun', async route => {
      await fulfillJson(route, {})
    })

    await page.route('**/apiService.DuckAPI/ForceStopRun', async route => {
      await fulfillJson(route, {})
    })

    await page.route('**/apiService.DuckAPI/RestartServices', async route => {
      await fulfillJson(route, {})
    })

    await page.route('**/apiService.DuckAPI/CreateGDC', async route => {
      const newGdc = { ...mockGDCs[0], id: 999 }
      await fulfillJson(route, newGdc)
    })

    await page.route('**/apiService.DuckAPI/UpdateGDC', async route => {
      await fulfillJson(route, mockGDCs[0])
    })

    await page.route('**/apiService.DuckAPI/DeleteGDC', async route => {
      await fulfillJson(route, {})
    })

    await page.route('**/apiService.DuckAPI/CreateLDC', async route => {
      const newLdc = { ...mockLDCs[0], id: 999 }
      await fulfillJson(route, newLdc)
    })

    await page.route('**/apiService.DuckAPI/UpdateLDC', async route => {
      await fulfillJson(route, mockLDCs[0])
    })

    await page.route('**/apiService.DuckAPI/DeleteLDC', async route => {
      await fulfillJson(route, {})
    })

    await page.route('**/apiService.DuckAPI/CreateEquipment', async route => {
      const newEq = { ...mockEquipments[0], id: 999 }
      await fulfillJson(route, newEq)
    })

    await page.route('**/apiService.DuckAPI/UpdateEquipment', async route => {
      await fulfillJson(route, mockEquipments[0])
    })

    await page.route('**/apiService.DuckAPI/DeleteEquipment', async route => {
      await fulfillJson(route, {})
    })

    await page.route('**/apiService.DuckAPI/UpdateDecoderConfiguration', async route => {
      await fulfillJson(route, { configuration: mockDecoderConfig })
    })

    // Catch-all for any WebSocket connections
    await page.route('**/connection/websocket', route => route.abort()).catch(() => {})

    await use({
      overrideRoute: async (url, handler) => {
        await page.route(url, handler)
      }
    })
  },

  websocket: async ({ page }, use) => {
    await use(new WebSocketHelper(page))
  }
})

export { expect } from '@playwright/test'
