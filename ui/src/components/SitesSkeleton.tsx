import { SkeletonCard } from './SkeletonCard'

export function SitesListSkeleton() {
  return (
    <div className="sites-skeleton-list" aria-busy="true" aria-label="Loading sites">
      <SkeletonCard count={3} />
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
