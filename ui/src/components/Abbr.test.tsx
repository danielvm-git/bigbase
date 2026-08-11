import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Abbr } from './Abbr'

describe('Abbr', () => {
  it('renders the abbreviated text', () => {
    render(<Abbr title="Structured Query Language">SQL</Abbr>)
    expect(screen.getByText('SQL')).toBeInTheDocument()
  })

  it('sets the title attribute to the expansion', () => {
    render(<Abbr title="Structured Query Language">SQL</Abbr>)
    expect(screen.getByText('SQL')).toHaveAttribute('title', 'Structured Query Language')
  })

  it('renders a plain abbr when no definition is given', () => {
    render(<Abbr title="Megabyte">MB</Abbr>)
    const el = screen.getByText('MB')
    expect(el.tagName).toBe('ABBR')
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('wires the definition through a tooltip when provided', () => {
    render(
      <Abbr
        title="Concurrent lightweight tasks managed by the Go runtime"
        definition="A goroutine is a lightweight concurrent execution unit managed by the Go runtime."
      >
        Goroutines
      </Abbr>,
    )
    const term = screen.getByText('Goroutines')
    expect(term).toHaveAttribute('title', 'Concurrent lightweight tasks managed by the Go runtime')
    expect(term).toHaveAttribute('aria-describedby')
  })

  it('shows the definition tooltip on focus', () => {
    render(
      <Abbr
        title="Concurrent lightweight tasks managed by the Go runtime"
        definition="A goroutine is a lightweight concurrent execution unit managed by the Go runtime."
      >
        Goroutines
      </Abbr>,
    )
    fireEvent.focus(screen.getByText('Goroutines'))
    expect(screen.getByRole('tooltip')).toBeVisible()
    expect(screen.getByText(/lightweight concurrent execution unit/)).toBeVisible()
  })

  it('hides the definition tooltip on blur', () => {
    render(
      <Abbr
        title="Concurrent lightweight tasks managed by the Go runtime"
        definition="A goroutine is a lightweight concurrent execution unit managed by the Go runtime."
      >
        Goroutines
      </Abbr>,
    )
    const term = screen.getByText('Goroutines')
    fireEvent.focus(term)
    fireEvent.blur(term)
    expect(screen.queryByText(/lightweight concurrent execution unit/)).not.toBeVisible()
  })

  it('is keyboard-focusable when a definition is provided', () => {
    render(
      <Abbr title="Git repository" definition="A repository is a version-controlled code store.">
        Repo
      </Abbr>,
    )
    expect(screen.getByText('Repo')).toHaveAttribute('tabindex', '0')
  })
})
