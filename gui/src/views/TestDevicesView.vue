<script setup lang="ts">
import { ref, onMounted, type Ref } from 'vue'
import { duckApiClient } from '@/api/client'
import AlertError from '@/components/AlertError.vue'
import AlertSuccess from '@/components/AlertSuccess.vue'
import AlertWarning from '@/components/AlertWarning.vue'
import { PlayIcon, StopIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'

interface TestDevice {
  equipmentId: number
  deviceName: string
  hostIp: string
  hostPort: number
  grpcPort: number
  prometheusPort: number
  rateHz?: number
  eventRateMs?: number
  packetsPerEvent: number
  packetSize: number
  errorInjectionRate: number
  maxEvents: number
  enabled: boolean
}

interface TestDeviceStates {
  [key: string]: string
}

interface TestDeviceStatistics {
  deviceId: number
  events: bigint
  bytes: bigint
  errors: bigint
}

const devices: Ref<TestDevice[]> = ref([])
const deviceStates: Ref<TestDeviceStates> = ref({})
const deviceStats: Ref<Record<number, TestDeviceStatistics>> = ref({})
const errorMsg: Ref<string> = ref('')
const successMsg: Ref<string> = ref('')
const loading: Ref<boolean> = ref(false)
const enabledControlButtons: Ref<boolean> = ref(false)

async function loadDevices() {
  try {
    const response = await duckApiClient.listTestDevices({})
    devices.value = response.devices || []
    errorMsg.value = ''
  } catch (error: any) {
    console.error('Error loading devices:', error)
    errorMsg.value = error.message || 'Failed to load devices'
  }
}

async function loadDeviceStates() {
  try {
    const response = await duckApiClient.getTestDevicesStates({})
    deviceStates.value = response.states || {}
    errorMsg.value = ''
  } catch (error: any) {
    console.error('Error loading device states:', error)
    errorMsg.value = error.message || 'Failed to load device states'
  }
}

async function loadDeviceStatistics(deviceId: number) {
  try {
    const response = await duckApiClient.getTestDeviceStatistics({ deviceId })
    if (response.statistics) {
      deviceStats.value[deviceId] = {
        deviceId: response.statistics.deviceId,
        events: response.statistics.events,
        bytes: response.statistics.bytes,
        errors: response.statistics.errors
      }
    }
  } catch (error: any) {
    console.error(`Error loading statistics for device ${deviceId}:`, error)
  }
}

async function loadAllStatistics() {
  for (const device of devices.value) {
    await loadDeviceStatistics(device.equipmentId)
  }
}

async function startAllDevices() {
  loading.value = true
  successMsg.value = ''
  errorMsg.value = ''

  try {
    await duckApiClient.startTestDevices({})
    successMsg.value = 'All test devices started successfully'
    setTimeout(() => {
      loadDeviceStates()
      loadAllStatistics()
    }, 500)
  } catch (error: any) {
    console.error('Error starting devices:', error)
    errorMsg.value = error.message || 'Failed to start devices'
  } finally {
    loading.value = false
  }
}

async function stopAllDevices() {
  loading.value = true
  successMsg.value = ''
  errorMsg.value = ''

  try {
    await duckApiClient.stopTestDevices({})
    successMsg.value = 'All test devices stopped successfully'
    setTimeout(() => {
      loadDeviceStates()
      loadAllStatistics()
    }, 500)
  } catch (error: any) {
    console.error('Error stopping devices:', error)
    errorMsg.value = error.message || 'Failed to stop devices'
  } finally {
    loading.value = false
  }
}

async function refreshData() {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    await loadDevices()
    await loadDeviceStates()
    await loadAllStatistics()
    successMsg.value = 'Data refreshed successfully'
    setTimeout(() => {
      successMsg.value = ''
    }, 2000)
  } catch (error: any) {
    console.error('Error refreshing data:', error)
    errorMsg.value = error.message || 'Failed to refresh data'
  } finally {
    loading.value = false
  }
}

function getStateClass(state: string): string {
  switch (state?.toLowerCase()) {
    case 'running':
      return 'badge-success'
    case 'stopped':
    case 'idle':
      return 'badge-neutral'
    case 'error':
      return 'badge-error'
    default:
      return 'badge-ghost'
  }
}

function formatNumber(num: number | bigint): string {
  const n = typeof num === 'bigint' ? Number(num) : num
  if (n >= 1000000) {
    return (n / 1000000).toFixed(2) + 'M'
  } else if (n >= 1000) {
    return (n / 1000).toFixed(2) + 'K'
  }
  return n.toString()
}

function getRateInHz(device: any): number {
  // Handle both old format (eventRateMs) and new format (rateHz)
  if (device.rateHz !== undefined) {
    return device.rateHz
  } else if (device.eventRateMs !== undefined) {
    // Convert from ms to Hz: 1000ms / eventRateMs
    return device.eventRateMs > 0 ? 1000 / device.eventRateMs : 0
  }
  return 0
}

onMounted(async () => {
  await loadDevices()
  await loadDeviceStates()
  await loadAllStatistics()

  // Auto-refresh every 5 seconds
  setInterval(async () => {
    if (!loading.value) {
      await loadDeviceStates()
      await loadAllStatistics()
    }
  }, 5000)
})
</script>

<template>
  <main>
    <div class="container mx-auto p-4">
      <div class="bg-slate-300 flex flex-col gap-4 p-4 rounded-md">
        <div class="flex justify-between items-center">
          <h1 class="text-2xl font-bold">Test Device Control</h1>
          <div class="form-control">
            <label class="label flex gap-2" for="lock-controls">
              <input type="checkbox" v-model="enabledControlButtons" id="lock-controls" class="toggle toggle-md" />
              <span v-if="enabledControlButtons" class="label-text">Disable control</span>
              <span v-if="!enabledControlButtons" class="label-text">Enable control</span>
            </label>
          </div>
        </div>

        <AlertError v-if="errorMsg" :text="`Error: ${errorMsg}`" />
        <AlertSuccess v-if="successMsg" :text="successMsg" />
        <AlertWarning
          v-if="devices.length === 0 && !errorMsg"
          text="No test devices configured. Configure devices using gen-compose.go"
        />

        <div class="flex flex-wrap gap-2">
          <button
            :disabled="!enabledControlButtons || loading || devices.length === 0"
            @click="startAllDevices"
            class="btn btn-success"
            data-testid="start-devices-button"
          >
            <PlayIcon class="w-5 h-5" />
            Start All Devices
          </button>

          <button
            :disabled="!enabledControlButtons || loading || devices.length === 0"
            @click="stopAllDevices"
            class="btn btn-error"
            data-testid="stop-devices-button"
          >
            <StopIcon class="w-5 h-5" />
            Stop All Devices
          </button>

          <button
            :disabled="loading || devices.length === 0"
            @click="refreshData"
            class="btn btn-info"
            data-testid="refresh-devices-button"
          >
            <ArrowPathIcon class="w-5 h-5" :class="{ 'animate-spin': loading }" />
            Refresh
          </button>
        </div>

        <div v-if="devices.length > 0" class="overflow-x-auto">
          <table class="table table-zebra w-full">
            <thead>
              <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Host IP</th>
                <th>Data Port</th>
                <th>gRPC Port</th>
                <th>Prom Port</th>
                <th>Rate (Hz)</th>
                <th>Packets/Event</th>
                <th>State</th>
                <th>Events Sent</th>
                <th>Bytes Sent</th>
                <th>Errors</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="device in devices" :key="device.equipmentId">
                <td>{{ device.equipmentId }}</td>
                <td class="font-semibold">{{ device.deviceName }}</td>
                <td>{{ device.hostIp }}</td>
                <td>{{ device.hostPort }}</td>
                <td>{{ device.grpcPort }}</td>
                <td>
                  <a
                    :href="`http://localhost:${device.prometheusPort}/metrics`"
                    target="_blank"
                    class="link link-primary"
                  >
                    {{ device.prometheusPort }}
                  </a>
                </td>
                <td>{{ getRateInHz(device) }}</td>
                <td>{{ device.packetsPerEvent }}</td>
                <td>
                  <span
                    class="badge"
                    :class="getStateClass(deviceStates[device.deviceName])"
                  >
                    {{ deviceStates[device.deviceName] || 'Unknown' }}
                  </span>
                </td>
                <td>{{ formatNumber(deviceStats[device.equipmentId]?.events || 0) }}</td>
                <td>{{ formatNumber(deviceStats[device.equipmentId]?.bytes || 0) }}</td>
                <td>{{ formatNumber(deviceStats[device.equipmentId]?.errors || 0) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="bg-blue-100 border-l-4 border-blue-500 p-4">
          <h2 class="text-lg font-bold mb-2">About Test Devices</h2>
          <p class="mb-2">
            Test devices are virtual equipment that generate synthetic DAQ data for testing purposes.
            They simulate real detector equipment by sending UDP packets to LDCs.
          </p>
          <ul class="list-disc list-inside space-y-1">
            <li>Control test devices separately from production DAQ components</li>
            <li>Configure event rates, packet sizes, and error injection via database</li>
            <li>Monitor device metrics via Prometheus endpoints</li>
            <li>Perfect for development, testing, and demonstrations</li>
          </ul>
          <p class="mt-2 text-sm text-gray-700">
            See <code>DATA_GENERATOR_INTEGRATION.md</code> for detailed documentation.
          </p>
        </div>
      </div>
    </div>
  </main>
</template>
