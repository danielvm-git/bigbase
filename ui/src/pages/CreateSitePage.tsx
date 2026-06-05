import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import {
  PageHeader,
  Button,
  Input,
  Card,
  ChoiceCard,
  WizardSteps,
  PreviewBanner,
  Badge,
  statusBadgeVariant,
  Breadcrumb,
  StreamLog,
} from '../components'
import { Icon } from '../components/Icon'
import {
  createSite,
  getGitHubRepos,
  getGitHubStatus,
  getGitRepos,
  githubInstallURL,
} from '../lib/sitesData'
import { isPreviewForced, previewQuerySuffix } from '../lib/previewMode'
import { siteDisplayUrl } from '../lib/format'
import { deployWizardTitle } from '../lib/deployWizard'
import { useDeployLogs } from '../lib/useDeployLogs'
import type { GitHubRepo, GitRepo, SiteSource } from '../types/sites'

const WIZARD_STEPS = ['Source', 'Configure', 'Deploy']

const PREVIEW_DEPLOY_LINES = [
  '→ Deployment started (branch: main)',
  '→ Status: building',
  '→ Cloning repository (branch: main)',
  '→ Clone complete',
  '→ Detected app type: static',
  '→ Serving static files',
  '→ Deployed at http://localhost:10001',
]

export default function CreateSitePage() {
  const nav = useNavigate()
  const [searchParams] = useSearchParams()
  const preRepo = searchParams.get('repo_id') ?? ''

  const [step, setStep] = useState(1)
  const [source, setSource] = useState<SiteSource>(preRepo ? 'existing' : 'github')
  const [previewMode, setPreviewMode] = useState(false)
  const [ghConnected, setGhConnected] = useState(false)
  const [ghConfigured, setGhConfigured] = useState(false)
  const [ghLogin, setGhLogin] = useState<string | undefined>()
  const [ghRepos, setGhRepos] = useState<GitHubRepo[]>([])
  const [ghReposError, setGhReposError] = useState('')
  const [ghLoading, setGhLoading] = useState(true)
  const [localRepos, setLocalRepos] = useState<GitRepo[]>([])
  const [selectedGh, setSelectedGh] = useState<GitHubRepo | null>(null)
  const [selectedLocalId, setSelectedLocalId] = useState(preRepo)
  const [name, setName] = useState('')
  const [branch, setBranch] = useState('main')
  const [rootPath, setRootPath] = useState('./')
  const [repoFilter, setRepoFilter] = useState('')
  const [error, setError] = useState('')
  const [deploying, setDeploying] = useState(false)
  const [doneSiteId, setDoneSiteId] = useState('')
  const [doneUrl, setDoneUrl] = useState('')
  const [doneStatus, setDoneStatus] = useState('building')
  const [doneError, setDoneError] = useState('')
  const [deploymentId, setDeploymentId] = useState('')
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const { lines: deployLines, errorMessage: logError, fetchError: logFetchError, isStreaming } = useDeployLogs({
    deploymentId: previewMode ? '' : deploymentId,
    enabled: deploying && !previewMode,
    deployStatus: doneStatus,
    previewLines: previewMode ? PREVIEW_DEPLOY_LINES : undefined,
  })

  useEffect(() => {
    if (logError) setDoneError(logError)
  }, [logError])

  const loadGitHub = async () => {
    setGhLoading(true)
    setGhReposError('')
    const st = await getGitHubStatus()
    setGhConnected(st.data.connected)
    setGhConfigured(st.data.configured)
    setGhLogin(st.data.login)
    setPreviewMode(p => p || st.previewMode)
    const repos = await getGitHubRepos()
    setGhRepos(repos.data)
    setGhReposError(repos.error ?? '')
    setPreviewMode(p => p || repos.previewMode)
    setGhLoading(false)
  }

  useEffect(() => {
    const init = async () => {
      const force = isPreviewForced()
      setPreviewMode(force)
      await loadGitHub()
      const local = await getGitRepos()
      setLocalRepos(local.data)
      if (preRepo) {
        const r = local.data.find(x => x.id === preRepo)
        if (r) {
          setName(r.name)
          setBranch(r.default_branch || 'main')
        }
      }
    }
    init()
  }, [preRepo])

  const ghReposUnavailable =
    ghConnected && !ghLoading && ghRepos.length === 0 && !ghReposError
  const showGitHubConnect =
    source === 'github' &&
    !previewMode &&
    (!ghConnected || !!ghReposError || !ghConfigured || ghReposUnavailable)
  const showGitHubRepoPicker =
    source === 'github' &&
    (previewMode || (ghConnected && !ghReposError && !ghReposUnavailable))

  const filteredGh = ghRepos.filter(r =>
    r.full_name.toLowerCase().includes(repoFilter.toLowerCase()),
  )

  const canNextSource = () => {
    if (source === 'github') return !!selectedGh || (previewMode && ghRepos.length > 0)
    if (source === 'existing') return !!selectedLocalId
    return false
  }

  const goConfigure = () => {
    if (source === 'github') {
      const repo = selectedGh ?? (previewMode && ghRepos[0] ? ghRepos[0] : null)
      if (repo) {
        setSelectedGh(repo)
        setName(repo.full_name.split('/').pop() ?? repo.full_name)
        setBranch(repo.default_branch || 'main')
      }
    }
    if (source === 'existing') {
      const r = localRepos.find(x => x.id === selectedLocalId)
      if (r) {
        setName(r.name)
        setBranch(r.default_branch || 'main')
      }
    }
    setStep(2)
  }

  const handleDeploy = async () => {
    setError('')
    setDoneError('')
    setDeploying(true)
    setDeploymentId('')
    setDoneStatus('building')
    setDoneUrl('')
    setStep(3)
    if (pollRef.current) {
      clearInterval(pollRef.current)
      pollRef.current = null
    }

    const gh = selectedGh ?? (previewMode ? ghRepos[0] : null)

    const result = await createSite({
      source: source === 'github' ? 'github' : 'existing',
      name,
      branch,
      root_path: rootPath,
      git_repo_id: source === 'existing' ? selectedLocalId : undefined,
      github_repo_id: source === 'github' ? gh?.id : undefined,
      github_full_name: source === 'github' ? gh?.full_name : undefined,
    })

    if (result.previewMode) {
      setDoneSiteId('site-preview-new')
      setDoneStatus('building')
      setTimeout(() => {
        setDoneStatus('running')
        setDoneUrl('http://localhost:10001')
        setDeploying(false)
      }, 2000)
      return
    }

    if (result.error) {
      setError(result.error)
      setDeploying(false)
      setStep(2)
      return
    }

    const dep = result.deployment ?? result.site?.latest_deployment
    setDoneSiteId(result.site?.id ?? selectedLocalId)
    const initialStatus = dep?.status ?? 'building'
    setDoneStatus(initialStatus)
    setDoneUrl(dep?.url ?? '')
    if (dep?.id) setDeploymentId(dep.id)
    if (initialStatus === 'running' || initialStatus === 'failed') {
      setDeploying(false)
      if (initialStatus === 'failed' && dep?.error_message) {
        setDoneError(dep.error_message)
      }
    }

    if (dep?.id && (initialStatus === 'building' || initialStatus === 'pending')) {
      pollRef.current = setInterval(async () => {
        try {
          const res = await fetch(`/api/deploy/${dep.id}`)
          if (!res.ok) return
          const d = (await res.json()) as {
            status?: string
            url?: string
            error_message?: string
          }
          if (d.status) setDoneStatus(d.status)
          if (d.url) setDoneUrl(d.url)
          if (d.error_message) setDoneError(d.error_message)
          if (d.status === 'running' || d.status === 'failed') {
            setDeploying(false)
            if (pollRef.current) {
              clearInterval(pollRef.current)
              pollRef.current = null
            }
          }
        } catch { /* ignore */ }
      }, 3000)
    }
  }

  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current)
    }
  }, [])

  const pq = previewQuerySuffix()
  const sourceLabel =
    source === 'github'
      ? (selectedGh ?? ghRepos[0])?.full_name
      : localRepos.find(r => r.id === selectedLocalId)?.name

  return (
    <div className={`wizard${step === 3 ? ' wizard--deploy' : ''}`}>
      <Breadcrumb
        items={[
          { label: 'Sites', to: `/deploy${pq}` },
          { label: 'Create site' },
        ]}
      />

      <PageHeader title="Create a new site">
        <Button variant="ghost" size="sm" onClick={() => nav(`/deploy${pq}`)}>
          <Icon name="x" size={16} />
          Cancel
        </Button>
      </PageHeader>

      {(previewMode || isPreviewForced()) && <PreviewBanner />}

      <WizardSteps steps={WIZARD_STEPS} current={step} />

      {step === 1 && (
        <div className="wizard-panel">
          <div className="wizard-intro">
            <h2 className="wizard-intro-title">Where&apos;s your code?</h2>
            <p className="wizard-intro-desc">
              Pick a source. We detect the stack and build it for you.
            </p>
          </div>

          <div className="choice-grid">
            <ChoiceCard
              icon={<Icon name="github" size={22} />}
              title="Connect Git"
              description="Deploy from GitHub. Redeploys automatically on push."
              selected={source === 'github'}
              onClick={() => setSource('github')}
            />
            <ChoiceCard
              icon={<Icon name="box" size={22} />}
              title="Existing BigBase repo"
              description="Use a repository already on this instance."
              selected={source === 'existing'}
              onClick={() => setSource('existing')}
            />
            <ChoiceCard
              icon={<Icon name="rocket" size={22} />}
              title="Start from template"
              description="Astro, Next.js & more."
              badge="Coming soon"
              disabled
            />
          </div>

          {source === 'github' && (
            <div className="source-panel">
              {showGitHubConnect && (
                <Card>
                  {ghReposError && (
                    <p className="input-error-text" style={{ marginBottom: 'var(--space-6)' }}>
                      {ghReposError}
                    </p>
                  )}
                  <div className="github-connect-panel">
                    <div className="choice-card-icon">
                      <Icon name="github" size={26} />
                    </div>
                    <h3 className="github-connect-title">
                      {ghReposError ? 'Reconnect GitHub' : 'Connect your GitHub account'}
                    </h3>
                    <p className="github-connect-desc">
                      Choose which repositories BigBase can deploy from your GitHub account.
                    </p>
                    <p className="github-connect-perms">
                      We read repo metadata and listen for pushes to trigger builds—no write access to your code.
                    </p>
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={() => { window.location.href = githubInstallURL() }}
                    >
                      <Icon name="github" size={16} />
                      {ghReposError ? 'Reconnect GitHub' : 'Authorize GitHub'}
                    </Button>
                  </div>
                </Card>
              )}
              {showGitHubRepoPicker && (
                <Card>
                  {ghLoading && <p className="dim">Loading repositories…</p>}
                  {!ghLoading && (
                    <>
                      {ghLogin && (
                        <p className="dim" style={{ marginBottom: 'var(--space-4)' }}>
                          Connected as <strong>{ghLogin}</strong>
                        </p>
                      )}
                      <div className="input-with-prefix" style={{ marginBottom: 'var(--space-4)' }}>
                        <span className="input-prefix" aria-hidden>
                          <Icon name="search" size={14} />
                        </span>
                        <Input
                          placeholder="Search repositories…"
                          value={repoFilter}
                          onChange={e => setRepoFilter(e.target.value)}
                          aria-label="Search repositories"
                        />
                      </div>
                      <div className="repo-picker">
                        {filteredGh.map(r => (
                          <button
                            key={r.id}
                            type="button"
                            className={[
                              'repo-picker-item',
                              selectedGh?.id === r.id ? 'repo-picker-item--selected' : '',
                            ].filter(Boolean).join(' ')}
                            onClick={() => setSelectedGh(r)}
                          >
                            <div className="repo-picker-row">
                              <Icon name="git-branch" size={16} className="dim" />
                              <span className="repo-picker-name">{r.full_name}</span>
                              {r.private && <Badge variant="neutral">Private</Badge>}
                              {selectedGh?.id === r.id && <Badge variant="accent">Selected</Badge>}
                            </div>
                            {r.description && (
                              <span className="repo-picker-desc">{r.description}</span>
                            )}
                          </button>
                        ))}
                        {filteredGh.length === 0 && ghRepos.length === 0 && (
                          <p className="repo-picker-empty">
                            No repos yet—grant access on GitHub, then reconnect.
                          </p>
                        )}
                        {filteredGh.length === 0 && ghRepos.length > 0 && (
                          <p className="repo-picker-empty">
                            {repoFilter
                              ? `No repositories match "${repoFilter}".`
                              : 'No repositories match your search.'}
                          </p>
                        )}
                      </div>
                      {!previewMode && (
                        <Button
                          variant="link"
                          size="sm"
                          style={{ marginTop: 'var(--space-4)' }}
                          onClick={() => { window.location.href = githubInstallURL() }}
                        >
                          Manage repository access on GitHub
                        </Button>
                      )}
                    </>
                  )}
                </Card>
              )}
            </div>
          )}

          {source === 'existing' && (
            <Card>
              <div className="repo-picker">
                {localRepos.length === 0 && (
                  <p className="repo-picker-empty">
                    No git repos yet.{' '}
                    <Link to="/repos">Create one under Git Repos</Link> first.
                  </p>
                )}
                {localRepos.map(r => (
                  <button
                    key={r.id}
                    type="button"
                    className={[
                      'repo-picker-item',
                      selectedLocalId === r.id ? 'repo-picker-item--selected' : '',
                    ].filter(Boolean).join(' ')}
                    onClick={() => setSelectedLocalId(r.id)}
                  >
                    <div className="repo-picker-row">
                      <Icon name="box" size={16} className="dim" />
                      <span className="repo-picker-name">{r.name}</span>
                      {selectedLocalId === r.id && <Badge variant="accent">Selected</Badge>}
                    </div>
                    <span className="repo-picker-desc">on this instance · default: {r.default_branch}</span>
                  </button>
                ))}
              </div>
            </Card>
          )}

          <div className="wizard-actions">
            <Button variant="secondary" size="sm" onClick={() => nav(`/deploy${pq}`)}>
              Cancel
            </Button>
            <Button variant="primary" size="sm" disabled={!canNextSource()} onClick={goConfigure}>
              Continue →
            </Button>
          </div>
        </div>
      )}

      {step === 2 && (
        <div className="wizard-panel">
          <div className="wizard-intro">
            <h2 className="wizard-intro-title">Configure your site</h2>
            <p className="wizard-intro-desc">
              Review settings, then deploy. URL preview:{' '}
              <span className="mono" style={{ fontFamily: 'var(--font-mono)' }}>
                {siteDisplayUrl(name || 'your-site')}
              </span>
            </p>
          </div>
          <div className="card">
            <div className="form-row" style={{ flexDirection: 'column', alignItems: 'stretch' }}>
              <Input label="Site name" value={name} onChange={e => setName(e.target.value)} required />
              <Input label="Production branch" value={branch} onChange={e => setBranch(e.target.value)} />
              <Input label="Root directory" value={rootPath} onChange={e => setRootPath(e.target.value)} />
            </div>
          </div>
          <Card>
            <p><strong>Source:</strong> {sourceLabel}</p>
            <p><strong>Branch:</strong> {branch}</p>
            <p><strong>Root:</strong> <code>{rootPath}</code></p>
            <p style={{ marginTop: 'var(--space-6)' }}>
              <Badge variant="neutral">Stack detected after build</Badge>
              <span className="dim" style={{ marginLeft: 'var(--space-4)' }}>
                Node, Go, Python, or static
              </span>
            </p>
          </Card>
          {error && <p className="input-error-text">{error}</p>}
          <div className="wizard-actions">
            <Button variant="secondary" size="sm" onClick={() => setStep(1)}>
              ← Back
            </Button>
            <Button variant="primary" size="sm" disabled={!name.trim()} onClick={handleDeploy}>
              Deploy
            </Button>
          </div>
        </div>
      )}

      {step === 3 && (
        <div className="wizard-panel">
          <div className="deploy-step-layout">
            <Card className="deploy-status-card">
              <div className="deploy-progress-card deploy-progress-card--split">
                {deploying && <div className="spinner" aria-hidden />}
                <h2 className="section-title" style={{ marginTop: 0 }}>
                  {deployWizardTitle(deploying, doneStatus)}
                </h2>
                <p className="dim">
                  {siteDisplayUrl(name)} · {branch}
                </p>
                <Badge variant={statusBadgeVariant(doneStatus)}>{doneStatus}</Badge>
                {doneError && (
                  <p className="input-error-text deploy-status-error">
                    {doneError}
                  </p>
                )}
                {doneUrl && doneStatus === 'running' && (
                  <p className="deploy-status-actions">
                    <Button as="a" href={doneUrl} target="_blank" rel="noreferrer" variant="primary" size="sm">
                      Open app
                    </Button>
                  </p>
                )}
                {!deploying && doneSiteId && (
                  <div className="wizard-actions deploy-status-footer">
                    <Button variant="secondary" size="sm" onClick={() => nav(`/deploy/${doneSiteId}${pq}`)}>
                      View site
                    </Button>
                    <Button variant="primary" size="sm" onClick={() => nav(`/deploy${pq}`)}>
                      All sites
                    </Button>
                  </div>
                )}
              </div>
            </Card>
            <div className="deploy-log-panel">
              <div className="deploy-log-header">
                <span className="deploy-log-title">Build output</span>
                {(deploying || isStreaming) && (
                  <span className="deploy-log-live" aria-live="polite">Live</span>
                )}
              </div>
              {logFetchError && (
                <p className="dim deploy-log-fetch-error">{logFetchError}</p>
              )}
              <StreamLog
                logs={deployLines}
                isStreaming={deploying || isStreaming}
                className="deploy-log-terminal"
              />
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
