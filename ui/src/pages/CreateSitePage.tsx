import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  PageHeader,
  Button,
  Input,
  Card,
  CardHeader,
  ChoiceCard,
  WizardSteps,
  PreviewBanner,
  Badge,
  statusBadgeVariant,
} from '../components'
import {
  createSite,
  getGitHubRepos,
  getGitHubStatus,
  getGitRepos,
  githubInstallURL,
} from '../lib/sitesData'
import { isPreviewForced, previewQuerySuffix } from '../lib/previewMode'
import type { GitHubRepo, GitRepo, SiteSource } from '../types/sites'

const WIZARD_STEPS = ['Source', 'Configure', 'Review', 'Deploy']

export default function CreateSitePage() {
  const nav = useNavigate()
  const [searchParams] = useSearchParams()
  const preRepo = searchParams.get('repo_id') ?? ''

  const [step, setStep] = useState(1)
  const [source, setSource] = useState<SiteSource>(preRepo ? 'existing' : 'github')
  const [previewMode, setPreviewMode] = useState(false)
  const [ghConnected, setGhConnected] = useState(false)
  const [ghRepos, setGhRepos] = useState<GitHubRepo[]>([])
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

  useEffect(() => {
    const init = async () => {
      const force = isPreviewForced()
      setPreviewMode(force)
      const st = await getGitHubStatus()
      setGhConnected(st.data.connected)
      setPreviewMode(p => p || st.previewMode)
      const repos = await getGitHubRepos()
      setGhRepos(repos.data)
      setPreviewMode(p => p || repos.previewMode)
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

  const filteredGh = ghRepos.filter(r =>
    r.full_name.toLowerCase().includes(repoFilter.toLowerCase()),
  )

  const canNextSource = () => {
    if (source === 'github') return !!selectedGh || (previewMode && ghRepos.length > 0)
    if (source === 'existing') return !!selectedLocalId
    return false
  }

  const goConfigure = () => {
    if (source === 'github' && selectedGh) {
      setName(selectedGh.full_name.split('/').pop() ?? selectedGh.full_name)
      setBranch(selectedGh.default_branch || 'main')
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
    setDeploying(true)
    setStep(4)

    const result = await createSite({
      source: source === 'github' ? 'github' : 'existing',
      name,
      branch,
      root_path: rootPath,
      git_repo_id: source === 'existing' ? selectedLocalId : undefined,
      github_repo_id: source === 'github' ? selectedGh?.id : undefined,
      github_full_name: source === 'github' ? selectedGh?.full_name : undefined,
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
      setStep(3)
      return
    }

    const dep = result.deployment ?? result.site?.latest_deployment
    setDoneSiteId(result.site?.id ?? selectedLocalId)
    setDoneStatus(dep?.status ?? 'building')
    setDoneUrl(dep?.url ?? '')
    setDeploying(false)

    if (dep?.status === 'building' || dep?.status === 'pending') {
      const poll = setInterval(async () => {
        try {
          const res = await fetch(`/api/deploy/${dep?.id}`)
          if (!res.ok) return
          const d = await res.json()
          setDoneStatus(d.status)
          if (d.url) setDoneUrl(d.url)
          if (d.status === 'running' || d.status === 'failed') clearInterval(poll)
        } catch { /* ignore */ }
      }, 3000)
    }
  }

  const pq = previewQuerySuffix()

  return (
    <div className="wizard">
      <PageHeader title="Create site">
        <Button variant="secondary" size="sm" onClick={() => nav(`/deploy${pq}`)}>
          Cancel
        </Button>
      </PageHeader>

      {(previewMode || isPreviewForced()) && <PreviewBanner />}

      <WizardSteps steps={WIZARD_STEPS} current={step} />

      {step === 1 && (
        <div className="wizard-panel">
          <h2 className="section-title">How do you want to add your app?</h2>
          <div className="choice-grid">
            <ChoiceCard
              icon="⎇"
              title="Connect GitHub"
              description="Import from a repository on GitHub. Recommended for ongoing deploys."
              badge="Recommended"
              selected={source === 'github'}
              onClick={() => setSource('github')}
            />
            <ChoiceCard
              icon="◆"
              title="BigBase git repo"
              description="Deploy code already hosted on this server's git service."
              selected={source === 'existing'}
              onClick={() => setSource('existing')}
            />
            <ChoiceCard
              icon="▣"
              title="Clone template"
              description="Start from a starter template."
              disabled
            />
            <ChoiceCard
              icon="↑"
              title="Manual upload"
              description="Upload a tarball of your built site."
              disabled
            />
          </div>

          {source === 'github' && (
            <div style={{ marginTop: 'var(--space-12)' }}>
              {!ghConnected && !previewMode && (
                <Card>
                  <CardHeader title="Connect GitHub" />
                  <p className="dim" style={{ marginBottom: 'var(--space-6)' }}>
                    Install the BigBase GitHub App to list and deploy your repositories.
                  </p>
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={() => { window.location.href = githubInstallURL() }}
                  >
                    Connect GitHub
                  </Button>
                </Card>
              )}
              {(ghConnected || previewMode) && (
                <>
                  <Input
                    placeholder="Search repositories…"
                    value={repoFilter}
                    onChange={e => setRepoFilter(e.target.value)}
                  />
                  <div className="repo-picker" style={{ marginTop: 'var(--space-4)' }}>
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
                        <span className="repo-picker-name">{r.full_name}</span>
                        {r.description && <span className="dim">{r.description}</span>}
                      </button>
                    ))}
                    {filteredGh.length === 0 && (
                      <p className="dim" style={{ padding: 'var(--space-6)' }}>No repositories match.</p>
                    )}
                  </div>
                </>
              )}
            </div>
          )}

          {source === 'existing' && (
            <div className="repo-picker" style={{ marginTop: 'var(--space-12)' }}>
              {localRepos.length === 0 && (
                <p className="dim">No git repos yet. Create one under Git Repos first.</p>
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
                  <span className="repo-picker-name">{r.name}</span>
                  <span className="dim">default: {r.default_branch}</span>
                </button>
              ))}
            </div>
          )}

          <div className="wizard-actions">
            <Button variant="primary" size="sm" disabled={!canNextSource()} onClick={goConfigure}>
              Continue
            </Button>
          </div>
        </div>
      )}

      {step === 2 && (
        <div className="wizard-panel">
          <h2 className="section-title">Configure your site</h2>
          <div className="card">
            <div className="form-row" style={{ flexDirection: 'column', alignItems: 'stretch' }}>
              <Input label="Site name" value={name} onChange={e => setName(e.target.value)} required />
              <Input label="Production branch" value={branch} onChange={e => setBranch(e.target.value)} />
              <Input label="Root directory" value={rootPath} onChange={e => setRootPath(e.target.value)} />
            </div>
          </div>
          <div className="wizard-actions">
            <Button variant="secondary" size="sm" onClick={() => setStep(1)}>Back</Button>
            <Button variant="primary" size="sm" disabled={!name.trim()} onClick={() => setStep(3)}>
              Continue
            </Button>
          </div>
        </div>
      )}

      {step === 3 && (
        <div className="wizard-panel">
          <h2 className="section-title">Review and deploy</h2>
          <Card>
            <CardHeader title={name} />
            <p><strong>Source:</strong> {source === 'github' ? selectedGh?.full_name : localRepos.find(r => r.id === selectedLocalId)?.name}</p>
            <p><strong>Branch:</strong> {branch}</p>
            <p><strong>Root:</strong> <code>{rootPath}</code></p>
            <p style={{ marginTop: 'var(--space-6)' }}>
              <Badge variant="neutral">Stack detected after build</Badge>
              <span className="dim" style={{ marginLeft: 'var(--space-4)' }}>Node, Go, Python, or static</span>
            </p>
          </Card>
          {error && <p className="input-error-text">{error}</p>}
          <div className="wizard-actions">
            <Button variant="secondary" size="sm" onClick={() => setStep(2)}>Back</Button>
            <Button variant="primary" size="sm" onClick={handleDeploy}>Deploy</Button>
          </div>
        </div>
      )}

      {step === 4 && (
        <div className="wizard-panel">
          <Card>
            <div className="deploy-progress-card">
              {deploying && <div className="spinner" aria-hidden />}
              <h2 className="section-title" style={{ marginTop: 0 }}>
                {deploying ? 'Building your site…' : 'Deployment ready'}
              </h2>
              <Badge variant={statusBadgeVariant(doneStatus)}>{doneStatus}</Badge>
              {doneUrl && (
                <p style={{ marginTop: 'var(--space-8)' }}>
                  <Button as="a" href={doneUrl} target="_blank" rel="noreferrer" variant="primary" size="sm">
                    Open app
                  </Button>
                </p>
              )}
              {!deploying && doneSiteId && (
                <div className="wizard-actions" style={{ justifyContent: 'center', marginTop: 'var(--space-12)' }}>
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
        </div>
      )}
    </div>
  )
}
