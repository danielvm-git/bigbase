import { useEffect, useState, useRef } from 'react'
import { PageHeader, Button, Modal } from '../components'

interface FileObj {
  id: string
  name: string
  size: number
  mime_type: string
  created_at: string
}

function fmtSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export default function StoragePage() {
  const [files, setFiles] = useState<FileObj[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [uploading, setUploading] = useState(false)
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('list')
  const [previewFile, setPreviewFile] = useState<FileObj | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  const fetchFiles = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/storage/files')
      const d = await res.json()
      if (!res.ok) { setError(d.error || `error: ${res.status}`); setFiles([]) }
      else { setFiles((d as { data: FileObj[] }).data || []) }
    } catch { setError('network error') }
    finally { setLoading(false) }
  }

  // eslint-disable-next-line react-hooks/set-state-in-effect
  useEffect(() => { fetchFiles() }, [])

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault()
    const input = fileRef.current
    if (!input || !input.files?.length) return
    setUploading(true); setError('')
    const fd = new FormData()
    fd.append('file', input.files[0])
    try {
      const res = await fetch('/api/storage/upload', { method: 'POST', body: fd })
      const d = await res.json()
      if (!res.ok) { setError(d.error || 'upload failed'); setUploading(false); return }
      input.value = ''
      fetchFiles()
    } catch { setError('network error') }
    finally { setUploading(false) }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this file?')) return
    try {
      const res = await fetch(`/api/storage/files/${id}`, { method: 'DELETE' })
      if (!res.ok) { const d = await res.json(); setError(d.error || 'delete failed'); return }
      setFiles(prev => prev.filter(f => f.id !== id))
    } catch { setError('network error') }
  }

  if (loading) return <div className="loading">Loading files...</div>

  return (
    <div>
      <PageHeader title="Storage">
        <div style={{ display: 'flex', gap: 'var(--space-4)' }}>
          <Button variant={viewMode === 'list' ? 'primary' : 'secondary'} size="sm" onClick={() => setViewMode('list')}>List</Button>
          <Button variant={viewMode === 'grid' ? 'primary' : 'secondary'} size="sm" onClick={() => setViewMode('grid')}>Grid</Button>
          <Button variant="secondary" size="sm" onClick={fetchFiles}>Refresh</Button>
        </div>
      </PageHeader>
      {error && <p className="input-error-text">{error}</p>}

      <div className="card" style={{ marginBottom: 'var(--space-8)' }}>
        <form onSubmit={handleUpload} className="upload-form">
          <input type="file" ref={fileRef} required aria-label="Upload file" style={{ fontSize: 'var(--text-s)' }} />
          <Button type="submit" size="sm" disabled={uploading}>
            {uploading ? 'Uploading...' : 'Upload'}
          </Button>
        </form>
      </div>

      {files.length === 0 && !error && <p className="dim">No files uploaded.</p>}
      {files.length > 0 && viewMode === 'list' && (
        <div className="table-wrap">
          <table>
            <caption className="visually-hidden">Stored files</caption>
            <thead>
              <tr>
                <th scope="col">Name</th>
                <th scope="col">Size</th>
                <th scope="col">Type</th>
                <th scope="col">Uploaded</th>
                <th scope="col">Actions</th>
              </tr>
            </thead>
            <tbody>
              {files.map(f => (
                <tr key={f.id}>
                  <td>
                    {f.mime_type.startsWith('image/') ? (
                      <button
                        type="button"
                        className="btn-reset"
                        style={{ cursor: 'pointer', color: 'var(--fg-accent)' }}
                        onClick={() => setPreviewFile(f)}
                      >
                        {f.name}
                      </button>
                    ) : (
                      <span>{f.name}</span>
                    )}
                  </td>
                  <td>{fmtSize(f.size)}</td>
                  <td><code>{f.mime_type}</code></td>
                  <td>{new Date(f.created_at).toLocaleString()}</td>
                  <td className="actions-cell">
                    <a href={`/api/storage/files/${f.id}`} className="btn btn-secondary btn-sm" download={f.name} style={{ marginRight: 'var(--space-2)' }}>Download</a>
                    <Button variant="danger" size="sm" onClick={() => handleDelete(f.id)}>Delete</Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {files.length > 0 && viewMode === 'grid' && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 'var(--space-6)' }}>
          {files.map(f => {
            const isImage = f.mime_type.startsWith('image/')
            return (
              <button
                key={f.id}
                type="button"
                className="card btn-reset"
                style={{ display: 'block', width: '100%', padding: 'var(--space-6)', cursor: isImage ? 'pointer' : 'default', textAlign: 'center' }}
                onClick={() => isImage && setPreviewFile(f)}
                disabled={!isImage}
              >
                {isImage ? (
                  <img
                    src={`/api/storage/files/${f.id}/thumbnail?w=150&h=150`}
                    alt={f.name}
                    style={{ width: '100%', height: 120, objectFit: 'cover', borderRadius: 'var(--radius-s)', marginBottom: 'var(--space-3)' }}
                  />
                ) : (
                  <div aria-hidden="true" style={{ width: '100%', height: 120, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-subtle)', borderRadius: 'var(--radius-s)', marginBottom: 'var(--space-3)', fontSize: '2rem' }}>
                    📄
                  </div>
                )}
                <div style={{ fontSize: 'var(--text-xs)', fontWeight: 500, wordBreak: 'break-word' }}>{f.name}</div>
                <div style={{ fontSize: 'var(--text-xs)', color: 'var(--fg-tertiary)' }}>{fmtSize(f.size)}</div>
              </button>
            )
          })}
        </div>
      )}

      <Modal
        open={!!previewFile}
        onClose={() => setPreviewFile(null)}
        title={previewFile?.name ?? 'Image preview'}
      >
        {previewFile && (
          <img
            src={`/api/storage/files/${previewFile.id}`}
            alt={previewFile.name}
            style={{ maxWidth: '100%', maxHeight: '70vh', borderRadius: 'var(--radius-s)' }}
          />
        )}
      </Modal>
    </div>
  )
}
