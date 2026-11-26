<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, DataZoomComponent, BrushComponent, AriaComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import type { EChartsOption } from 'echarts'
import type { RateHistoryBin } from '@/gen/api_pb'
import { timestampMillis, type TimeWindow } from '@/composables/useRateHistory'

use([LineChart, GridComponent, TooltipComponent, DataZoomComponent, BrushComponent, AriaComponent, CanvasRenderer])
const props = defineProps<{ bins: RateHistoryBin[]; domain: TimeWindow; window: TimeWindow | null; run: number; canGoBack: boolean }>()
const emit = defineEmits<{ zoom: [TimeWindow]; resize: [number]; back: []; selecting: [boolean] }>()
const selecting = ref(false)
const root = ref<HTMLElement>()
const chartWidth = ref(1000)
const chart = ref<InstanceType<typeof VChart>>()
let observer: ResizeObserver | undefined
const formatTime = (value: number) => new Date(value).toISOString().replace('T', ' ').replace('.000Z', ' UTC')
const option = computed<EChartsOption>(() => {
  const data: [number, number | null][] = []
  for (const bin of props.bins) {
    const start = timestampMillis(bin.start), end = timestampMillis(bin.end)
    if (start === undefined || end === undefined) continue
    const rate = bin.complete && bin.triggerRateHz !== undefined ? bin.triggerRateHz : null
    data.push([start, rate], [end, rate])
  }
  const bounds = props.window ?? props.domain
  return {
    animation: false,
    useUTC: true,
    aria: { enabled: true, description: `Aggregate trigger rate history for run ${props.run}. Missing coverage is shown as gaps.` },
    grid: { left: 65, right: 25, top: 45, bottom: 85 },
    xAxis: { axisLabel: { hideOverlap: true }, splitNumber: Math.max(3, Math.min(12, Math.floor(chartWidth.value / 110))), type: 'time', min: props.domain.from, max: props.domain.to, name: 'UTC', nameLocation: 'middle', nameGap: 30 },
    yAxis: { type: 'value', min: 0, name: 'Hz' },
    tooltip: {
      trigger: 'axis', renderMode: 'richText', confine: true,
      formatter: (raw: unknown) => {
        const entry = (raw as { axisValue?: number }[])[0]
        const at = Number(entry?.axisValue)
        const bin = props.bins.find(bin => at >= (timestampMillis(bin.start) ?? Infinity) && at < (timestampMillis(bin.end) ?? -Infinity))
        if (!bin) return ''
        return `${formatTime(timestampMillis(bin.start)!)}\nto ${formatTime(timestampMillis(bin.end)!)}\n${bin.triggerRateHz === undefined ? 'Rate unavailable' : `${bin.triggerRateHz.toFixed(2)} Hz`}\nCoverage: ${(bin.coverage * 100).toFixed(1)}%`
      },
    },
    brush: { toolbox: [], xAxisIndex: 0, brushType: false, brushMode: 'single', outOfBrush: { colorAlpha: 1 } },
    dataZoom: [
      { type: 'inside', xAxisIndex: 0, filterMode: 'none', startValue: bounds.from, endValue: bounds.to, zoomOnMouseWheel: 'ctrl', throttle: 100 },
      { type: 'slider', xAxisIndex: 0, filterMode: 'none', startValue: bounds.from, endValue: bounds.to, showDataShadow: false, bottom: 15, height: 22, throttle: 100 },
    ],
    series: [{ id: 'trigger-rate', name: 'Trigger rate', type: 'line', showSymbol: false, connectNulls: false, smooth: false, data, lineStyle: { width: 1.5, color: '#2563eb' }, itemStyle: { color: '#2563eb' } }],
  }
})
type ZoomRange = { startValue?: number; endValue?: number }
function zoomed(raw: unknown) {
  const event = raw as ZoomRange & { batch?: ZoomRange[] }
  const options = chart.value?.getOption() as { dataZoom?: ZoomRange[] } | undefined
  const zoom = event.batch?.[0] ?? event
  const fromValue = zoom.startValue ?? options?.dataZoom?.[0]?.startValue
  const toValue = zoom.endValue ?? options?.dataZoom?.[0]?.endValue
  if (fromValue === undefined || toValue === undefined) return
  const from = Math.max(props.domain.from, Number(fromValue)), to = Math.min(props.domain.to, Number(toValue))
  if (to > from) emit('zoom', { from, to })
}
async function selectRange() {
  selecting.value = !selecting.value
  emit('selecting', selecting.value)
  await nextTick()
  chart.value?.dispatchAction({ type: 'takeGlobalCursor', key: 'brush', brushOption: { brushType: selecting.value ? 'lineX' : false, brushMode: 'single' } })
}
function selected(raw: unknown) {
  const event = raw as { areas?: { coordRange?: number[] }[] }
  const range = event.areas?.[0]?.coordRange
  if (!range || range.length !== 2) return
  const from = Math.max(props.domain.from, Math.min(...range)), to = Math.min(props.domain.to, Math.max(...range))
  selecting.value = false
  chart.value?.dispatchAction({ type: 'brush', areas: [] })
  chart.value?.dispatchAction({ type: 'takeGlobalCursor', key: 'brush', brushOption: { brushType: false } })
  if (to > from) emit('zoom', { from, to })
}
onMounted(() => {
  observer = new ResizeObserver(entries => {
    const width = entries[0].contentRect.width
    if (width > 0) { chartWidth.value = width; emit('resize', width) }
  })
  if (root.value) observer.observe(root.value)
})
onUnmounted(() => observer?.disconnect())
</script>

<template>
  <div ref="root" data-testid="rate-history-chart">
    <div class="flex justify-end items-center gap-2 mt-1">
      <span v-if="selecting" class="text-xs text-slate-600" role="status">Drag across the plot to select a time range.</span>
      <button type="button" class="btn btn-sm" :class="{ 'btn-primary': selecting }" :aria-pressed="selecting" @click="selectRange">{{ selecting ? 'Cancel selection' : 'Select range' }}</button>
      <button type="button" class="btn btn-sm" :disabled="!canGoBack" @click="emit('back')">Back</button>
    </div>
    <div class="rate-chart" data-testid="rate-history-canvas">
      <VChart ref="chart" :option="option" :update-options="{ replaceMerge: ['series'], silent: true }" autoresize @datazoom="zoomed" @brushend="selected" />
    </div>
  </div>
</template>
<style scoped>
.rate-chart { width: 100%; height: 360px; }
@media (max-width: 640px) { .rate-chart { height: 300px; } }
</style>
