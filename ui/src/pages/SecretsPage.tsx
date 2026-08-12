import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Breadcrumb, Button, PageHeader, ProjectSecretsTab } from '../components'
import { getProject, listProjectEnvironments, listProjects } from '../lib/secretsData'
import type { Project, ProjectEnvironment } from '../types/secrets'

// SecretsPage — Project → Environment → Folder navigation for native project
// secrets (e89s05). Three route levels:
//   /secrets                     → organization Projects
//   /secrets/:projectId          → that Project's Environments
//   /secrets/:projectId/:envId   → secret management in the default Folder

export default function SecretsPage() {
  const { projectId, envId } = useParams<{ projectId?: string; envId?: string }>()

  const [projects, setProjects] = useState<Project[]>([])
  const [project, setProject] = useState<Project | null>(null)
  const [environments, setEnvironments] = useState<ProjectEnvironment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      setLoading(true); setError(null)
      try {
        if (!projectId) {
          const items = await listProjects()
          if (!cancelled) setProjects(items)
        } else if (!envId) {
          const [proj, envs] = await Promise.all([getProject(projectId), listProjectEnvironments(projectId)])
          if (!cancelled) { setProject(proj); setEnvironments(envs) }
        } else {
          const [proj, envs] = await Promise.all([getProject(projectId), listProjectEnvironments(projectId)])
          if (!cancelled) { setProject(proj); setEnvironments(envs) }
        }
      } catch {
        if (!cancelled) setError('Could not load data. Try again.')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void load()
    return () => { cancelled = true }
  }, [projectId, envId])

  const env = envId ? environments.find(e => e.id === envId) ?? null : null

  if (loading) return <div className="loading">Loading…</div>
  if (error) return <p className="input-error-text" role="alert">{error}</p>

  // Level 2: manage secrets in the Project Environment's default folder.
  if (projectId && envId) {
    return (
      <div>
        <PageHeader title="Project Secrets">
          <Breadcrumb
            items={[
              { label: 'Projects', to: '/secrets' },
              { label: project?.name ?? projectId, to: `/secrets/${projectId}` },
              { label: env?.name ?? env?.slug ?? envId },
              { label: 'default' },
            ]}
          />
        </PageHeader>
        <ProjectSecretsTab projectId={projectId} envId={envId} />
      </div>
    )
  }

  // Level 1: pick an Environment.
  if (projectId) {
    return (
      <div>
        <PageHeader title={project?.name ?? 'Project'}>
          <Breadcrumb
            items={[
              { label: 'Projects', to: '/secrets' },
              { label: project?.name ?? projectId },
            ]}
          />
        </PageHeader>
        {environments.length === 0 ? (
          <p className="dim">No environments in this project yet.</p>
        ) : (
          <div className="table-wrap">
            <table>
              <caption className="visually-hidden">Project environments</caption>
              <thead>
                <tr>
                  <th scope="col">Environment</th>
                  <th scope="col">Slug</th>
                  <th scope="col">Actions</th>
                </tr>
              </thead>
              <tbody>
                {environments.map(e => (
                  <tr key={e.id}>
                    <td>{e.name}</td>
                    <td><code style={{ fontSize: 'var(--text-sm)' }}>{e.slug}</code></td>
                    <td>
                      <Link to={`/secrets/${projectId}/${e.id}`} className="btn btn-secondary btn-sm">
                        Open secrets
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    )
  }

  // Level 0: pick a Project.
  return (
    <div>
      <PageHeader title="Project Secrets" subtitle="Manage reusable secrets scoped to a Project and Environment.">
        <Button
          variant="secondary"
          size="sm"
          onClick={() => { setLoading(true); setError(null); void listProjects().then(setProjects).finally(() => setLoading(false)) }}
        >
          Refresh
        </Button>
      </PageHeader>
      {projects.length === 0 ? (
        <p className="dim">No projects found. Create a project through the API first.</p>
      ) : (
        <div className="table-wrap">
          <table>
            <caption className="visually-hidden">Organization projects</caption>
            <thead>
              <tr>
                <th scope="col">Project</th>
                <th scope="col">Created</th>
                <th scope="col">Actions</th>
              </tr>
            </thead>
            <tbody>
              {projects.map(p => (
                <tr key={p.id}>
                  <td>{p.name}</td>
                  <td style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)' }}>
                    {new Date(p.created_at).toLocaleDateString()}
                  </td>
                  <td>
                    <Link to={`/secrets/${p.id}`} className="btn btn-secondary btn-sm">
                      Open environments
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
