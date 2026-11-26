<script setup lang="ts">
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useMessagesStore } from '@/stores/messages'
import { useRateHistory } from '@/composables/useRateHistory'
const RateHistoryChart = defineAsyncComponent(() => import('./RateHistoryChart.vue'))

const messagesStore = useMessagesStore()
const { runNumber, status } = storeToRefs(messagesStore)
const history = useRateHistory(runNumber, computed(() => status.value === 'RUNNING'))
const { selectedRun, mode, window, response, domain, loading, error, updatedAt, followLive, canGoBack } = history
// A run may be started from another browser; refresh its number on RUNNING.
watch(status, state => { if (state === 'RUNNING' && followLive.value) void messagesStore.getRunNumber() })
const fromInput = ref(''), toInput = ref('')
watch(window, bounds => {
  fromInput.value = bounds ? new Date(bounds.from).toISOString().slice(0, 19) : ''
  toInput.value = bounds ? new Date(bounds.to).toISOString().slice(0, 19) : ''
})
const manualBounds = computed(() => ({ from: Date.parse(fromInput.value + 'Z'), to: Date.parse(toInput.value + 'Z') }))
const validBounds = computed(() => Number.isFinite(manualBounds.value.from) && Number.isFinite(manualBounds.value.to) && manualBounds.value.from < manualBounds.value.to)
const incompleteBins = computed(() => response.value?.bins.filter(bin => !bin.complete).length ?? 0)
</script>

<template>
  <section class="rate-history rounded-lg border border-slate-300 bg-white text-slate-800 p-4 mx-auto my-4 max-w-7xl" aria-labelledby="rate-history-title" data-testid="rate-history-panel">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2 id="rate-history-title" class="text-xl font-bold">Trigger rate history</h2>
      <span v-if="selectedRun > 0" class="text-sm text-slate-600">Run {{ selectedRun }}</span>
    </div>
    <div class="flex flex-wrap items-center gap-2 mt-3">
      <div class="join" aria-label="History display mode">
        <button class="btn btn-sm join-item" :class="{ 'btn-primary': mode === 'auto' }" :aria-pressed="mode === 'auto'" @click="history.auto">Auto</button>
        <button class="btn btn-sm join-item" :class="{ 'btn-primary': mode === 'manual' }" :aria-pressed="mode === 'manual'" :disabled="!domain" @click="history.manual">Manual</button>
      </div>
      <button class="btn btn-sm btn-ghost" :disabled="selectedRun <= 0 || loading" @click="history.refresh">Refresh</button>
      <span class="text-sm text-slate-600" role="status" aria-live="polite">
        <template v-if="loading">Updating…</template>
        <template v-else-if="updatedAt">Updated {{ new Date(updatedAt).toLocaleTimeString() }}</template>
        <template v-if="response?.binWidthSeconds"> · {{ response.binWidthSeconds }} s bins</template>
      </span>
    </div>
    <form v-if="mode === 'manual'" class="flex flex-wrap items-end gap-3 my-3" @submit.prevent="history.zoom(manualBounds)">
      <label class="text-sm">From (UTC)<input v-model="fromInput" class="input input-bordered input-sm block" type="datetime-local" step="1" /></label>
      <label class="text-sm">To (UTC)<input v-model="toInput" class="input input-bordered input-sm block" type="datetime-local" step="1" /></label>
      <button class="btn btn-sm" :disabled="!validBounds">Apply window</button>
    </form>
    <p v-if="error" class="text-red-700 mt-3" role="alert">{{ error }} <span v-if="response">Showing the last successful result.</span></p>
    <template v-if="domain && response?.bins.length">
      <RateHistoryChart :bins="response.bins" :domain="domain" :window="window" :run="selectedRun" :can-go-back="canGoBack" @back="history.back" @selecting="active => active ? history.pause() : history.refresh()" @zoom="history.zoom" @resize="history.setWidth" />
      <p class="text-xs text-slate-600">Sum of GDC rates from event-counter differences. Drag the time slider or choose Select range and drag to zoom; Ctrl + wheel also zooms. Auto follows the whole run.</p>
      <p v-if="incompleteBins" class="text-sm text-amber-800 mt-1">{{ incompleteBins }} interval(s) have incomplete coverage and appear as gaps.</p>
      <p class="text-xs text-slate-500 mt-1">Coverage is relative to GDCs recorded in this run.</p>
    </template>
    <p v-else-if="!loading && !error" class="py-12 text-center text-slate-500">{{ selectedRun > 0 ? (mode === 'manual' ? 'No rate history in the selected time window.' : 'No rate history available for this run yet.') : 'Waiting for the current run.' }}</p>
    <p v-else-if="loading && !response" class="py-12 text-center text-slate-500">Loading rate history…</p>
  </section>
</template>
