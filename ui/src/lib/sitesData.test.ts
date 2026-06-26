import { describe, expect, it, vi, afterEach } from 'vitest'
import { getGitHubRepos, rollbackDeployment, getRollbackEvents } from './sitesData'

describe('getGitHubRepos', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns error on 404 when GitHub is not installed', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 404,
          json: () =>
            Promise.resolve({ error: 'GitHub App is not installed', code: 'github_not_installed' }),
        }),
      ),
    )
    const result = await getGitHubRepos()
    expect(result.data).toEqual([])
    expect(result.previewMode).toBe(false)
    expect(result.error).toContain('not installed')
  })

  it('returns error when API responds 502 with github_api_error code', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 502,
          json: () =>
            Promise.resolve({
              error: 'failed to list repositories from GitHub',
              code: 'github_api_error',
            }),
        }),
      ),
    )
    const result = await getGitHubRepos()
    expect(result.data).toEqual([])
    expect(result.previewMode).toBe(false)
    expect(result.error).toContain('repositories')
  })
})

describe('rollbackDeployment', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns event on successful rollback', async () => {
    const event = {
      id: 'rb-1',
      site_id: 'site-1',
      rolled_back_from: 'dep-old',
      rolled_back_to: 'dep-new',
      created_at: '2026-06-26T20:00:00Z',
    }
    vi.stubGlobal('fetch', vi.fn(() =>
      Promise.resolve({ ok: true, json: () => Promise.resolve(event) })
    ))
    const result = await rollbackDeployment('dep-new')
    expect(result.ok).toBe(true)
    expect(result.event?.id).toBe('rb-1')
    expect(result.event?.rolled_back_from).toBe('dep-old')
    expect(result.event?.rolled_back_to).toBe('dep-new')
  })

  it('returns error message on API failure (409)', async () => {
    vi.stubGlobal('fetch', vi.fn(() =>
      Promise.resolve({
        ok: false,
        status: 409,
        json: () => Promise.resolve({ error: 'no previous deployment found' }),
      })
    ))
    const result = await rollbackDeployment('dep-1')
    expect(result.ok).toBe(false)
    expect(result.error).toContain('no previous deployment found')
  })

  it('returns generic error on non-JSON response', async () => {
    vi.stubGlobal('fetch', vi.fn(() =>
      Promise.resolve({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error('not json')),
      })
    ))
    const result = await rollbackDeployment('dep-1')
    expect(result.ok).toBe(false)
    expect(result.error).toContain('HTTP 500')
  })

  it('returns network error message on fetch failure', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new Error('network failure'))))
    const result = await rollbackDeployment('dep-1')
    expect(result.ok).toBe(false)
    expect(result.error).toContain('network error')
  })
})

describe('getRollbackEvents', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns events array on success', async () => {
    const events = [
      { id: 'rb-1', site_id: 'site-1', rolled_back_from: 'dep-1', rolled_back_to: 'dep-2', created_at: '2026-06-26T20:00:00Z' },
    ]
    vi.stubGlobal('fetch', vi.fn(() =>
      Promise.resolve({ ok: true, json: () => Promise.resolve({ data: events }) })
    ))
    const result = await getRollbackEvents('site-1')
    expect(result.ok).toBe(true)
    expect(result.events).toHaveLength(1)
    expect(result.events[0].id).toBe('rb-1')
    expect(result.events[0].rolled_back_from).toBe('dep-1')
  })

  it('returns empty events array on API error', async () => {
    vi.stubGlobal('fetch', vi.fn(() =>
      Promise.resolve({ ok: false, status: 500, json: () => Promise.resolve({ error: 'internal error' }) })
    ))
    const result = await getRollbackEvents('site-1')
    expect(result.ok).toBe(false)
    expect(result.events).toEqual([])
  })

  it('returns empty events array on network failure', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new Error('network'))))
    const result = await getRollbackEvents('site-1')
    expect(result.ok).toBe(false)
    expect(result.events).toEqual([])
  })
})
