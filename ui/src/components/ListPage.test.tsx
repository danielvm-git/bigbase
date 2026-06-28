import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { ListPage } from './ListPage'

describe('ListPage', () => {
  it('renders title', () => {
    render(<ListPage title="Users"><p>List</p></ListPage>)
    expect(screen.getByRole('heading', { name: 'Users' })).toBeInTheDocument()
  })

  it('renders children (content)', () => {
    render(<ListPage title="Users"><p>User list</p></ListPage>)
    expect(screen.getByText('User list')).toBeInTheDocument()
  })

  it('renders loading state', () => {
    render(<ListPage title="Users" loading><p>ignored</p></ListPage>)
    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('renders error state', () => {
    render(<ListPage title="Users" error="Failed to load"><p>ignored</p></ListPage>)
    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText('Failed to load')).toBeInTheDocument()
  })

  it('renders empty state', () => {
    render(<ListPage title="Users" empty emptyMessage="No users yet"><p>ignored</p></ListPage>)
    expect(screen.getByText('No users yet')).toBeInTheDocument()
  })

  it('renders filters slot', () => {
    render(<ListPage title="Users" filters={<input placeholder="Search" />}><p>List</p></ListPage>)
    expect(screen.getByPlaceholderText('Search')).toBeInTheDocument()
  })

  it('renders actions slot', () => {
    render(<ListPage title="Users" actions={<button>New User</button>}><p>List</p></ListPage>)
    expect(screen.getByRole('button', { name: 'New User' })).toBeInTheDocument()
  })

  it('renders pagination slot', () => {
    render(<ListPage title="Users" pagination={<nav aria-label="Pagination">page nav</nav>}><p>List</p></ListPage>)
    expect(screen.getByRole('navigation', { name: 'Pagination' })).toBeInTheDocument()
  })
})
