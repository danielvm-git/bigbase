import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Sparkline } from './Sparkline'

describe('Sparkline', () => {
  it('uses the provided label', () => {
    render(<Sparkline values={[1, 3, 2]} label="CPU trend" />)
    expect(screen.getByRole('img', { name: 'CPU trend' })).toBeInTheDocument()
  })

  it('default label summarizes the data (min/max/last)', () => {
    render(<Sparkline values={[1, 5, 3]} />)
    expect(
      screen.getByRole('img', { name: 'sparkline: 3 data points, min 1, max 5, last 3' }),
    ).toBeInTheDocument()
  })

  it('handles empty values', () => {
    render(<Sparkline values={[]} />)
    expect(screen.getByRole('img', { name: 'sparkline: no data' })).toBeInTheDocument()
  })

  it('handles a single value', () => {
    render(<Sparkline values={[7]} />)
    expect(
      screen.getByRole('img', { name: 'sparkline: 1 data points, min 7, max 7, last 7' }),
    ).toBeInTheDocument()
  })
})
