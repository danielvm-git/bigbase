import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { IMPORT_CONCURRENCY_LIMIT } from '../lib/boundedConcurrency'

const getEnvVarsMock = vi.fn()
const createEnvVarMock = vi.fn()
const updateEnvVarMock = vi.fn()
const deleteEnvVarMock = vi.fn()

vi.mock('../lib/sitesData', () => ({
  getEnvVars: (...args: unknown[]) => getEnvVarsMock(...args),
  createEnvVar: (...args: unknown[]) => createEnvVarMock(...args),
  updateEnvVar: (...args: unknown[]) => updateEnvVarMock(...args),
  deleteEnvVar: (...args: unknown[]) => deleteEnvVarMock(...args),
}))

import { SiteEnvVarsTab } from './SiteEnvVarsTab'

function envFile(keys: string[]): File {
  return new File([keys.map((key, i) => `${key}=value${i}`).join('\n')], 'bulk.env', {
    type: 'text/plain',
  })
}

async function renderReady() {
  render(<SiteEnvVarsTab siteId="site-1" />)
  await screen.findByRole('button', { name: /Import \.env/ })
}

async function importKeys(keys: string[]) {
  fireEvent.change(screen.getByLabelText('Import .env file'), {
    target: { files: [envFile(keys)] },
  })
}

describe('SiteEnvVarsTab .env import', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getEnvVarsMock.mockResolvedValue([])
    createEnvVarMock.mockResolvedValue({ ok: true })
    updateEnvVarMock.mockResolvedValue({ ok: true })
    deleteEnvVarMock.mockResolvedValue({ ok: true })
  })

  it('caps concurrent env-var writes at the configured import limit', async () => {
    let inFlight = 0
    let maxInFlight = 0
    createEnvVarMock.mockImplementation(async () => {
      inFlight += 1
      maxInFlight = Math.max(maxInFlight, inFlight)
      await new Promise(resolve => setTimeout(resolve, 5))
      inFlight -= 1
      return { ok: true }
    })

    await renderReady()
    const keys = Array.from({ length: 12 }, (_, i) => `KEY_${i}`)
    await importKeys(keys)

    await waitFor(() => {
      expect(createEnvVarMock).toHaveBeenCalledTimes(keys.length)
    })
    expect(maxInFlight).toBe(IMPORT_CONCURRENCY_LIMIT)
    expect(maxInFlight).toBeLessThanOrEqual(IMPORT_CONCURRENCY_LIMIT)
  })

  it('reports per-key failures in input order and never echoes values', async () => {
    const failing = new Set(['KEY_1', 'KEY_4', 'KEY_7'])
    createEnvVarMock.mockImplementation(
      async (_siteId: string, payload: { key: string }) => {
        const index = Number(payload.key.slice('KEY_'.length))
        await new Promise(resolve => setTimeout(resolve, (10 - index) * 2))
        if (failing.has(payload.key)) return { ok: false, error: 'write rejected' }
        return { ok: true }
      },
    )

    await renderReady()
    const keys = Array.from({ length: 10 }, (_, i) => `KEY_${i}`)
    await importKeys(keys)

    const message = await screen.findByText(/3 var\(s\) failed to import/)
    const text = message.textContent ?? ''
    expect(failing.size).toBe(3)
    const order = [...failing].map(key => text.indexOf(key))
    expect(order).toEqual([...order].sort((a, b) => a - b))
    expect(order.every(position => position >= 0)).toBe(true)
    // Submitted values never appear in the aggregated error report.
    expect(text).not.toMatch(/value\d/)
  })

  it('deduplicates duplicate keys in imported file, keeping the last value', async () => {
    createEnvVarMock.mockResolvedValue({ ok: true })
    await renderReady()
    const file = new File(['KEY_A=first\nKEY_B=only\nKEY_A=second\n'], 'bulk.env', { type: 'text/plain' })
    fireEvent.change(screen.getByLabelText('Import .env file'), {
      target: { files: [file] },
    })

    await waitFor(() => {
      expect(createEnvVarMock).toHaveBeenCalledTimes(2)
    })
    expect(createEnvVarMock).toHaveBeenCalledWith('site-1', expect.objectContaining({ key: 'KEY_A', value: 'second' }))
    expect(createEnvVarMock).toHaveBeenCalledWith('site-1', expect.objectContaining({ key: 'KEY_B', value: 'only' }))
  })
})
