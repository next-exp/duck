import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { Timestamp } from '@bufbuild/protobuf'
import { GetRateHistoryResponse } from '@/gen/api_pb'
import { mockDuckApiClient } from '@/test/setup'
import { useRateHistory } from '../useRateHistory'

const origin = Date.parse('2026-11-04T12:00:00Z')
function result(run = 123) {
  return new GetRateHistoryResponse({ run, hasData: true, from: Timestamp.fromDate(new Date(origin)), to: Timestamp.fromDate(new Date(origin + 3600000)), latestSample: Timestamp.fromDate(new Date(origin + 3600000)), bins: [], binWidthSeconds: 5 })
}
function setup() {
  const run = ref(123), running = ref(true)
  let history!: ReturnType<typeof useRateHistory>
  const wrapper = mount(defineComponent({ setup() { history = useRateHistory(run, running); return () => null } }))
  return { wrapper, history, run, running }
}
let cleanup: (() => void) | undefined
beforeEach(() => {
  vi.useFakeTimers()
  Object.defineProperty(document, 'hidden', { configurable: true, value: false })
  mockDuckApiClient.getRateHistory.mockReset().mockResolvedValue(result())
})
afterEach(() => { cleanup?.(); cleanup = undefined; vi.useRealTimers() })

describe('rate history requests', () => {
  it('polls after completion, requests a bounded whole-run series, and preserves the run after Stop', async () => {
    const { wrapper, history, run, running } = setup(); cleanup = () => wrapper.unmount()
    await flushPromises()
    expect(mockDuckApiClient.getRateHistory).toHaveBeenCalledWith(expect.objectContaining({ run: 123, maxPoints: 1000 }), expect.objectContaining({ signal: expect.any(AbortSignal) }))
    running.value = false; run.value = 124
    await flushPromises()
    expect(history.selectedRun.value).toBe(123)
    await vi.advanceTimersByTimeAsync(5000)
    expect(mockDuckApiClient.getRateHistory).toHaveBeenCalledTimes(2)
    running.value = true
    await flushPromises()
    expect(history.selectedRun.value).toBe(124)
  })
  it('debounces zoom, retains the overview, and restores automatic requests', async () => {
    const { wrapper, history } = setup(); cleanup = () => wrapper.unmount()
    await flushPromises()
    const overview = history.overview.value
    history.zoom({ from: origin + 10000, to: origin + 60000 })
    history.zoom({ from: origin + 20000, to: origin + 50000 })
    await vi.advanceTimersByTimeAsync(250)
    expect(mockDuckApiClient.getRateHistory).toHaveBeenCalledTimes(2)
    const request = mockDuckApiClient.getRateHistory.mock.calls[1][0] as any
    expect(request.from.toDate().getTime()).toBe(origin + 20000)
    expect(history.overview.value).toBe(overview)
    await vi.advanceTimersByTimeAsync(10000)
    expect(mockDuckApiClient.getRateHistory).toHaveBeenCalledTimes(2)
    history.auto(); await flushPromises()
    expect(mockDuckApiClient.getRateHistory.mock.calls[2][0]).not.toHaveProperty('from')
  })
  it('ignores superseded results even if the transport ignores abort', async () => {
    let resolve!: (response: GetRateHistoryResponse) => void
    mockDuckApiClient.getRateHistory.mockImplementationOnce(() => new Promise(done => { resolve = done }))
    const { wrapper, history } = setup(); cleanup = () => wrapper.unmount()
    mockDuckApiClient.getRateHistory.mockResolvedValue(result(222))
    history.selectRun(222)
    await flushPromises()
    resolve(result(123)); await flushPromises()
    expect(history.selectedRun.value).toBe(222)
    expect(history.response.value?.run).toBe(222)
  })
  it('pauses hidden pages, resumes on visibility, and cleans timers on unmount', async () => {
    const { wrapper } = setup(); await flushPromises()
    Object.defineProperty(document, 'hidden', { configurable: true, value: true })
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(10000)
    expect(mockDuckApiClient.getRateHistory).toHaveBeenCalledTimes(1)
    Object.defineProperty(document, 'hidden', { configurable: true, value: false })
    document.dispatchEvent(new Event('visibilitychange')); await flushPromises()
    expect(mockDuckApiClient.getRateHistory).toHaveBeenCalledTimes(2)
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(10000)
    expect(mockDuckApiClient.getRateHistory).toHaveBeenCalledTimes(2)
  })
  it('does not overlap slow automatic requests', async () => {
    let resolve!: (value: GetRateHistoryResponse) => void
    mockDuckApiClient.getRateHistory.mockImplementationOnce(() => new Promise(done => { resolve = done }))
    const { wrapper } = setup(); cleanup = () => wrapper.unmount()
    await vi.advanceTimersByTimeAsync(20000)
    expect(mockDuckApiClient.getRateHistory).toHaveBeenCalledTimes(1)
    resolve(result()); await flushPromises()
    await vi.advanceTimersByTimeAsync(5000)
    expect(mockDuckApiClient.getRateHistory).toHaveBeenCalledTimes(2)
  })
  it('ends automatic polling only after final stopped-run data settles', async () => {
    const stopped = result(); stopped.runStop = Timestamp.fromDate(new Date(origin + 3600000))
    mockDuckApiClient.getRateHistory.mockResolvedValue(stopped)
    const { wrapper } = setup(); cleanup = () => wrapper.unmount()
    await flushPromises()
    await vi.advanceTimersByTimeAsync(30000)
    expect(mockDuckApiClient.getRateHistory).toHaveBeenCalledTimes(4)
  })
  it('keeps zoom history across requests and restores prior views', async () => {
    const { wrapper, history } = setup(); cleanup = () => wrapper.unmount()
    await flushPromises()
    history.zoom({ from: origin + 10000, to: origin + 60000 })
    await vi.advanceTimersByTimeAsync(250)
    history.zoom({ from: origin + 20000, to: origin + 40000 })
    await vi.advanceTimersByTimeAsync(250)
    expect(history.canGoBack.value).toBe(true)
    history.back(); await flushPromises()
    expect(history.window.value).toEqual({ from: origin + 10000, to: origin + 60000 })
    history.back(); await flushPromises()
    expect(history.window.value).toBeNull()
    expect(history.mode.value).toBe('auto')
    expect(history.canGoBack.value).toBe(false)
  })
  it('keeps the last successful data on failure and bounds the resize budget', async () => {
    const { wrapper, history } = setup(); cleanup = () => wrapper.unmount()
    await flushPromises()
    const previous = history.response.value
    mockDuckApiClient.getRateHistory.mockRejectedValue(new Error('Server unavailable'))
    history.setWidth(5000); await vi.advanceTimersByTimeAsync(250)
    expect(history.pointBudget.value).toBe(2000)
    expect(history.response.value).toBe(previous)
    expect(history.error.value).toBe('Server unavailable')
  })
})
