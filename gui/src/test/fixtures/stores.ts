import { GDC, LDC, Equipment } from '@/gen/api_pb'

// Use the actual Protobuf-generated classes for type safety
export const mockGDCs: GDC[] = [
  Object.assign(new GDC(), {
    id: 1,
    name: 'gdc1',
    hostname: 'gdc1-host',
    ip: '192.168.1.10',
    port: 6005,
    datapath: '/data/gdc1',
    enabled: true,
    writeOutput: true,
    decode: false,
    grpcPort: 50051,
    prometheusPort: 9090
  }),
  Object.assign(new GDC(), {
    id: 2,
    name: 'gdc2',
    hostname: 'gdc2-host',
    ip: '192.168.1.11',
    port: 6006,
    datapath: '/data/gdc2',
    enabled: false,
    writeOutput: true,
    decode: true,
    grpcPort: 50052,
    prometheusPort: 9091
  })
]

export const mockLDCs: LDC[] = [
  Object.assign(new LDC(), {
    id: 1,
    name: 'ldc1',
    hostname: 'ldc1-host',
    ip: '192.168.1.20',
    enabled: true,
    equipments: [
      Object.assign(new Equipment(), {
        id: 1,
        type: 22,
        deviceIp: '192.168.1.100',
        hostIp: '192.168.1.20',
        hostPort: 6006,
        ldcId: 1,
        enabled: true
      })
    ],
    grpcPort: 50053,
    prometheusPort: 9092
  }),
  Object.assign(new LDC(), {
    id: 2,
    name: 'ldc2',
    hostname: 'ldc2-host',
    ip: '192.168.1.21',
    enabled: false,
    equipments: [],
    grpcPort: 50054,
    prometheusPort: 9093
  })
]

export const mockEquipments: Equipment[] = [
  Object.assign(new Equipment(), {
    id: 1,
    type: 22,
    deviceIp: '192.168.1.100',
    hostIp: '192.168.1.20',
    hostPort: 6006,
    ldcId: 1,
    enabled: true
  }),
  Object.assign(new Equipment(), {
    id: 2,
    type: 23,
    deviceIp: '192.168.1.101',
    hostIp: '192.168.1.20',
    hostPort: 6007,
    ldcId: 1,
    enabled: false
  })
]

export const mockWebSocketMessages = {
  metricMessage: {
    host: 'gdc1',
    timestamp: '2025-10-16T11:30:00Z',
    type: 'metric',
    value: JSON.stringify({
      EventCounter: 1000,
      ByteCounter: 5000000,
      CurrentDataRate: 1000,
      CurrentTrgRate: 10,
      AvgDataRate: 950,
      AvgTrgRate: 9.5
    }),
    run: 123
  },
  stateMessage: {
    host: 'ldc1',
    timestamp: '2025-10-16T11:30:00Z',
    type: 'state',
    value: JSON.stringify({
      state: 'RUNNING'
    }),
    run: 123
  },
  errorMessage: {
    host: 'gdc1',
    timestamp: '2025-10-16T11:30:00Z',
    type: 'error',
    value: 'Connection lost to equipment',
    run: 123
  },
  outputMessage: {
    host: 'gdc1',
    timestamp: '2025-10-16T11:30:00Z',
    type: 'output',
    value: JSON.stringify({
      server: 'gdc1',
      subrun: 1
    }),
    run: 123
  },
  summaryMessage: {
    host: 'ldc1',
    timestamp: '2025-10-16T11:30:00Z',
    type: 'summary',
    value: JSON.stringify({
      events: 500,
      bytes: 2500000,
      errors: 2
    }),
    run: 123
  }
}
