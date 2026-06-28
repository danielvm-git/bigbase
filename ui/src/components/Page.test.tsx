import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Page } from './Page'

describe('Page', () => {
  it('renders children', () => {
    render(<Page><p>Content</p></Page>)
    expect(screen.getByText('Content')).toBeInTheDocument()
  })

  it('has page class', () => {
    const { container } = render(<Page><p>x</p></Page>)
    expect(container.querySelector('.page')).toBeInTheDocument()
  })

  it('renders title when provided', () => {
    render(<Page title="My Page"><p>x</p></Page>)
    expect(screen.getByRole('heading', { name: 'My Page' })).toBeInTheDocument()
  })

  it('renders subtitle when provided', () => {
    render(<Page title="Title" subtitle="Subtitle text"><p>x</p></Page>)
    expect(screen.getByText('Subtitle text')).toBeInTheDocument()
  })

  it('renders actions slot', () => {
    render(<Page actions={<button>Create</button>}><p>x</p></Page>)
    expect(screen.getByRole('button', { name: 'Create' })).toBeInTheDocument()
  })

  it('applies custom className', () => {
    const { container } = render(<Page className="custom"><p>x</p></Page>)
    expect(container.querySelector('.page')?.className).toContain('custom')
  })
})
