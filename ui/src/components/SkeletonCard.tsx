export interface SkeletonCardProps {
  /** How many placeholder cards to render. Defaults to 1. */
  count?: number
}

/**
 * Generic shimmer-animated card placeholder. Used for loading states on
 * list pages and grid views. Each card reuses the .skeleton-card class
 * from index.css (100px tall, --radius-m, shimmer animation).
 */
export function SkeletonCard({ count = 1 }: SkeletonCardProps) {
  const n = Math.max(1, count)
  return (
    <div role="status" aria-busy="true" aria-label="Loading content">
      {Array.from({ length: n }, (_, i) => (
        <div key={i} className="skeleton skeleton-card" />
      ))}
    </div>
  )
}
