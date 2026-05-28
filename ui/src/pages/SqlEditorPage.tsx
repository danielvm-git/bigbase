import { useState } from 'react'
import { PageHeader, Button } from '../components'

interface SqlResult {
  columns: string[]
  rows: Record<string, unknown>[]
}

export default function SqlEditorPage() {
  const [query, setQuery] = useState("SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name")
  const [result, setResult] = useState<SqlResult | null>(null)
  const [error, setError] = useState('')
  const [running, setRunning] = useState(false)

  const handleRun = async () => {
    setError('')
    setResult(null)
    setRunning(true)

    try {
      const res = await fetch('/api/sql', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query }),
      })
      const data = await res.json()
      if (!res.ok) {
        setError(data.error || `error: ${res.status}`)
        return
      }
      setResult(data as SqlResult)
    } catch {
      setError('network error')
    } finally {
      setRunning(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault()
      handleRun()
    }
  }

  return (
    <div className="sql-editor">
      <PageHeader title="SQL Editor" />
      <div className="sql-layout">
        <div className="sql-input-area">
          <textarea
            className="sql-textarea"
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            rows={8}
            spellCheck={false}
            placeholder="Enter SQL query..."
          />
          <div className="sql-actions">
            <Button onClick={handleRun} disabled={running || !query.trim()}>
              {running ? 'Running...' : 'Run (⌘⏎)'}
            </Button>
          </div>
        </div>

        {error && <p className="sql-error">{error}</p>}

        {result && (
          <div className="sql-results">
            <p className="result-meta">{result.rows.length} row{result.rows.length !== 1 ? 's' : ''} returned</p>
            {result.columns.length > 0 && result.rows.length > 0 && (
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      {result.columns.map(col => (
                        <th key={col}>{col}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {result.rows.map((row, i) => (
                      <tr key={i}>
                        {result.columns.map(col => (
                          <td className="max" key={col}>{formatCell(row[col])}</td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function formatCell(val: unknown): string {
  if (val === null || val === undefined) return 'NULL'
  if (typeof val === 'object') return JSON.stringify(val)
  return String(val)
}
