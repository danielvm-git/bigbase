import { type ReactNode } from 'react'
import { Button, Input } from './index'

interface EnvVarEditorProps {
  vars: Record<string, string>
  onChange: (vars: Record<string, string>) => void
}

export function EnvVarEditor({ vars, onChange }: EnvVarEditorProps): ReactNode {
  const entries = Object.entries(vars)

  const updateKey = (oldKey: string, newKey: string) => {
    if (oldKey === newKey) return
    // Guard against silent overwrite on key collision
    if (newKey in vars) return
    const copy = { ...vars }
    delete copy[oldKey]
    copy[newKey] = vars[oldKey]
    onChange(copy)
  }

  const updateValue = (key: string, value: string) => {
    onChange({ ...vars, [key]: value })
  }

  const remove = (key: string) => {
    const copy = { ...vars }
    delete copy[key]
    onChange(copy)
  }

  const add = () => {
    const copy = { ...vars }
    let i = 1
    let next = `KEY_${i}`
    while (next in copy && i < 1000) { i++; next = `KEY_${i}` }
    if (i >= 1000) next = `KEY_${Date.now()}`
    copy[next] = ''
    onChange(copy)
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
          <Button variant="danger" size="sm" onClick={() => remove(key)}>Remove</Button>
        </div>
      ))}
      <Button variant="secondary" size="sm" onClick={add}>Add Variable</Button>
    </div>
  )
}
