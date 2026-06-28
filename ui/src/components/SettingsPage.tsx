import { type ReactNode } from 'react'

interface SettingsPageProps {
  title: string
  children: ReactNode
  subtitle?: string
  tabs?: ReactNode
  className?: string
}

interface SettingsSectionProps {
  title: string
  children: ReactNode
  description?: string
  actions?: ReactNode
  className?: string
}

export function SettingsPage({ title, subtitle, children, tabs, className = '' }: SettingsPageProps) {
  return (
    <div className={`page settings-page ${className}`.trim()}>
      <div className="page-header">
        <h1 className="page-title">{title}</h1>
        {subtitle && <p className="page-subtitle">{subtitle}</p>}
      </div>
      {tabs && <div className="settings-page-tabs">{tabs}</div>}
      <div className="settings-page-content">
        {children}
      </div>
    </div>
  )
}

export function SettingsSection({ title, children, description, actions, className = '' }: SettingsSectionProps) {
  return (
    <section className={`settings-section ${className}`.trim()}>
      <div className="settings-section-header">
        <div className="settings-section-info">
          <h2 className="settings-section-title">{title}</h2>
          {description && <p className="settings-section-description">{description}</p>}
        </div>
        {actions && <div className="settings-section-actions">{actions}</div>}
      </div>
      <div className="settings-section-content">
        {children}
      </div>
      <hr className="settings-divider" />
    </section>
  )
}
