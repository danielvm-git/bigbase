import { describe, expect, it, vi, afterEach } from 'vitest'
import { getGitHubRepos } from './sitesData'

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
