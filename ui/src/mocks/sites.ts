import type { Deployment, GitHubRepo, Site } from '../types/sites'

export const mockSites: Site[] = [
  {
    id: 'site-demo-1',
    name: 'marketing-site',
    full_name: 'acme/marketing-site',
    git_repo_id: 'a1b2c3d4',
    production_branch: 'main',
    root_path: './',
    github_connected: true,
    latest_deployment: {
      id: 'dep-1',
      repo_id: 'a1b2c3d4',
      branch: 'main',
      commit_sha: 'abc1234',
      status: 'running',
      url: 'http://localhost:10042',
      port: 10042,
      app_type: 'static',
      created_at: new Date(Date.now() - 3600000).toISOString(),
    },
  },
  {
    id: 'site-demo-2',
    name: 'api-dashboard',
    full_name: 'acme/api-dashboard',
    git_repo_id: 'e5f6g7h8',
    production_branch: 'main',
    root_path: './',
    github_connected: true,
    latest_deployment: {
      id: 'dep-2',
      repo_id: 'e5f6g7h8',
      branch: 'main',
      commit_sha: 'def5678',
      status: 'building',
      url: '',
      port: 10088,
      app_type: 'node',
      created_at: new Date().toISOString(),
    },
  },
]

export const mockGitHubRepos: GitHubRepo[] = [
  { id: 1, full_name: 'acme/marketing-site', default_branch: 'main', private: false, description: 'Public marketing site' },
  { id: 2, full_name: 'acme/api-dashboard', default_branch: 'main', private: true, description: 'Internal dashboard' },
  { id: 3, full_name: 'acme/docs', default_branch: 'main', private: false },
]

export const mockDeployments: Deployment[] = [
  {
    id: 'dep-1',
    repo_id: 'a1b2c3d4',
    branch: 'main',
    commit_sha: 'abc1234',
    status: 'running',
    url: 'http://localhost:10042',
    port: 10042,
    app_type: 'static',
    created_at: new Date(Date.now() - 86400000).toISOString(),
  },
  {
    id: 'dep-2',
    repo_id: 'a1b2c3d4',
    branch: 'main',
    commit_sha: '999aaaa',
    status: 'failed',
    url: '',
    port: 0,
    app_type: 'node',
    created_at: new Date(Date.now() - 172800000).toISOString(),
  },
  {
    id: 'dep-3',
    repo_id: 'a1b2c3d4',
    branch: 'develop',
    commit_sha: 'bbbcccc',
    status: 'running',
    url: 'http://localhost:10099',
    port: 10099,
    app_type: 'node',
    created_at: new Date(Date.now() - 7200000).toISOString(),
  },
]
