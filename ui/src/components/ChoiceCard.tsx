import type { ReactNode } from 'react'

interface ChoiceCardProps {
  icon: ReactNode
  title: string
  description: string
  selected?: boolean
  disabled?: boolean
  badge?: string
  onClick?: () => void
}

export function ChoiceCard({ icon, title, description, selected, disabled, badge, onClick }: ChoiceCardProps) {
  return (
    <button
      type="button"
      className={[
        'choice-card',
        selected ? 'choice-card--selected' : '',
        disabled ? 'choice-card--disabled' : '',
      ].filter(Boolean).join(' ')}
      onClick={disabled ? undefined : onClick}
      disabled={disabled}
      title={disabled ? 'Coming soon' : undefined}
    >
      <span className="choice-card-icon">{icon}</span>
      <span className="choice-card-body">
        <span className="choice-card-title">
          {title}
          {badge && <span className="choice-card-badge">{badge}</span>}
        </span>
        <span className="choice-card-desc">{description}</span>
      </span>
    </button>
  )
}
