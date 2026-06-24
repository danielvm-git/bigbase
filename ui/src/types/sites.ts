export type DeploymentStatus = 'pending' | 'building' | 'deploying' | 'running' | 'failed'

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
}

export interface StatusTransition {
  from: string
  to: string
  timestamp: string
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
  value: string
  is_build_time: boolean
  is_runtime: boolean
  created_at: string
  updated_at: string
}

export type SiteSource = 'github' | 'existing' | 'empty' | 'template'

export interface SitesDataResult<T> {
  data: T
  previewMode: boolean
  error?: string
}
