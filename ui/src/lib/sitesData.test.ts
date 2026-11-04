import { describe, expect, it, vi, afterEach } from 'vitest'
import {
  getGitHubRepos,
  getRollbackEvents,
  getSite,
  getSiteDeployments,
  rollbackDeployment,
} from './sitesData'
import type { Site } from '../types/sites'

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function stubFetchUrls(handler: (url: string) => Response) {
  const urls: string[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : String(input)
      urls.push(url)
      return Promise.resolve(handler(url))
    }),
  )
  return urls
}

const siteFixture: Site = {
  id: 'site-1',
  name: 'My Site',
  full_name: 'org/mysite',
  git_repo_id: 'repo-1',
  production_branch: 'main',
  root_path: './',
}

const deploymentFixture = {
  id: 'd1',
  repo_id: 'repo-1',
  branch: 'main',
  commit_sha: 'abc1234',
  status: 'running',
  url: '',
  port: 4000,
  app_type: 'static',
  created_at: '2026-06-01T10:00:00Z',
}

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

describe('getSiteDeployments (prefetched site)', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('falls back to getSite for legacy callers that pass no site', async () => {
    const urls = stubFetchUrls(url => {
      if (url === '/api/sites/site-1') return jsonResponse(siteFixture)
      if (url === '/api/deploy') return jsonResponse({ data: [deploymentFixture] })
      return jsonResponse({}, 404)
    })

    const result = await getSiteDeployments('site-1')

    expect(urls).toContain('/api/sites/site-1')
    expect(urls).toContain('/api/deploy')
    expect(result.data).toHaveLength(1)
    expect(result.data[0].repo_id).toBe('repo-1')
  })

  it('uses a prefetched site and never re-fetches it', async () => {
    const urls = stubFetchUrls(url => {
      if (url === '/api/deploy') return jsonResponse({ data: [deploymentFixture] })
      return jsonResponse({}, 404)
    })

    const result = await getSiteDeployments('site-1', { data: siteFixture, previewMode: false })

    expect(urls).toEqual(['/api/deploy'])
    expect(result.data).toHaveLength(1)
    expect(result.data[0].repo_id).toBe('repo-1')
  })

  it('accepts an in-flight getSite promise so callers can prefetch concurrently', async () => {
    const urls = stubFetchUrls(url => {
      if (url === '/api/sites/site-1') return jsonResponse(siteFixture)
      if (url === '/api/deploy') return jsonResponse({ data: [deploymentFixture] })
      return jsonResponse({}, 404)
    })

    const sitePromise = getSite('site-1')
    const result = await getSiteDeployments('site-1', sitePromise)

    expect(urls.filter(url => url === '/api/sites/site-1')).toHaveLength(1)
    expect(result.data).toHaveLength(1)
  })
})
