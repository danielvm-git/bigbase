export type DeploymentStatus = 'pending' | 'building' | 'deploying' | 'running' | 'draining' | 'stopped' | 'failed' | 'rolled_back' | 'replaced'

export interface Deployment {
  id: string
  repo_id: string
  branch: string
  commit_sha: string
  status: DeploymentStatus | string
  url: string
  port: number
  app_type: string
  created_at: string
  error_message?: string
  status_history?: StatusTransition[]
  health_summary?: string
}

export interface StatusTransition {
  from: string
  to: string
  timestamp: string
}

export interface RollbackEvent {
  id: string
  site_id: string
  rolled_back_from: string
  rolled_back_to: string
  created_at: string
}

export interface GitRepo {
  id: string
  name: string
  default_branch: string
  description?: string
}

export interface GitHubRepo {
  id: number
  full_name: string
  default_branch: string
  private: boolean
  description?: string
}

export interface GitHubStatus {
  connected: boolean
  configured: boolean
  login?: string
}

export interface Site {
  id: string
  name: string
  full_name: string
  git_repo_id: string
  production_branch: string
  root_path: string
  latest_deployment?: Deployment
  github_connected?: boolean
}

export interface EnvVar {
  id: string
  site_id: string
  key: string
  value?: string         // present only on create/update responses
  value_preview: string  // server-masked (e.g. ••••XXXX), always present
  is_build_time: boolean
  is_runtime: boolean
  created_at: string
  updated_at: string
}

export type SiteSource = 'github' | 'existing' | 'empty' | 'template'

export interface CacheEntry {
  key: string
  site_id: string
  repo_id: string
  branch: string
  size: number
  hit_count: number
  created_at: string
}

export interface SiteCacheStatus {
  entries: CacheEntry[]
  total_size_bytes: number
  total_hits: number
}

export interface CacheStats {
  total_entries: number
  total_size_bytes: number
  max_size_bytes: number
}

export interface SiteDomain {
  id: string
  site_id: string
  domain: string
  verify_token: string
  verified_at: string | null
  created_at: string
  // Computed / display-only fields
  cert_status?: string
  cert_expires_at?: string | null
}

export interface SitesDataResult<T> {
  data: T
  previewMode: boolean
  error?: string
}
