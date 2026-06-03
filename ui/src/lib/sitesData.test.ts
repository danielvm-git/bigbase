import { describe, expect, it, vi, afterEach } from 'vitest'
import { getGitHubRepos } from './sitesData'

describe('getGitHubRepos', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns error when API responds 500', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 502,
          json: () => Promise.resolve({ error: 'failed to list repositories from GitHub' }),
        }),
      ),
    )
    const result = await getGitHubRepos()
    expect(result.data).toEqual([])
    expect(result.previewMode).toBe(false)
    expect(result.error).toContain('repositories')
  })
})
