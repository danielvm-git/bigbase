import { Link } from 'react-router-dom'
import type { Site } from '../types/sites'
import { Badge, statusBadgeVariant } from './Badge'
import { previewQuerySuffix } from '../lib/previewMode'

interface SiteCardProps {
  site: Site
}

export function SiteCard({ site }: SiteCardProps) {
  const dep = site.latest_deployment
  const status = dep?.status ?? '—'
  const pq = previewQuerySuffix()

  return (
    <Link to={`/deploy/${site.id}${pq}`} className="site-card">
      <div className="site-card-main">
        <span className="site-card-name">{site.name}</span>
        <span className="site-card-repo dim">{site.full_name || site.name}</span>
      </div>
      <div className="site-card-meta">
        <span className="dim">{site.production_branch}</span>
        {dep && (
          <Badge variant={statusBadgeVariant(status)}>{status}</Badge>
        )}
        {dep?.created_at && (
          <span className="site-card-time dim">
            {new Date(dep.created_at).toLocaleDateString()}
          </span>
        )}
      </div>
    </Link>
  )
}
