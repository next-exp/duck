<script setup lang="ts">
import { onMounted, ref } from "vue";
import type { Ref } from "vue";
import { useGDCStore } from '../stores/gdc'
import { useLDCStore } from '../stores/ldc'
import { useMessagesStore } from "@/stores/messages";
import { storeToRefs } from 'pinia'

const gdcStore = useGDCStore()
const ldcStore = useLDCStore()
const messagesStore = useMessagesStore()
const { gdcs, errorGDCs } = storeToRefs(gdcStore)
const { ldcs, errorLDCs } = storeToRefs(ldcStore)
const { evts, bytes, evtRateAvg, evtRateCurrent, byteRateAvg, byteRateCurrent, states, fileNumbers } = storeToRefs(messagesStore)
const { evtsTotal, bytesTotal, evtRateAvgTotal, evtRateCurrentTotal, byteRateAvgTotal, byteRateCurrentTotal } = storeToRefs(messagesStore)

function formatBytes(bytes: number, decimals: number) {
  if (bytes == 0) return '0 Bytes';
  var k = 1024,
    dm = decimals || 2,
    sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'],
    i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}
</script>

<template>
  <div data-testid="run-statistics-container" class="rounded-lg w-fit mx-auto p-2">
    <div class="hidden max-lg:block">
      <table data-testid="statistics-table-mobile" class="table table-zebra">
        <thead>
          <tr>
            <th>Server</th>
            <th>GDCs</th>
            <th>LDCs</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Total events</td>
            <td>{{ evtsTotal["gdc"] ?? 0 }}</td>
            <td>{{ evtsTotal["ldc"] ?? 0 }}</td>
          </tr>
          <tr>
            <td>Total size</td>
            <td>{{ formatBytes(bytesTotal["gdc"] ?? 0, 2) }}</td>
            <td>{{ formatBytes(bytesTotal["ldc"] ?? 0, 2) }}</td>
          </tr>
          <tr>
            <td>Current trigger rate</td>
            <td>{{ (evtRateCurrentTotal["gdc"] ?? 0).toFixed(2) }} Hz</td>
            <td>{{ (evtRateCurrentTotal["ldc"] ?? 0).toFixed(2) }} Hz</td>
          </tr>
          <tr>
            <td>Average trigger rate</td>
            <td>{{ (evtRateAvgTotal["gdc"] ?? 0).toFixed(2) }} Hz</td>
            <td>{{ (evtRateAvgTotal["ldc"] ?? 0).toFixed(2) }} Hz</td>
          </tr>
          <tr>
            <td>Current byte rate</td>
            <td>{{ formatBytes(byteRateCurrentTotal["gdc"] ?? 0, 2) }}/s</td>
            <td>{{ formatBytes(byteRateCurrentTotal["ldc"] ?? 0, 2) }}/s</td>
          </tr>
          <tr>
            <td>Average byte rate</td>
            <td>{{ formatBytes(byteRateAvgTotal["gdc"] ?? 0, 2) }}/s</td>
            <td>{{ formatBytes(byteRateAvgTotal["ldc"] ?? 0, 2) }}/s</td>
          </tr>
        </tbody>
      </table>
    </div>

    <table data-testid="statistics-table-desktop" class="table table-zebra max-2xl:hidden">
      <thead>
        <tr>
          <th>Server</th>
          <th v-for="gdc in gdcs">GDC {{ gdc.name }}</th>
          <th v-for="ldc in ldcs">LDC {{ ldc.name }}</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>State</td>
          <td v-for="gdc in gdcs">{{ states[gdc.name] ?? '' }} </td>
          <td v-for="ldc in ldcs">{{ states[ldc.name] ?? '' }} </td>
        </tr>
        <tr>
          <td>Total files</td>
          <td v-for="gdc in gdcs">{{ fileNumbers[gdc.name] ?? 0 }}</td>
          <td v-for="ldc in ldcs">{{ fileNumbers[ldc.name] ?? 0 }}</td>
        </tr>
        <tr>
          <td>Total events</td>
          <td v-for="gdc in gdcs">{{ evts[gdc.name] ?? 0 }}</td>
          <td v-for="ldc in ldcs">{{ evts[ldc.name] ?? 0 }}</td>
        </tr>
        <tr>
          <td>Total size</td>
          <td v-for="gdc in gdcs">{{ formatBytes(bytes[gdc.name] ?? 0, 2) }}</td>
          <td v-for="ldc in ldcs">{{ formatBytes(bytes[ldc.name] ?? 0, 2) }}</td>
        </tr>
        <tr>
          <td>Current trigger rate</td>
          <td v-for="gdc in gdcs">{{ (evtRateCurrent[gdc.name] ?? 0).toFixed(2) }} Hz</td>
          <td v-for="ldc in ldcs">{{ (evtRateCurrent[ldc.name] ?? 0).toFixed(2) }} Hz</td>
        </tr>
        <tr>
          <td>Average trigger rate</td>
          <td v-for="gdc in gdcs">{{ (evtRateAvg[gdc.name] ?? 0).toFixed(2) }} Hz</td>
          <td v-for="ldc in ldcs">{{ (evtRateAvg[ldc.name] ?? 0).toFixed(2) }} Hz</td>
        </tr>
        <tr>
          <td>Current byte rate</td>
          <td v-for="gdc in gdcs">{{ formatBytes(byteRateCurrent[gdc.name] ?? 0, 2) }}/s</td>
          <td v-for="ldc in ldcs">{{ formatBytes(byteRateCurrent[ldc.name] ?? 0, 2) }}/s</td>
        </tr>
        <tr>
          <td>Average byte rate</td>
          <td v-for="gdc in gdcs">{{ formatBytes(byteRateAvg[gdc.name] ?? 0, 2) }}/s</td>
          <td v-for="ldc in ldcs">{{ formatBytes(byteRateAvg[ldc.name] ?? 0, 2) }}/s</td>
        </tr>
      </tbody>
    </table>

    <div class="hidden lg:max-2xl:block">
      <h2 class="text-center text-2xl">GDCs</h2>
      <table class="table table-zebra">
        <thead>
          <tr>
            <th>Server</th>
            <th v-for="gdc in gdcs">GDC {{ gdc.name }}</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>State</td>
            <td v-for="gdc in gdcs">{{ states[gdc.name] ?? '' }} </td>
          </tr>
          <tr>
            <td>Total files</td>
            <td v-for="gdc in gdcs">{{ fileNumbers[gdc.name] ?? 0 }}</td>
          </tr>
          <tr>
            <td>Total events</td>
            <td v-for="gdc in gdcs">{{ evts[gdc.name] ?? 0 }}</td>
          </tr>
          <tr>
            <td>Total size</td>
            <td v-for="gdc in gdcs">{{ formatBytes(bytes[gdc.name] ?? 0, 2) }}</td>
          </tr>
          <tr>
            <td>Current trigger rate</td>
            <td v-for="gdc in gdcs">{{ (evtRateCurrent[gdc.name] ?? 0).toFixed(2) }} Hz</td>
          </tr>
          <tr>
            <td>Average trigger rate</td>
            <td v-for="gdc in gdcs">{{ (evtRateAvg[gdc.name] ?? 0).toFixed(2) }} Hz</td>
          </tr>
          <tr>
            <td>Current byte rate</td>
            <td v-for="gdc in gdcs">{{ formatBytes(byteRateCurrent[gdc.name] ?? 0, 2) }}/s</td>
          </tr>
          <tr>
            <td>Average byte rate</td>
            <td v-for="gdc in gdcs">{{ formatBytes(byteRateAvg[gdc.name] ?? 0, 2) }}/s</td>
          </tr>
        </tbody>
      </table>

      <h2 class="text-center text-2xl">LDCs</h2>

      <table class="table table-zebra mt-4">
        <thead>
          <tr>
            <th>Server</th>
            <th v-for="ldc in ldcs">LDC {{ ldc.name }}</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>State</td>
            <td v-for="ldc in ldcs">{{ states[ldc.name] ?? '' }} </td>
          </tr>
          <tr>
            <td>Total files</td>
            <td v-for="ldc in ldcs">{{ fileNumbers[ldc.name] ?? 0 }}</td>
          </tr>
          <tr>
            <td>Total events</td>
            <td v-for="ldc in ldcs">{{ evts[ldc.name] ?? 0 }}</td>
          </tr>
          <tr>
            <td>Total size</td>
            <td v-for="ldc in ldcs">{{ formatBytes(bytes[ldc.name] ?? 0, 2) }}</td>
          </tr>
          <tr>
            <td>Current trigger rate</td>
            <td v-for="ldc in ldcs">{{ (evtRateCurrent[ldc.name] ?? 0).toFixed(2) }} Hz</td>
          </tr>
          <tr>
            <td>Average trigger rate</td>
            <td v-for="ldc in ldcs">{{ (evtRateAvg[ldc.name] ?? 0).toFixed(2) }} Hz</td>
          </tr>
          <tr>
            <td>Current byte rate</td>
            <td v-for="ldc in ldcs">{{ formatBytes(byteRateCurrent[ldc.name] ?? 0, 2) }}/s</td>
          </tr>
          <tr>
            <td>Average byte rate</td>
            <td v-for="ldc in ldcs">{{ formatBytes(byteRateAvg[ldc.name] ?? 0, 2) }}/s</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
