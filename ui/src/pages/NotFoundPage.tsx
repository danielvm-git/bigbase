import { Link } from 'react-router-dom'
import { Button } from '../components'

export default function NotFoundPage() {
  return (
    <div className="not-found">
      <h1>404</h1>
      <p>Page not found</p>
      <Link to="/">
        <Button variant="primary" style={{ marginTop: 'var(--space-8)' }}>Go to Dashboard</Button>
      </Link>
    </div>
  )
}
