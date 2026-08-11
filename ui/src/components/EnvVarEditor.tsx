import { useState, type ReactNode } from 'react'
import { Button, Input } from './index'

interface EnvVarRowOrigin {
  /** Original key the row was created with (survives renames). */
  key: string
  /** Original value at snapshot time. */
  value: string
  /** True when the row did not exist in the snapshot (added mid-edit). */
  added: boolean
}

interface EnvVarEditorProps {
  vars: Record<string, string>
  onChange: (vars: Record<string, string>) => void
}

export function EnvVarEditor({ vars, onChange }: EnvVarEditorProps): ReactNode {
  // Snapshot of the initial vars, keyed by the CURRENT row key so renames
  // keep pointing at the row's original key/value for the Revert action.
  const [origins, setOrigins] = useState<Record<string, EnvVarRowOrigin>>(() =>
    Object.fromEntries(Object.entries(vars).map(([key, value]) => [key, { key, value, added: false }])),
  )
  const entries = Object.entries(vars)

  const updateKey = (oldKey: string, newKey: string) => {
    if (oldKey === newKey) return
    // Guard against silent overwrite on key collision
    if (newKey in vars) return
    const copy = { ...vars }
    delete copy[oldKey]
    copy[newKey] = vars[oldKey]
    onChange(copy)
    setOrigins(prev => {
      const next = { ...prev }
      if (next[oldKey]) {
        next[newKey] = next[oldKey]
        delete next[oldKey]
      }
      return next
    })
  }

  const updateValue = (key: string, value: string) => {
    onChange({ ...vars, [key]: value })
  }

  const remove = (key: string) => {
    const copy = { ...vars }
    delete copy[key]
    onChange(copy)
    setOrigins(prev => {
      const next = { ...prev }
      delete next[key]
      return next
    })
  }

  const add = () => {
    const copy = { ...vars }
    let i = 1
    let next = `KEY_${i}`
    while (next in copy && i < 1000) { i++; next = `KEY_${i}` }
    if (i >= 1000) next = `KEY_${Date.now()}`
    copy[next] = ''
    onChange(copy)
    setOrigins(prev => ({ ...prev, [next]: { key: next, value: '', added: true } }))
  }

  // A row is dirty when its key or value differs from the snapshot, so the
  // Revert action is only offered while an edit is actually pending.
  const isDirty = (key: string, value: string): boolean => {
    const origin = origins[key]
    if (!origin) return false
    return origin.key !== key || origin.value !== value
  }

  const revert = (key: string) => {
    const origin = origins[key]
    if (!origin) return
    const copy = { ...vars }
    delete copy[key]
    if (!origin.added) {
      copy[origin.key] = origin.value
    }
    onChange(copy)
    setOrigins(prev => {
      const next = { ...prev }
      delete next[key]
      if (!origin.added) next[origin.key] = origin
      return next
    })
  }

  if (entries.length === 0) {
    return (
      <div data-testid="env-var-editor-empty">
        <p className="dim">No environment variables configured</p>
        <Button variant="secondary" size="sm" onClick={add}>Add Variable</Button>
      </div>
    )
  }

  return (
    <div data-testid="env-var-editor">
      {entries.map(([key, value]) => (
        <div key={key} className="form-row" style={{ marginBottom: 'var(--space-3)' }}>
          <Input
            placeholder="KEY"
            value={key}
            onChange={(e) => updateKey(key, e.target.value)}
            style={{ flex: 1 }}
          />
          <Input
            placeholder="VALUE"
            value={value}
            onChange={(e) => updateValue(key, e.target.value)}
            style={{ flex: 2 }}
          />
          {isDirty(key, value) && (
            <Button variant="secondary" size="sm" onClick={() => revert(key)}>Revert</Button>
          )}
          <Button variant="danger" size="sm" onClick={() => remove(key)}>Remove</Button>
        </div>
      ))}
      <Button variant="secondary" size="sm" onClick={add}>Add Variable</Button>
    </div>
  )
}
