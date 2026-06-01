export function SitesListSkeleton() {
  return (
    <div className="sites-skeleton-list" aria-busy="true" aria-label="Loading sites">
      {[1, 2, 3].map(i => (
        <div key={i} className="skeleton site-card-skeleton" />
      ))}
    </div>
  )
}

export function WizardSkeleton() {
  return (
    <div className="wizard-panel">
      <div className="skeleton skeleton-title" />
      <div className="skeleton skeleton-block" />
      <div className="skeleton skeleton-block short" />
    </div>
  )
}
