import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader, Button, Input, Modal } from '../components'
import { useToast } from '../hooks/useToast'

type ColName = string
type ColRecord = { id: number } & Record<string, unknown>

interface SchemaColumn {
  name: string
  type: string
}

type StudioMode = 'data' | 'schema'

export default function DataStudioPage() {
  const navigate = useNavigate()
  const toast = useToast()
  const [collections, setCollections] = useState<ColName[]>([])
  const [selected, setSelected] = useState<string | null>(null)
  const [records, setRecords] = useState<ColRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [recordError, setRecordError] = useState('')
  const [mode, setMode] = useState<StudioMode>('data')
  const [columns, setColumns] = useState<SchemaColumn[]>([])
  const [modalOpen, setModalOpen] = useState(false)
  const [editCol, setEditCol] = useState<SchemaColumn | null>(null)
  const [colName, setColName] = useState('')
  const [colType, setColType] = useState('text')
  const [filterValue, setFilterValue] = useState('')
  const [sortField, setSortField] = useState('')

  useEffect(() => {
    fetch('/api/collections/')
      .then(r => r.json())
      .then(d => setCollections((d as { data: ColName[] }).data || []))
      .catch(() => setCollections([]))
      .finally(() => setLoading(false))
  }, [])

  const inferSchema = (recs: ColRecord[]): SchemaColumn[] => {
    if (recs.length === 0) return []
    const keys = Array.from(new Set(recs.flatMap(r => Object.keys(r))))
    return keys.map(name => ({
      name,
      type: name === 'id' ? 'integer' : typeof recs[0][name] === 'number' ? 'number' : 'text',
    }))
  }

  const loadRecords = async (name: string, filter?: string, sort?: string) => {
    setSelected(name)
    setRecordError('')
    try {
      const params = new URLSearchParams()
      if (filter && filter.trim()) params.set('filter', filter.trim())
      if (sort && sort.trim()) params.set('sort', sort.trim())
      const qs = params.toString()
      const url = `/api/collections/${name}${qs ? '?' + qs : ''}`
      const res = await fetch(url)
      if (!res.ok) {
        setRecordError(`error: ${res.status}`)
        setRecords([])
        setColumns([])
        return
      }
      const d = (await res.json()) as { data: ColRecord[] }
      const recs = d.data || []
      setRecords(recs)
      setColumns(inferSchema(recs))
    } catch {
      setRecordError('network error')
      setRecords([])
      setColumns([])
    }
  }

  const openAddColumn = () => {
    setEditCol(null)
    setColName('')
    setColType('text')
    setModalOpen(true)
  }

  const openEditColumn = (col: SchemaColumn) => {
    setEditCol(col)
    setColName(col.name)
    setColType(col.type)
    setModalOpen(true)
  }

  const saveColumn = () => {
    if (!colName.trim()) return
    if (editCol) {
      setColumns(prev => prev.map(c => c.name === editCol.name ? { name: colName, type: colType } : c))
      toast.show('Column updated (preview — DDL not wired)', 'info')
    } else {
      setColumns(prev => [...prev, { name: colName, type: colType }])
      toast.show('Column added (preview — DDL not wired)', 'info')
    }
    setModalOpen(false)
  }

  const deleteColumn = (name: string) => {
    if (!confirm(`Remove column "${name}" from schema view?`)) return
    setColumns(prev => prev.filter(c => c.name !== name))
    toast.show('Column removed (preview — DDL not wired)', 'info')
  }

  const queryThis = () => {
    if (!selected) return
    const quoted = `"${selected.replace(/"/g, '""')}"`
    navigate('/sql', { state: { collection: selected, hint: `SELECT * FROM ${quoted} LIMIT 100` } })
  }

  const allKeys = records.length > 0
    ? Array.from(new Set(records.flatMap(r => Object.keys(r))))
    : []

  if (loading) return <div className="loading">Loading collections...</div>

  return (
    <div className="data-studio">
      <PageHeader title="Data Studio">
        {selected && mode === 'schema' && (
          <Button variant="secondary" size="sm" onClick={queryThis}>Query this</Button>
        )}
      </PageHeader>

      {selected && mode === 'schema' && (
        <p className="dim" style={{ marginBottom: 'var(--space-6)' }}>
          Schema preview only — column changes are not persisted until DDL API ships.
        </p>
      )}

      {selected && (
        <div className="studio-mode-toggle">
          <Button variant={mode === 'data' ? 'primary' : 'secondary'} size="sm" onClick={() => setMode('data')}>Data</Button>
          <Button variant={mode === 'schema' ? 'primary' : 'secondary'} size="sm" onClick={() => setMode('schema')}>Schema</Button>
        </div>
      )}

      <div className="studio-layout">
        <aside className="collection-list">
          <div className="collection-list-title">Collections</div>
          {collections.length === 0 && <p className="dim">No collections yet.</p>}
          <ul className="collection-list-nav">
            {collections.map(c => (
              <li key={c}>
                <button
                  className={`collection-btn${selected === c ? ' active' : ''}`}
                  onClick={() => loadRecords(c, filterValue, sortField)}
                >
                  {c}
                </button>
              </li>
            ))}
          </ul>
        </aside>
        <section className="record-view">
          {!selected && <p className="dim">Select a collection to browse.</p>}
          {recordError && <p className="input-error-text">{recordError}</p>}

          {selected && mode === 'schema' && !recordError && (
            <>
              <div className="schema-actions">
                <Button variant="primary" size="sm" onClick={openAddColumn}>Add column</Button>
                <Button variant="secondary" size="sm" onClick={queryThis}>Query this</Button>
              </div>
              {columns.length === 0 ? (
                <p className="dim">No columns inferred yet.</p>
              ) : (
                <div className="table-wrap">
                  <table>
                    <thead>
                      <tr><th>Name</th><th>Type</th><th>Actions</th></tr>
                    </thead>
                    <tbody>
                      {columns.map(col => (
                        <tr key={col.name}>
                          <td><code>{col.name}</code></td>
                          <td>{col.type}</td>
                          <td className="actions-cell">
                            <Button variant="ghost" size="sm" onClick={() => openEditColumn(col)}>Edit</Button>
                            <Button variant="danger" size="sm" onClick={() => deleteColumn(col.name)}>Delete</Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}

          {selected && mode === 'data' && !recordError && records.length === 0 && (
            <p className="dim">No records found.</p>
          )}
          {selected && mode === 'data' && !recordError && (
            <div style={{ display: 'flex', gap: 'var(--space-4)', marginBottom: 'var(--space-4)', flexWrap: 'wrap' }}>
              <Input
                aria-label="Filter records"
                placeholder="filter: field=value or field:gt=value"
                value={filterValue}
                onChange={e => { setFilterValue(e.target.value); if (selected) loadRecords(selected, e.target.value, sortField) }}
                style={{ flex: 2, minWidth: 200 }}
              />
              <Input
                aria-label="Sort records"
                placeholder="sort: field or -field"
                value={sortField}
                onChange={e => { setSortField(e.target.value); if (selected) loadRecords(selected, filterValue, e.target.value) }}
                style={{ flex: 1, minWidth: 150 }}
              />
            </div>
          )}
          {selected && mode === 'data' && records.length > 0 && (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>{allKeys.map(k => <th key={k}>{k}</th>)}</tr>
                </thead>
                <tbody>
                  {records.map(r => (
                    <tr key={r.id}>
                      {allKeys.map(k => (
                        <td className="max" key={k}>{JSON.stringify(r[k] ?? '')}</td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>

      <Modal open={modalOpen} onClose={() => setModalOpen(false)} title={editCol ? 'Edit column' : 'Add column'}>
        <Input label="Column name" placeholder="Column name" value={colName} onChange={e => setColName(e.target.value)} />
        <Input as="select" aria-label="Column type" value={colType} onChange={e => setColType(e.target.value)} style={{ marginTop: 'var(--space-4)' }}>
          <option value="text">text</option>
          <option value="integer">integer</option>
          <option value="number">number</option>
          <option value="boolean">boolean</option>
        </Input>
        <div className="form-actions" style={{ marginTop: 'var(--space-6)' }}>
          <Button onClick={saveColumn}>Save</Button>
          <Button variant="secondary" onClick={() => setModalOpen(false)}>Cancel</Button>
        </div>
      </Modal>
    </div>
  )
}
