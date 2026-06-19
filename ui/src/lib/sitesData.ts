import { isPreviewForced } from './previewMode'
import { mockDeployments, mockGitHubRepos, mockSites } from '../mocks/sites'
import type {
  Deployment,
  GitHubRepo,
  GitHubStatus,
  GitRepo,
  Site,
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
