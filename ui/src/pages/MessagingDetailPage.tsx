import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { PageHeader, Button, Input, Breadcrumb, Tabs } from '../components'
import { getTemplate } from '../mocks/templates'

const detailTabs = [
  { id: 'editor', label: 'Editor' },
  { id: 'preview', label: 'Preview' },
]

export default function MessagingDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const template = id ? getTemplate(id) : undefined
  const [tab, setTab] = useState('editor')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [variables, setVariables] = useState('')

  useEffect(() => {
    if (!template) return
    setSubject(template.subject)
    setBody(template.body)
    setVariables(template.variables.join(', '))
  }, [template])

  if (!template) {
    return (
      <div>
        <PageHeader title="Template not found" />
        <Button variant="secondary" onClick={() => navigate('/messaging')}>Back to templates</Button>
      </div>
    )
  }

  const previewBody = body
    .replace(/\{\{app_name\}\}/g, 'BigBase')
    .replace(/\{\{user_name\}\}/g, 'Alex')
    .replace(/\{\{reset_link\}\}/g, 'https://example.com/reset')
    .replace(/\{\{code\}\}/g, '123456')
    .replace(/\{\{sender\}\}/g, 'Team')

  return (
    <div>
      <Breadcrumb items={[
        { label: 'Messaging', to: '/messaging' },
        { label: template.name },
      ]} />
      <PageHeader title={template.name}>
        <Button variant="secondary" size="sm" onClick={() => navigate('/messaging')}>Back</Button>
        <Button variant="primary" size="sm" onClick={() => navigate('/messaging?tab=send')}>Send test</Button>
      </PageHeader>
      <p className="dim" style={{ marginBottom: 'var(--space-6)' }}>
        Preview only — edits are not saved until the template API ships.
      </p>

      <Tabs tabs={detailTabs} active={tab} onChange={setTab} />

      {tab === 'editor' && (
        <div className="card">
          <Input placeholder="Subject" value={subject} onChange={e => setSubject(e.target.value)} />
          <Input as="textarea" placeholder="Body" value={body} onChange={e => setBody(e.target.value)} rows={8} style={{ marginTop: 'var(--space-4)' }} />
          <Input placeholder="Variables (comma-separated)" value={variables} onChange={e => setVariables(e.target.value)} style={{ marginTop: 'var(--space-4)' }} />
          <p className="dim" style={{ marginTop: 'var(--space-4)' }}>
            Type: {template.type} · Status: {template.status}
          </p>
        </div>
      )}

      {tab === 'preview' && (
        <div className="template-preview-grid">
          <div className="card">
            <h3 style={{ marginBottom: 'var(--space-4)' }}>Rendered preview</h3>
            {subject && <p><strong>{subject.replace(/\{\{app_name\}\}/g, 'BigBase')}</strong></p>}
            <pre style={{ whiteSpace: 'pre-wrap', fontFamily: 'var(--font-sans)' }}>{previewBody}</pre>
          </div>
          <div className="card">
            <h3 style={{ marginBottom: 'var(--space-4)' }}>Variables</h3>
            <ul>
              {variables.split(',').map(v => v.trim()).filter(Boolean).map(v => (
                <li key={v}><code>{'{{' + v + '}}'}</code></li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </div>
  )
}
