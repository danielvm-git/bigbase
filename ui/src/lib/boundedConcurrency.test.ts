import { describe, expect, it } from 'vitest'
import { mapWithConcurrency } from './boundedConcurrency'

describe('mapWithConcurrency', () => {
  it('returns empty array when items is empty without spawning workers', async () => {
    let callCount = 0
    const res = await mapWithConcurrency([], 4, async (item) => {
      callCount++
      return item
    })
    expect(res).toEqual([])
    expect(callCount).toBe(0)
  })

  it('preserves input order even when tasks finish out of order', async () => {
    const items = [50, 10, 30, 20]
    const res = await mapWithConcurrency(items, 4, async (delay, idx) => {
      await new Promise(r => setTimeout(r, delay))
      return `item-${idx}-${delay}`
    })
    expect(res).toEqual([
      'item-0-50',
      'item-1-10',
      'item-2-30',
      'item-3-20',
    ])
  })

  it('bounds active concurrency to limit', async () => {
    const items = Array.from({ length: 10 }, (_, i) => i)
    let active = 0
    let maxActive = 0
    await mapWithConcurrency(items, 3, async () => {
      active++
      maxActive = Math.max(maxActive, active)
      await new Promise(r => setTimeout(r, 20))
      active--
    })
    expect(maxActive).toBeLessThanOrEqual(3)
  })

  it('aborts remaining tasks on failure without unhandled rejections', async () => {
    const items = [1, 2, 3, 4, 5, 6, 7, 8]
    const executed: number[] = []
    await expect(
      mapWithConcurrency(items, 2, async (item) => {
        executed.push(item)
        if (item === 2) {
          throw new Error('fail on 2')
        }
        await new Promise(r => setTimeout(r, 30))
        return item
      }),
    ).rejects.toThrow('fail on 2')

    // Remaining tasks beyond the initial worker pool must not be executed
    expect(executed).not.toContain(5)
    expect(executed).not.toContain(6)
    expect(executed).not.toContain(7)
    expect(executed).not.toContain(8)
  })
})
