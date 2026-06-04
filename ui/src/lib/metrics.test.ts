import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { fetchMonitoringMetrics, fetchMonitoringMetricsWarmed } from './metrics'

describe('fetchMonitoringMetrics', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  it('fetches metrics once', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ system: { cpu_percent: 5, memory_mb: 10, goroutines: 1, uptime_seconds: 1 } }),
    } as Response)

    const data = await fetchMonitoringMetrics()
    expect(data.system?.cpu_percent).toBe(5)
    expect(fetch).toHaveBeenCalledTimes(1)
  })

  it('warm fetch calls metrics twice after gap', async () => {
    const mock = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ system: { cpu_percent: 12, memory_mb: 10, goroutines: 1, uptime_seconds: 1 } }),
    } as Response)

    const promise = fetchMonitoringMetricsWarmed(undefined, 200)
    await vi.advanceTimersByTimeAsync(200)
    const data = await promise

    expect(data.system?.cpu_percent).toBe(12)
    expect(mock).toHaveBeenCalledTimes(2)
    expect(mock.mock.calls[0][0]).toContain('/api/monitoring/metrics')
  })
})
