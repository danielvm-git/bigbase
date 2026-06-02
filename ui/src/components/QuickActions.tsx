import type { ReactNode } from 'react'
import { Button } from './index'

interface QuickAction {
  icon: string
  label: string
  link: string
  disabled?: boolean
}

interface QuickActionsProps {
  actions: QuickAction[]
  onAction: (link: string) => void
}

export function QuickActions({ actions, onAction }: QuickActionsProps): ReactNode {
  if (actions.length === 0) {
    return (
      <div data-testid="quick-actions-empty">
        <p className="dim">No quick actions</p>
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', gap: 'var(--space-4)', marginBottom: 'var(--space-8)', flexWrap: 'wrap' }}
      data-testid="quick-actions">
      {actions.map((a) => (
        <Button
          key={a.label}
          variant={a.disabled ? 'ghost' : 'primary'}
          size="sm"
          onClick={() => onAction(a.link)}
          disabled={a.disabled}
        >
          {a.icon} {a.label}
        </Button>
      ))}
    </div>
  )
}
