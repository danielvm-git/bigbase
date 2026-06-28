import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { DetailPage } from './DetailPage'

describe('DetailPage', () => {
  it('renders title', () => {
    render(<DetailPage title="Site Detail"><p>content</p></DetailPage>)
    expect(screen.getByRole('heading', { name: 'Site Detail' })).toBeInTheDocument()
  })

  it('renders children', () => {
    render(<DetailPage title="Site"><p>Detail content</p></DetailPage>)
    expect(screen.getByText('Detail content')).toBeInTheDocument()
  })

  it('renders back button when onBack provided', () => {
    render(<DetailPage title="Site" onBack={vi.fn()}><p>x</p></DetailPage>)
    expect(screen.getByRole('button', { name: /back/i })).toBeInTheDocument()
  })

  it('calls onBack when back button clicked', () => {
    const onBack = vi.fn()
    render(<DetailPage title="Site" onBack={onBack}><p>x</p></DetailPage>)
    fireEvent.click(screen.getByRole('button', { name: /back/i }))
    expect(onBack).toHaveBeenCalledOnce()
  })

  it('does not render back button when onBack not provided', () => {
    render(<DetailPage title="Site"><p>x</p></DetailPage>)
    expect(screen.queryByRole('button', { name: /back/i })).not.toBeInTheDocument()
  })

  it('renders tabs when provided', () => {
    const tabs = <nav aria-label="Tabs"><a href="#overview">Overview</a></nav>
    render(<DetailPage title="Site" tabs={tabs}><p>x</p></DetailPage>)
    expect(screen.getByRole('navigation', { name: 'Tabs' })).toBeInTheDocument()
  })

  it('renders loading spinner', () => {
    render(<DetailPage title="Site" loading><p>x</p></DetailPage>)
    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('renders error', () => {
    render(<DetailPage title="Site" error="Not found"><p>x</p></DetailPage>)
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })

  it('renders actions slot', () => {
    render(<DetailPage title="Site" actions={<button>Edit</button>}><p>x</p></DetailPage>)
    expect(screen.getByRole('button', { name: 'Edit' })).toBeInTheDocument()
  })
})
