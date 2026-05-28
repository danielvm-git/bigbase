import { useEffect, useState } from 'react'
import { PageHeader } from '../components'

type ColName = string
type ColRecord = { id: number } & Record<string, unknown>

export default function DataStudioPage() {
  const [collections, setCollections] = useState<ColName[]>([])
  const [selected, setSelected] = useState<string | null>(null)
  const [records, setRecords] = useState<ColRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [recordError, setRecordError] = useState('')

  useEffect(() => {
    fetch('/api/collections/')
      .then(r => r.json())
      .then(d => setCollections((d as { data: ColName[] }).data || []))
      .catch(() => setCollections([]))
      .finally(() => setLoading(false))
  }, [])

  const loadRecords = async (name: string) => {
    setSelected(name)
    setRecordError('')
    try {
      const res = await fetch(`/api/collections/${name}`)
      if (!res.ok) {
        setRecordError(`error: ${res.status}`)
        setRecords([])
        return
      }
      const d = (await res.json()) as { data: ColRecord[] }
      setRecords(d.data || [])
    } catch {
      setRecordError('network error')
      setRecords([])
    }
  }

  const allKeys = records.length > 0
    ? Array.from(new Set(records.flatMap(r => Object.keys(r))))
    : []

  if (loading) return <div className="loading">Loading collections...</div>

  return (
    <div className="data-studio">
      <PageHeader title="Data Studio" />
      <div className="studio-layout">
        <aside className="collection-list">
          <div className="collection-list-title">Collections</div>
          {collections.length === 0 && <p className="dim">No collections yet.</p>}
          <ul className="collection-list-nav">
            {collections.map(c => (
              <li key={c}>
                <button
                  className={`collection-btn${selected === c ? ' active' : ''}`}
                  onClick={() => loadRecords(c)}
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
          {selected && !recordError && records.length === 0 && (
            <p className="dim">No records found.</p>
          )}
          {selected && records.length > 0 && (
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
    </div>
  )
}
