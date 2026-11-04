import { computed, onMounted, onUnmounted, ref, shallowRef, watch, type Ref } from 'vue'
import { Timestamp } from '@bufbuild/protobuf'
import duckApiClient from '@/api/client'
import type { GetRateHistoryResponse } from '@/gen/api_pb'

export type TimeWindow = { from: number; to: number }
export const timestampMillis = (value?: Timestamp) => value?.toDate().getTime()

export function useRateHistory(liveRun: Ref<number>, running: Ref<boolean>) {
  const selectedRun = ref(0)
  const followLive = ref(true)
  const mode = ref<'auto' | 'manual'>('auto')
  const zoomHistory = ref<(TimeWindow | null)[]>([])
  const canGoBack = computed(() => zoomHistory.value.length > 0)
  const window = ref<TimeWindow | null>(null)
  // Optional demo limit makes adaptive rebinning visible in short test runs.
  const configuredLimit = Number(import.meta.env.VITE_RATE_HISTORY_MAX_POINTS)
  const pointLimit = Number.isInteger(configuredLimit) && configuredLimit >= 10 && configuredLimit <= 2000 ? configuredLimit : 2000
  const pointBudget = ref(Math.min(1000, pointLimit))
  const response = shallowRef<GetRateHistoryResponse | null>(null)
  const overview = shallowRef<GetRateHistoryResponse | null>(null)
  const loading = ref(false)
  const error = ref('')
  const updatedAt = ref<number | null>(null)
  const domain = computed<TimeWindow | null>(() => {
    const data = overview.value ?? response.value
    const from = timestampMillis(data?.from)
    const to = timestampMillis(data?.to)
    return from !== undefined && to !== undefined && to > from ? { from, to } : null
  })
  let timer: ReturnType<typeof setTimeout> | undefined
  let zoomTimer: ReturnType<typeof setTimeout> | undefined
  let controller: AbortController | undefined
  let generation = 0
  let disposed = false
  let mounted = false
  let stoppedStablePolls = 0

  function cancel() {
    clearTimeout(timer)
    clearTimeout(zoomTimer)
    zoomTimer = undefined
    generation++
    controller?.abort()
    loading.value = false
  }

  async function refresh() {
    cancel()
    if (disposed || !mounted || selectedRun.value <= 0 || document.hidden) return
    const requestGeneration = generation
    controller = new AbortController()
    loading.value = true
    try {
      const bounds = mode.value === 'manual' ? window.value : null
      const result = await duckApiClient.getRateHistory({
        run: selectedRun.value,
        maxPoints: pointBudget.value,
        ...(bounds ? { from: Timestamp.fromDate(new Date(bounds.from)), to: Timestamp.fromDate(new Date(bounds.to)) } : {}),
      }, { signal: controller.signal })
      if (generation !== requestGeneration || disposed) return
      const previousLatest = timestampMillis(response.value?.latestSample)
      stoppedStablePolls = result.runStop && previousLatest === timestampMillis(result.latestSample) ? stoppedStablePolls + 1 : 0
      response.value = result
      if (mode.value === 'auto' || !overview.value) overview.value = result
      updatedAt.value = Date.now()
      error.value = ''
    } catch (cause) {
      if (generation !== requestGeneration || disposed) return
      error.value = cause instanceof Error ? cause.message : 'Could not load rate history.'
    } finally {
      if (generation === requestGeneration && !disposed) {
        loading.value = false
        // Allow the final collector writes to arrive after Stop before ending polling.
        if (mode.value === 'auto' && !document.hidden && stoppedStablePolls < 3) {
          timer = setTimeout(() => void refresh(), 5000)
        }
      }
    }
  }

  function selectRun(run: number, follow = false) {
    if (!Number.isInteger(run) || run <= 0 || run > 2147483647) return
    cancel()
    zoomHistory.value = []
    selectedRun.value = run
    followLive.value = follow
    mode.value = 'auto'
    window.value = null
    response.value = overview.value = null
    updatedAt.value = null
    stoppedStablePolls = 0
    error.value = ''
    void refresh()
  }
  function auto() {
    zoomHistory.value = []
    mode.value = 'auto'
    window.value = null
    stoppedStablePolls = 0
    void refresh()
  }
  function manual() {
    cancel()
    mode.value = 'manual'
    window.value = domain.value ? { ...domain.value } : null
  }
  function zoom(bounds: TimeWindow) {
    if (!Number.isFinite(bounds.from) || !Number.isFinite(bounds.to) || bounds.from >= bounds.to) return
    const previous = mode.value === 'manual' ? window.value : null
    const sameWindow = previous && Math.abs(previous.from - bounds.from) < 1 && Math.abs(previous.to - bounds.to) < 1
    if (!sameWindow && zoomTimer === undefined) {
      zoomHistory.value.push(previous ? { ...previous } : null)
      if (zoomHistory.value.length > 50) zoomHistory.value.shift()
    }
    cancel()
    mode.value = 'manual'
    window.value = bounds
    zoomTimer = setTimeout(() => { zoomTimer = undefined; void refresh() }, 250)
  }
  function back() {
    if (!canGoBack.value) return
    const previous = zoomHistory.value.pop()!
    cancel()
    mode.value = previous ? 'manual' : 'auto'
    window.value = previous
    stoppedStablePolls = 0
    void refresh()
  }
  function setWidth(width: number) {
    const budget = Math.min(pointLimit, Math.max(100, Math.round(width)))
    if (Math.abs(budget - pointBudget.value) < 50) return
    pointBudget.value = budget
    clearTimeout(zoomTimer)
    zoomTimer = setTimeout(() => void refresh(), 250)
  }
  function visibilityChanged() {
    if (document.hidden) cancel()
    else if (mode.value === 'auto') void refresh()
  }
  watch([liveRun, running], ([run, active]) => {
    if (followLive.value && run > 0 && (selectedRun.value === 0 || active && run !== selectedRun.value)) selectRun(run, true)
  })
  onMounted(() => {
    mounted = true
    document.addEventListener('visibilitychange', visibilityChanged)
    if (liveRun.value > 0 && selectedRun.value === 0) selectRun(liveRun.value, true)
    else void refresh()
  })
  onUnmounted(() => {
    disposed = true
    cancel()
    document.removeEventListener('visibilitychange', visibilityChanged)
  })
  return { selectedRun, followLive, mode, window, pointBudget, response, overview, domain, loading, error, updatedAt, canGoBack, back, pause: cancel, refresh, selectRun, auto, manual, zoom, setWidth }
}
