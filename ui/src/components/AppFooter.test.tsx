import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { AppFooter } from './AppFooter'

describe('AppFooter', () => {
  it('renders copyright', () => {
    render(<AppFooter />)
    expect(screen.getByText(/© 2026 BigBase/)).toBeInTheDocument()
  })

  it('renders GitHub link', () => {
    render(<AppFooter />)
    expect(screen.getByRole('link', { name: 'GitHub' })).toBeInTheDocument()
  })

  it('renders Glossary link to the definition mechanism', () => {
    render(<AppFooter />)
    const link = screen.getByRole('link', { name: 'Glossary' })
    expect(link).toHaveAttribute('href', expect.stringContaining('GLOSSARY_LATEST.md'))
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('renders version when provided', () => {
    render(<AppFooter appVersion="2.5.0" />)
    expect(screen.getByText('v2.5.0')).toBeInTheDocument()
  })

  it('does not render version when not provided', () => {
    render(<AppFooter />)
    expect(screen.queryByText(/^v\d/)).not.toBeInTheDocument()
  })

  it('renders Help button when showTutorial is true', () => {
    render(<AppFooter showTutorial onOpenTutorial={vi.fn()} />)
    expect(screen.getByRole('button', { name: 'Help' })).toBeInTheDocument()
  })

  it('does not render Help button when showTutorial is false', () => {
    render(<AppFooter showTutorial={false} />)
    expect(screen.queryByRole('button', { name: 'Help' })).not.toBeInTheDocument()
  })

  it('calls onOpenTutorial when Help is clicked', () => {
    const onOpen = vi.fn()
    render(<AppFooter showTutorial onOpenTutorial={onOpen} />)
    fireEvent.click(screen.getByRole('button', { name: 'Help' }))
    expect(onOpen).toHaveBeenCalledOnce()
  })

  it('has data-testid app-footer', () => {
    render(<AppFooter />)
    expect(screen.getByTestId('app-footer')).toBeInTheDocument()
  })
})
