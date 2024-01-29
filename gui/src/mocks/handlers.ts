import { http, HttpResponse } from 'msw'
import { mockGDCs, mockLDCs, mockEquipments } from '@/test/fixtures/stores'
import { GDC, LDC, Equipment } from '@/gen/api_pb'

// API response handlers - using camelCase to match ConnectRPC/Protobuf serialization
export const handlers = [
  // GDC endpoints
  http.get('http://localhost:1323/gdc', ({ request }) => {
    const url = new URL(request.url)
    const searchParams = url.searchParams

    // Handle filtering/searching if needed
    let filteredGDCs = [...mockGDCs]

    if (searchParams.get('enabled') === 'true') {
      filteredGDCs = filteredGDCs.filter(gdc => gdc.enabled)
    }

    return HttpResponse.json({ gdcs: filteredGDCs })
  }),

  http.get('http://localhost:1323/gdc/:id', ({ params }) => {
    const { id } = params
    const gdc = mockGDCs.find(g => g.id === Number(id))

    if (!gdc) {
      return new HttpResponse(null, { status: 404 })
    }

    return HttpResponse.json(gdc)
  }),

  http.post('http://localhost:1323/gdc', async ({ request }) => {
    const newGDC = await request.json() as any
    const createdGDC = Object.assign(new GDC(), {
      id: Math.max(...mockGDCs.map(g => g.id)) + 1,
      name: newGDC.name || `gdc-${Date.now()}`,
      hostname: newGDC.hostname || `gdc-${Date.now()}`,
      ip: newGDC.ip || '192.168.1.100',
      port: newGDC.port || 6005,
      datapath: newGDC.datapath || '/data',
      grpcPort: newGDC.grpcPort || 50051,
      prometheusPort: newGDC.prometheusPort || 9090,
      enabled: newGDC.enabled ?? true,
      writeOutput: newGDC.writeOutput ?? true,
      decode: newGDC.decode ?? false
    })

    return HttpResponse.json(createdGDC, { status: 201 })
  }),

  http.put('http://localhost:1323/gdc/:id', async ({ params, request }) => {
    const { id } = params
    const updateData = await request.json() as any
    const gdcIndex = mockGDCs.findIndex(g => g.id === Number(id))

    if (gdcIndex === -1) {
      return new HttpResponse(null, { status: 404 })
    }

    // Merge with existing GDC, preserving type safety
    const updatedGDC = Object.assign(new GDC(), mockGDCs[gdcIndex], updateData)
    return HttpResponse.json(updatedGDC)
  }),

  http.delete('http://localhost:1323/gdc/:id', ({ params }) => {
    const { id } = params
    const gdcIndex = mockGDCs.findIndex(g => g.id === Number(id))

    if (gdcIndex === -1) {
      return new HttpResponse(null, { status: 404 })
    }

    return new HttpResponse(null, { status: 204 })
  }),

  // LDC endpoints
  http.get('http://localhost:1323/ldc', ({ request }) => {
    const url = new URL(request.url)
    const searchParams = url.searchParams

    let filteredLDCs = [...mockLDCs]

    if (searchParams.get('enabled') === 'true') {
      filteredLDCs = filteredLDCs.filter(ldc => ldc.enabled)
    }

    return HttpResponse.json({ ldcs: filteredLDCs })
  }),

  http.get('http://localhost:1323/ldc/:id', ({ params }) => {
    const { id } = params
    const ldc = mockLDCs.find(l => l.id === Number(id))

    if (!ldc) {
      return new HttpResponse(null, { status: 404 })
    }

    return HttpResponse.json(ldc)
  }),

  http.post('http://localhost:1323/ldc', async ({ request }) => {
    const newLDC = await request.json() as any
    const createdLDC = Object.assign(new LDC(), {
      id: Math.max(...mockLDCs.map(l => l.id)) + 1,
      name: newLDC.name || `ldc-${Date.now()}`,
      hostname: newLDC.hostname || `ldc-${Date.now()}`,
      ip: newLDC.ip || '192.168.1.100',
      grpcPort: newLDC.grpcPort || 50053,
      prometheusPort: newLDC.prometheusPort || 9092,
      enabled: newLDC.enabled ?? true,
      equipments: []
    })

    return HttpResponse.json(createdLDC, { status: 201 })
  }),

  http.put('http://localhost:1323/ldc/:id', async ({ params, request }) => {
    const { id } = params
    const updateData = await request.json() as any
    const ldcIndex = mockLDCs.findIndex(l => l.id === Number(id))

    if (ldcIndex === -1) {
      return new HttpResponse(null, { status: 404 })
    }

    const updatedLDC = Object.assign(new LDC(), mockLDCs[ldcIndex], updateData)
    return HttpResponse.json(updatedLDC)
  }),

  http.delete('http://localhost:1323/ldc/:id', ({ params }) => {
    const { id } = params
    const ldcIndex = mockLDCs.findIndex(l => l.id === Number(id))

    if (ldcIndex === -1) {
      return new HttpResponse(null, { status: 404 })
    }

    return new HttpResponse(null, { status: 204 })
  }),

  // Equipment endpoints
  http.get('http://localhost:1323/equipment', ({ request }) => {
    const url = new URL(request.url)
    const searchParams = url.searchParams

    let filteredEquipments = [...mockEquipments]

    if (searchParams.get('enabled') === 'true') {
      filteredEquipments = filteredEquipments.filter(eq => eq.enabled)
    }

    if (searchParams.get('ldcid')) {
      const ldcId = Number(searchParams.get('ldcid'))
      filteredEquipments = filteredEquipments.filter(eq => eq.ldcId === ldcId)
    }

    return HttpResponse.json({ equipments: filteredEquipments })
  }),

  http.get('http://localhost:1323/equipment/:id', ({ params }) => {
    const { id } = params
    const equipment = mockEquipments.find(e => e.id === Number(id))

    if (!equipment) {
      return new HttpResponse(null, { status: 404 })
    }

    return HttpResponse.json(equipment)
  }),

  http.post('http://localhost:1323/equipment', async ({ request }) => {
    const newEquipment = await request.json() as any
    const createdEquipment = Object.assign(new Equipment(), {
      id: Math.max(...mockEquipments.map(e => e.id)) + 1,
      type: newEquipment.type || 22,
      deviceIp: newEquipment.deviceIp || '192.168.1.100',
      hostIp: newEquipment.hostIp || '192.168.1.20',
      hostPort: newEquipment.hostPort || 6006,
      ldcId: newEquipment.ldcId || 1,
      enabled: newEquipment.enabled ?? true
    })

    return HttpResponse.json(createdEquipment, { status: 201 })
  }),

  http.put('http://localhost:1323/equipment/:id', async ({ params, request }) => {
    const { id } = params
    const updateData = await request.json() as any
    const equipmentIndex = mockEquipments.findIndex(e => e.id === Number(id))

    if (equipmentIndex === -1) {
      return new HttpResponse(null, { status: 404 })
    }

    const updatedEquipment = Object.assign(new Equipment(), mockEquipments[equipmentIndex], updateData)
    return HttpResponse.json(updatedEquipment)
  }),

  http.delete('http://localhost:1323/equipment/:id', ({ params }) => {
    const { id } = params
    const equipmentIndex = mockEquipments.findIndex(e => e.id === Number(id))

    if (equipmentIndex === -1) {
      return new HttpResponse(null, { status: 404 })
    }

    return new HttpResponse(null, { status: 204 })
  }),

  // Decoder endpoints
  http.get('http://localhost:1323/decoder', () => {
    return HttpResponse.json({
      configuration: {
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
    })
  }),

  http.put('http://localhost:1323/decoder', async ({ request }) => {
    const updateData = await request.json() as any
    return HttpResponse.json({ configuration: updateData.configuration })
  }),

  // Run control endpoints
  http.post('http://localhost:1323/start', async ({ request }) => {
    const runData = await request.json() as any
    // Simulate starting a run
    return HttpResponse.json({
      success: true,
      runNumber: runData.runNumber || Math.floor(Math.random() * 10000),
      message: 'Run started successfully'
    })
  }),

  http.post('http://localhost:1323/stop', async () => {
    // Simulate stopping a run
    return HttpResponse.json({
      success: true,
      message: 'Run stopped successfully'
    })
  }),

  http.post('http://localhost:1323/force-stop', async () => {
    // Simulate force stopping a run
    return HttpResponse.json({
      success: true,
      message: 'Run force stopped successfully'
    })
  }),

  http.post('http://localhost:1323/restart-services', async () => {
    // Simulate restarting services
    return HttpResponse.json({
      success: true,
      message: 'Services restarted successfully'
    })
  }),

  // Run number endpoint
  http.get('http://localhost:1323/run_number', () => {
    return HttpResponse.json({
      runNumber: Math.floor(Math.random() * 10000)
    })
  }),

  // Statistics endpoints
  http.get('http://localhost:1323/statistics', () => {
    return HttpResponse.json({
      gdcs: [
        {
          name: 'gdc-01',
          state: 'RUNNING',
          events: 1500000,
          bytes: 3000000000,
          evtRateCurrent: 100.5,
          evtRateAvg: 95.2,
          byteRateCurrent: 200000000,
          byteRateAvg: 190000000,
          fileNumber: 42
        }
      ],
      ldcs: [
        {
          name: 'ldc-01',
          state: 'RUNNING',
          events: 750000,
          bytes: 1500000000,
          evtRateCurrent: 50.25,
          evtRateAvg: 47.6,
          byteRateCurrent: 100000000,
          byteRateAvg: 95000000,
          fileNumber: 0
        },
        {
          name: 'ldc-02',
          state: 'IDLE',
          events: 750000,
          bytes: 1500000000,
          evtRateCurrent: 50.25,
          evtRateAvg: 47.6,
          byteRateCurrent: 100000000,
          byteRateAvg: 95000000,
          fileNumber: 0
        }
      ],
      totals: {
        gdc: {
          events: 1500000,
          bytes: 3000000000,
          evtRateCurrent: 100.5,
          evtRateAvg: 95.2,
          byteRateCurrent: 200000000,
          byteRateAvg: 190000000
        },
        ldc: {
          events: 1500000,
          bytes: 3000000000,
          evtRateCurrent: 100.5,
          evtRateAvg: 95.2,
          byteRateCurrent: 200000000,
          byteRateAvg: 190000000
        }
      }
    })
  }),

  // Error simulation endpoints
  http.get('http://localhost:1323/gdc/error', () => {
    return new HttpResponse('Internal Server Error', {
      status: 500,
      headers: { 'Content-Type': 'text/plain' }
    })
  }),

  http.get('http://localhost:1323/ldc/network-error', () => {
    // Simulate network error
    throw new Error('Network connection failed')
  }),

  http.get('http://localhost:1323/equipment/timeout', async () => {
    // Simulate timeout
    await new Promise(resolve => setTimeout(resolve, 10000))
    return HttpResponse.json({ message: 'This should timeout' })
  })
]
