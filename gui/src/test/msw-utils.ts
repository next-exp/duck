import { http, HttpResponse } from 'msw'
import { server } from '../mocks/server'
import { mockGDCs, mockLDCs } from '@/test/fixtures/stores'

// Utility functions for MSW testing

/**
 * Simulate API delay for testing loading states
 */
export const createDelayHandler = (delay: number) => {
  return async () => {
    await new Promise(resolve => setTimeout(resolve, delay))
    return HttpResponse.json({ message: 'Delayed response' })
  }
}

/**
 * Create error handler for testing error states
 */
export const createErrorHandler = (status: number, message: string) => {
  return () => new HttpResponse(message, {
    status,
    headers: { 'Content-Type': 'text/plain' }
  })
}

/**
 * Create network error handler
 */
export const createNetworkErrorHandler = () => {
  return () => {
    throw new Error('Network connection failed')
  }
}

/**
 * Override a specific endpoint handler
 */
export const overrideHandler = (method: 'get' | 'post' | 'put' | 'delete', path: string, handler: any) => {
  server.use(
    http[method](path, handler)
  )
}

/**
 * Mock specific API responses
 */
export const mockApiResponse = {
  // GDC mocks
  gdc: {
    getGDCs: (gdcs: any[]) =>
      overrideHandler('get', 'http://localhost:1323/gdc', () => HttpResponse.json(gdcs)),

    getGDCById: (id: number, gdc: any) =>
      overrideHandler('get', `http://localhost:1323/gdc/${id}`, () => HttpResponse.json(gdc)),

    createGDC: (gdc: any) =>
      overrideHandler('post', 'http://localhost:1323/gdc', () => HttpResponse.json(gdc, { status: 201 })),

    updateGDC: (id: number, gdc: any) =>
      overrideHandler('put', `http://localhost:1323/gdc/${id}`, () => HttpResponse.json(gdc)),

    deleteGDC: (id: number) =>
      overrideHandler('delete', `http://localhost:1323/gdc/${id}`, () => new HttpResponse(null, { status: 204 })),

    gdcError: () =>
      overrideHandler('get', 'http://localhost:1323/gdc', createErrorHandler(500, 'Internal Server Error')),

    slowResponse: (delay: number) =>
      overrideHandler('get', 'http://localhost:1323/gdc', async () => {
        await new Promise(resolve => setTimeout(resolve, delay))
        return HttpResponse.json(mockGDCs)
      })
  },

  // LDC mocks
  ldc: {
    getLDCs: (ldcs: any[]) =>
      overrideHandler('get', 'http://localhost:1323/ldc', () => HttpResponse.json(ldcs)),

    getLDCById: (id: number, ldc: any) =>
      overrideHandler('get', `http://localhost:1323/ldc/${id}`, () => HttpResponse.json(ldc)),

    createLDC: (ldc: any) =>
      overrideHandler('post', 'http://localhost:1323/ldc', () => HttpResponse.json(ldc, { status: 201 })),

    updateLDC: (id: number, ldc: any) =>
      overrideHandler('put', `http://localhost:1323/ldc/${id}`, () => HttpResponse.json(ldc)),

    deleteLDC: (id: number) =>
      overrideHandler('delete', `http://localhost:1323/ldc/${id}`, () => new HttpResponse(null, { status: 204 })),

    ldcError: () =>
      overrideHandler('get', 'http://localhost:1323/ldc', createErrorHandler(500, 'Internal Server Error')),

    slowResponse: (delay: number) =>
      overrideHandler('get', 'http://localhost:1323/ldc', async () => {
        await new Promise(resolve => setTimeout(resolve, delay))
        return HttpResponse.json(mockLDCs)
      })
  },

  // Equipment mocks
  equipment: {
    getEquipments: (equipments: any[]) =>
      overrideHandler('get', 'http://localhost:1323/equipment', () => HttpResponse.json(equipments)),

    getEquipmentById: (id: number, equipment: any) =>
      overrideHandler('get', `http://localhost:1323/equipment/${id}`, () => HttpResponse.json(equipment)),

    createEquipment: (equipment: any) =>
      overrideHandler('post', 'http://localhost:1323/equipment', () => HttpResponse.json(equipment, { status: 201 })),

    updateEquipment: (id: number, equipment: any) =>
      overrideHandler('put', `http://localhost:1323/equipment/${id}`, () => HttpResponse.json(equipment)),

    deleteEquipment: (id: number) =>
      overrideHandler('delete', `http://localhost:1323/equipment/${id}`, () => new HttpResponse(null, { status: 204 })),

    equipmentError: () =>
      overrideHandler('get', 'http://localhost:1323/equipment', createErrorHandler(500, 'Internal Server Error'))
  },

  // Run control mocks
  runControl: {
    startRun: (runNumber: number) =>
      overrideHandler('post', 'http://localhost:1323/start', () => HttpResponse.json({
        success: true,
        runNumber,
        message: 'Run started successfully'
      })),

    stopRun: () =>
      overrideHandler('post', 'http://localhost:1323/stop', () => HttpResponse.json({
        success: true,
        message: 'Run stopped successfully'
      })),

    forceStopRun: () =>
      overrideHandler('post', 'http://localhost:1323/force-stop', () => HttpResponse.json({
        success: true,
        message: 'Run force stopped successfully'
      })),

    restartServices: () =>
      overrideHandler('post', 'http://localhost:1323/restart-services', () => HttpResponse.json({
        success: true,
        message: 'Services restarted successfully'
      })),

    getRunNumber: (runNumber: number) =>
      overrideHandler('get', 'http://localhost:1323/run_number', () => HttpResponse.json({ runNumber })),

    startRunError: () =>
      overrideHandler('post', 'http://localhost:1323/start', createErrorHandler(500, 'Failed to start run'))
  },

  // Statistics mocks
  statistics: {
    getStatistics: (stats: any) =>
      overrideHandler('get', 'http://localhost:1323/statistics', () => HttpResponse.json(stats)),

    statisticsError: () =>
      overrideHandler('get', 'http://localhost:1323/statistics', createErrorHandler(500, 'Failed to fetch statistics'))
  },

  // Common error scenarios
  errors: {
    networkError: (path: string) =>
      overrideHandler('get', `http://localhost:1323${path}`, createNetworkErrorHandler()),

    timeout: (path: string) =>
      overrideHandler('get', `http://localhost:1323${path}`, createDelayHandler(10000)),

    notFound: (path: string) =>
      overrideHandler('get', `http://localhost:1323${path}`, createErrorHandler(404, 'Not Found')),

    unauthorized: (path: string) =>
      overrideHandler('get', `http://localhost:1323${path}`, createErrorHandler(401, 'Unauthorized')),

    forbidden: (path: string) =>
      overrideHandler('get', `http://localhost:1323${path}`, createErrorHandler(403, 'Forbidden')),

    serverError: (path: string, status: number = 500, message: string = 'Internal Server Error') =>
      overrideHandler('get', `http://localhost:1323${path}`, createErrorHandler(status, message))
  }
}

/**
 * Helper to create custom handlers for testing
 */
export const createCustomHandler = {
  success: (data: any, status = 200) => () => HttpResponse.json(data, { status }),
  error: (message: string, status = 500) => createErrorHandler(status, message),
  delay: (data: any, delay: number) => async () => {
    await new Promise(resolve => setTimeout(resolve, delay))
    return HttpResponse.json(data)
  },
  conditional: (condition: () => boolean, successHandler: any, errorHandler: any) => () =>
    condition() ? successHandler() : errorHandler()
}

/**
 * Test scenarios helpers
 */
export const testScenarios = {
  // Loading states
  slowNetwork: (delay = 2000) => {
    server.use(
      http.get('http://localhost:1323/*', createDelayHandler(delay)),
      http.post('http://localhost:1323/*', createDelayHandler(delay)),
      http.put('http://localhost:1323/*', createDelayHandler(delay)),
      http.delete('http://localhost:1323/*', createDelayHandler(delay))
    )
  },

  // All endpoints failing
  globalError: (status = 500, message = 'Internal Server Error') => {
    server.use(
      http.get('http://localhost:1323/*', createErrorHandler(status, message)),
      http.post('http://localhost:1323/*', createErrorHandler(status, message)),
      http.put('http://localhost:1323/*', createErrorHandler(status, message)),
      http.delete('http://localhost:1323/*', createErrorHandler(status, message))
    )
  },

  // Network offline
  offline: () => {
    server.use(
      http.get('http://localhost:1323/*', createNetworkErrorHandler()),
      http.post('http://localhost:1323/*', createNetworkErrorHandler()),
      http.put('http://localhost:1323/*', createNetworkErrorHandler()),
      http.delete('http://localhost:1323/*', createNetworkErrorHandler())
    )
  },

  // Intermittent failures
  intermittentFailure: (failureRate = 0.5) => {
    server.use(
      http.get('http://localhost:1323/*', () => {
        if (Math.random() < failureRate) {
          return createErrorHandler(500, 'Intermittent failure')()
        }
        // Fall through to default handlers
      })
    )
  }
}

/**
 * Request logging for debugging
 */
export const logRequests = () => {
  console.log('Registered handlers:', server.listHandlers())
}

/**
 * Reset all handlers to defaults
 */
export const resetHandlers = () => {
  server.resetHandlers()
}