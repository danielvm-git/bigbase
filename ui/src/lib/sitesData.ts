import { isPreviewForced } from './previewMode'
import { mockDeployments, mockGitHubRepos, mockSites } from '../mocks/sites'
import type {
  CacheStats,
  Deployment,
  EnvVar,
  GitHubRepo,
  GitHubStatus,
  GitRepo,
  RollbackEvent,
  Site,
  SiteCacheStatus,
  SiteDeployKey,
  SiteDomain,
  SitesDataResult,
} from '../types/sites'

async function fetchJSON<T>(url: string): Promise<{ ok: boolean; status: number; data: T }> {
  try {
    const res = await fetch(url)
    const data = (await res.json()) as T
    return { ok: res.ok, status: res.status, data }
  } catch {
    return { ok: false, status: 0, data: {} as T }
  }
}

export async function getGitHubStatus(): Promise<SitesDataResult<GitHubStatus>> {
  if (isPreviewForced()) {
    return { data: { connected: false, configured: false }, previewMode: true }
  }
  const { ok, status, data } = await fetchJSON<GitHubStatus>('/api/github/status')
  if (!ok && (status === 404 || status === 0)) {
    return { data: { connected: false, configured: false }, previewMode: true }
  }
  return { data, previewMode: false }
}

export async function getSites(): Promise<SitesDataResult<Site[]>> {
  if (isPreviewForced()) {
    return { data: mockSites, previewMode: true }
  }
  const { ok, status, data } = await fetchJSON<{ data: Site[] }>('/api/sites')
  if (ok && data.data) {
    return { data: data.data, previewMode: false }
  }
  if (status === 404 || status === 0) {
    return await getSitesFromLegacyAPI()
  }
  return { data: [], previewMode: false }
}

async function getSitesFromLegacyAPI(): Promise<SitesDataResult<Site[]>> {
  const [depRes, repoRes] = await Promise.all([
    fetchJSON<{ data: Deployment[] }>('/api/deploy'),
    fetchJSON<{ data: GitRepo[] }>('/api/git/repos'),
  ])
  if (!depRes.ok && !repoRes.ok) {
    return { data: isPreviewForced() ? mockSites : [], previewMode: true }
  }
  const repos = repoRes.ok ? repoRes.data.data || [] : []
  const deps = depRes.ok ? depRes.data.data || [] : []
  const byRepo = new Map<string, Deployment>()
  for (const d of deps) {
    const cur = byRepo.get(d.repo_id)
    if (!cur || new Date(d.created_at) > new Date(cur.created_at)) {
      byRepo.set(d.repo_id, d)
    }
  }
  const sites: Site[] = repos.map(r => ({
    id: r.id,
    name: r.name,
    full_name: r.name,
    git_repo_id: r.id,
    production_branch: r.default_branch || 'main',
    root_path: './',
    latest_deployment: byRepo.get(r.id),
  }))
  for (const d of deps) {
    if (!sites.some(s => s.git_repo_id === d.repo_id)) {
      sites.push({
        id: d.repo_id,
        name: d.repo_id.slice(0, 8),
        full_name: d.repo_id.slice(0, 8),
        git_repo_id: d.repo_id,
        production_branch: d.branch,
        root_path: './',
        latest_deployment: d,
      })
    }
  }
  return { data: sites, previewMode: !depRes.ok || !repoRes.ok }
}

export async function getSite(siteId: string): Promise<SitesDataResult<Site | null>> {
  if (isPreviewForced()) {
    const s = mockSites.find(x => x.id === siteId) ?? mockSites[0]
    return { data: s ?? null, previewMode: true }
  }
  const { ok, data } = await fetchJSON<Site>(`/api/sites/${siteId}`)
  if (ok && data && 'id' in data) {
    return { data, previewMode: false }
  }
  const all = await getSites()
  const found = all.data.find(s => s.id === siteId)
  return { data: found ?? null, previewMode: all.previewMode }
}

export async function getSiteDeployments(siteId: string): Promise<SitesDataResult<Deployment[]>> {
  if (isPreviewForced()) {
    return { data: mockDeployments, previewMode: true }
  }
  const site = await getSite(siteId)
  const repoId = site.data?.git_repo_id ?? siteId
  const { ok, data } = await fetchJSON<{ data: Deployment[] }>('/api/deploy')
  if (!ok) {
    return { data: isPreviewForced() ? mockDeployments : [], previewMode: true }
  }
  const filtered = (data.data || []).filter(d => d.repo_id === repoId)
  return { data: filtered.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()), previewMode: false }
}

export async function getGitRepos(): Promise<SitesDataResult<GitRepo[]>> {
  if (isPreviewForced()) {
    return {
      data: mockSites.map(s => ({
        id: s.git_repo_id,
        name: s.name,
        default_branch: s.production_branch,
      })),
      previewMode: true,
    }
  }
  const { ok, data } = await fetchJSON<{ data: GitRepo[] }>('/api/git/repos')
  if (ok) return { data: data.data || [], previewMode: false }
  return { data: [], previewMode: true }
}

export async function getGitHubRepos(): Promise<SitesDataResult<GitHubRepo[]>> {
  if (isPreviewForced()) {
    return { data: mockGitHubRepos, previewMode: true }
  }
  const { ok, status, data } = await fetchJSON<{ data: GitHubRepo[]; error?: string }>(
    '/api/github/repos',
  )
  if (ok) return { data: data.data || [], previewMode: false }
  const body = data && typeof data === 'object' ? data : null
  const apiCode = body && 'code' in body ? String(body.code) : ''
  const apiError = body && 'error' in body && body.error ? String(body.error) : ''
  if (status === 0) {
    return { data: [], previewMode: false, error: 'Could not reach the server. Try again.' }
  }
  if (status === 404 && (apiCode === 'github_not_installed' || apiError)) {
    return { data: [], previewMode: false, error: apiError || 'GitHub App is not installed' }
  }
  if (status === 404) {
    return { data: mockGitHubRepos, previewMode: true }
  }
  if (apiCode === 'github_api_error' || apiError) {
    return { data: [], previewMode: false, error: apiError || 'Could not load GitHub repositories. Try reconnecting.' }
  }
  return { data: [], previewMode: false, error: 'Could not load GitHub repositories. Try reconnecting.' }
}

export interface CreateSiteInput {
  source: 'github' | 'existing'
  name: string
  branch: string
  root_path: string
  git_repo_id?: string
  github_repo_id?: number
  github_full_name?: string
}

export interface CreateSiteResult {
  site?: Site
  deployment?: Deployment
  previewMode: boolean
  error?: string
}

export async function createSite(input: CreateSiteInput): Promise<CreateSiteResult> {
  if (isPreviewForced()) {
    return {
      previewMode: true,
      site: {
        id: 'site-preview-new',
        name: input.name,
        full_name: input.github_full_name ?? input.name,
        git_repo_id: 'preview-repo',
        production_branch: input.branch,
        root_path: input.root_path,
        latest_deployment: {
          id: 'dep-preview',
          repo_id: 'preview-repo',
          branch: input.branch,
          commit_sha: '',
          status: 'building',
          url: '',
          port: 10001,
          app_type: 'node',
          created_at: new Date().toISOString(),
        },
      },
    }
  }

  if (input.source === 'github' && input.github_repo_id) {
    const connectRes = await fetch('/api/github/repos/connect', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        github_repo_id: input.github_repo_id,
        full_name: input.github_full_name,
        branch: input.branch,
      }),
    })
    const connectBody = await connectRes.json()
    if (!connectRes.ok) {
      return { previewMode: false, error: connectBody.error || 'connect failed' }
    }
    input.git_repo_id = connectBody.git_repo_id as string
  }

  const res = await fetch('/api/sites', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: input.name,
      git_repo_id: input.git_repo_id,
      production_branch: input.branch,
      root_path: input.root_path,
      github_full_name: input.github_full_name,
    }),
  })
  const body = await res.json()
  if (!res.ok) {
    if (res.status === 404) {
      const depRes = await fetch('/api/deploy', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ repo_id: input.git_repo_id, branch: input.branch }),
      })
      const depBody = await depRes.json()
      if (!depRes.ok) {
        return { previewMode: false, error: depBody.error || 'deploy failed' }
      }
      return {
        previewMode: false,
        deployment: depBody as Deployment,
        site: {
          id: input.git_repo_id!,
          name: input.name,
          full_name: input.github_full_name ?? input.name,
          git_repo_id: input.git_repo_id!,
          production_branch: input.branch,
          root_path: input.root_path,
          latest_deployment: depBody as Deployment,
        },
      }
    }
    return { previewMode: false, error: body.error || 'create failed' }
  }
  return { previewMode: false, site: body as Site, deployment: (body as Site).latest_deployment }
}

export async function rollbackDeployment(deploymentID: string): Promise<{ ok: boolean; event?: RollbackEvent; error?: string }> {
  if (isPreviewForced()) {
    return {
      ok: true,
      event: {
        id: 'rollback-preview',
        site_id: 'preview',
        rolled_back_from: 'dep-old',
        rolled_back_to: deploymentID,
        created_at: new Date().toISOString(),
      },
    }
  }
  try {
    const res = await fetch(`/api/deploy/${deploymentID}/rollback`, { method: 'POST' })
    const body = await res.json().catch(() => ({ error: '' })) as { error?: string } & Partial<RollbackEvent>
    if (!res.ok) {
      return { ok: false, error: body.error || `Rollback failed (HTTP ${res.status})` }
    }
    const event: RollbackEvent = {
      id: body.id || '',
      site_id: body.site_id || '',
      rolled_back_from: body.rolled_back_from || '',
      rolled_back_to: body.rolled_back_to || '',
      created_at: body.created_at || '',
    }
    return { ok: true, event }
  } catch {
    return { ok: false, error: 'Rollback failed — network error' }
  }
}

export async function getRollbackEvents(siteId: string): Promise<{ ok: boolean; events: RollbackEvent[] }> {
  if (isPreviewForced()) {
    return {
      ok: true,
      events: [
        {
          id: 'rb-preview-1',
          site_id: siteId,
          rolled_back_from: 'dep-old',
          rolled_back_to: 'dep-current',
          created_at: new Date(Date.now() - 3600000).toISOString(),
        },
      ],
    }
  }
  try {
    const res = await fetch(`/api/deploy/${siteId}/rollback-events`)
    const body = await res.json().catch(() => ({ error: '', data: [] as RollbackEvent[] })) as { data?: RollbackEvent[] }
    if (res.ok && body.data) {
      return { ok: true, events: body.data }
    }
    return { ok: false, events: [] }
  } catch {
    return { ok: false, events: [] }
  }
}

export function githubInstallURL(): string {
  return '/api/github/install'
}

export async function deleteSite(siteId: string): Promise<{ ok: boolean; error?: string }> {
  if (isPreviewForced()) {
    return { ok: true }
  }
  try {
    const res = await fetch(`/api/sites/${siteId}`, { method: 'DELETE' })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
      return { ok: false, error: body.error || `Delete failed (HTTP ${res.status})` }
    }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Delete failed — network error' }
  }
}

export interface SiteManifestResult {
  exists: boolean
  content: string
}

export async function getSiteManifest(siteId: string): Promise<SitesDataResult<SiteManifestResult>> {
  if (isPreviewForced()) {
    return {
      data: {
        exists: true,
        content: `version: 1
framework: static
build:
  command: "npm run build"
start:
  command: "serve"
  port: 3000`,
      },
      previewMode: true,
    }
  }
  const { ok, data } = await fetchJSON<SiteManifestResult>(`/api/sites/${siteId}/manifest`)
  if (ok && data) {
    return { data, previewMode: false }
  }
  return { data: { exists: false, content: '' }, previewMode: false }
}

export async function saveSiteManifest(siteId: string, content: string): Promise<{ ok: boolean; error?: string; previewMode: boolean }> {
  if (isPreviewForced()) {
    return { ok: true, previewMode: true }
  }
  try {
    const res = await fetch(`/api/sites/${siteId}/manifest`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
    })
    const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
    if (!res.ok) {
      return { ok: false, error: body.error || `Save failed (HTTP ${res.status})`, previewMode: false }
    }
    return { ok: true, previewMode: false }
  } catch {
    return { ok: false, error: 'Save failed — network error', previewMode: false }
  }
}

export async function getEnvVars(siteId: string): Promise<EnvVar[]> {
  const { ok, data } = await fetchJSON<{ data: EnvVar[] }>(`/api/sites/${siteId}/env-vars`)
  if (ok && data.data) return data.data
  return []
}

export async function createEnvVar(
  siteId: string,
  payload: { key: string; value: string; is_build_time: boolean; is_runtime: boolean },
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(`/api/sites/${siteId}/env-vars`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
    if (!res.ok) return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

export async function updateEnvVar(
  siteId: string,
  key: string,
  payload: { value: string; is_build_time: boolean; is_runtime: boolean },
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(`/api/sites/${siteId}/env-vars/${key}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
    if (!res.ok) return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

export async function deleteEnvVar(siteId: string, key: string): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(`/api/sites/${siteId}/env-vars/${key}`, { method: 'DELETE' })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
      return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

// --- Custom domains (e46) ---

export async function getDomains(siteId: string): Promise<SiteDomain[]> {
  const { ok, data } = await fetchJSON<{ data: SiteDomain[] }>(`/api/sites/${siteId}/domains`)
  if (ok && data.data) return data.data
  return []
}

export async function addDomain(
  siteId: string,
  domain: string,
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(`/api/sites/${siteId}/domains`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain }),
    })
    const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
    if (!res.ok) return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

export async function verifyDomain(
  siteId: string,
  domain: string,
): Promise<{ ok: boolean; verified?: boolean; error?: string }> {
  try {
    const res = await fetch(`/api/sites/${siteId}/domains/${encodeURIComponent(domain)}/verify`, { method: 'POST' })
    const body = await res.json().catch(() => ({ error: '' })) as { verified?: boolean; error?: string }
    if (!res.ok) return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    return { ok: true, verified: body.verified }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

export async function deleteDomain(
  siteId: string,
  domain: string,
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(`/api/sites/${siteId}/domains/${encodeURIComponent(domain)}`, { method: 'DELETE' })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
      return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

// --- Build cache (e42s02) ---

const emptySiteCache: SiteCacheStatus = { entries: [], total_size_bytes: 0, total_hits: 0 }

export async function getSiteCache(siteId: string): Promise<{ status: SiteCacheStatus; ok: boolean }> {
  if (isPreviewForced()) {
    return {
      status: {
        entries: [
          { key: 'a1b2c3d4e5f6', site_id: siteId, repo_id: 'repo', branch: 'main', size: 134217728, hit_count: 12, created_at: new Date(Date.now() - 86400000).toISOString() },
        ],
        total_size_bytes: 134217728,
        total_hits: 12,
      },
      ok: true,
    }
  }
  const { ok, data } = await fetchJSON<SiteCacheStatus>(`/api/deploy/cache/site/${siteId}`)
  if (ok && data && Array.isArray(data.entries)) return { status: data, ok: true }
  return { status: emptySiteCache, ok: false }
}

export async function clearSiteCache(siteId: string): Promise<{ ok: boolean; error?: string }> {
  if (isPreviewForced()) return { ok: true }
  try {
    const res = await fetch(`/api/deploy/cache/site/${siteId}`, { method: 'DELETE' })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
      return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

export async function getCacheStats(): Promise<CacheStats> {
  if (isPreviewForced()) {
    return { total_entries: 14, total_size_bytes: 1288490188, max_size_bytes: 2147483648 }
  }
  const { ok, data } = await fetchJSON<CacheStats>('/api/deploy/cache')
  if (ok && data && typeof data.total_size_bytes === 'number') return data
  return { total_entries: 0, total_size_bytes: 0, max_size_bytes: 0 }
}

export async function clearAllCache(): Promise<{ ok: boolean; error?: string }> {
  if (isPreviewForced()) return { ok: true }
  try {
    const res = await fetch('/api/deploy/cache', { method: 'DELETE' })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
      return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

export async function pruneCache(maxAgeDays: number): Promise<{ ok: boolean; pruned?: number; error?: string }> {
  if (isPreviewForced()) return { ok: true, pruned: 0 }
  try {
    const res = await fetch('/api/deploy/cache/prune', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ max_age_days: maxAgeDays }),
    })
    const body = await res.json().catch(() => ({ error: '' })) as { error?: string; pruned?: number }
    if (!res.ok) return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    return { ok: true, pruned: body.pruned }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

export async function setCacheMaxSize(maxSizeBytes: number): Promise<{ ok: boolean; error?: string }> {
  if (isPreviewForced()) return { ok: true }
  try {
    const res = await fetch('/api/deploy/cache/config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ max_size_bytes: maxSizeBytes }),
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
      return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

// --- Site deploy keys (e74s02) ---

export async function getDeployKeys(siteId: string): Promise<SiteDeployKey[]> {
  const { ok, data } = await fetchJSON<{ data: SiteDeployKey[] }>(`/api/sites/${siteId}/deploy-keys`)
  if (ok && data.data) return data.data
  return []
}

export interface CreateDeployKeyResult {
  key_id: string
  name: string
  key: string
  prefix: string
  created_at: string
}

export async function createDeployKey(
  siteId: string,
  name: string,
): Promise<{ ok: boolean; data?: CreateDeployKeyResult; error?: string }> {
  try {
    const res = await fetch(`/api/sites/${siteId}/deploy-keys`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    const body = await res.json().catch(() => ({ error: '' })) as { data?: CreateDeployKeyResult; error?: string }
    if (!res.ok) return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    return { ok: true, data: body.data }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

export async function revokeDeployKey(
  siteId: string,
  keyId: string,
): Promise<{ ok: boolean; error?: string }> {
  try {
    const res = await fetch(`/api/sites/${siteId}/deploy-keys/${keyId}`, { method: 'DELETE' })
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: '' })) as { error?: string }
      return { ok: false, error: body.error || `Failed (HTTP ${res.status})` }
    }
    return { ok: true }
  } catch {
    return { ok: false, error: 'Network error' }
  }
}

