import { Link } from 'react-router-dom'
import type { Site } from '../types/sites'
import { Badge, statusBadgeVariant } from './Badge'
import { Icon } from './Icon'
import { previewQuerySuffix } from '../lib/previewMode'
import { mapDeployStatus, siteDisplayUrl, siteThumbStyle, timeAgo } from '../lib/format'

interface SiteCardProps {
  site: Site
}

export function SiteCard({ site }: SiteCardProps) {
  const dep = site.latest_deployment
  const rawStatus = dep?.status ?? '—'
  const displayStatus = dep ? mapDeployStatus(String(rawStatus)) : '—'
  const pq = previewQuerySuffix()
  const url = dep?.url || siteDisplayUrl(site.name)
  const commitShort = dep?.commit_sha?.slice(0, 7) ?? '—'

  return (
    <Link to={`/deploy/${site.id}${pq}`} className="site-card" aria-label={`Open ${site.name} deployment`}>
      <div className="site-card-thumb" style={siteThumbStyle(site.id)}>
        {(displayStatus === 'building' || rawStatus === 'building' || rawStatus === 'pending') && (
          <span className="site-card-thumb-badge">
            <Badge variant="warning">Building</Badge>
          </span>
        )}
        {(displayStatus === 'failed' || rawStatus === 'failed') && (
          <span className="site-card-thumb-badge">
            <Badge variant="error">Failed</Badge>
          </span>
        )}
      </div>
      <div className="site-card-body">
        <div className="site-card-row">
          <h3 className="site-card-name">{site.name}</h3>
          {dep && (
            <Badge variant={statusBadgeVariant(rawStatus)}>{displayStatus}</Badge>
          )}
        </div>
        <span className="site-card-url">
          <Icon name="globe" size={12} />
          {url.replace(/^https?:\/\//, '')}
        </span>
        <div className="site-card-meta">
          <Icon name="git-branch" size={13} />
          <span>
            {site.production_branch} · {commitShort}
            {dep?.created_at ? ` · ${timeAgo(dep.created_at)}` : ''}
          </span>
        </div>
      </div>
    </Link>
  )
}
