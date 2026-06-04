import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SkeletonCard } from './SkeletonCard'

describe('SkeletonCard', () => {
  it('renders a single card placeholder by default', () => {
    const { container } = render(<SkeletonCard />)
    expect(container.querySelectorAll('.skeleton-card').length).toBe(1)
  })

  it('renders N cards when count is provided', () => {
    const { container } = render(<SkeletonCard count={3} />)
    expect(container.querySelectorAll('.skeleton-card').length).toBe(3)
  })

  it('announces loading state to assistive tech', () => {
    render(<SkeletonCard count={2} />)
    const region = screen.getByRole('status')
    expect(region).toHaveAttribute('aria-busy', 'true')
    expect(region).toHaveAttribute('aria-label', 'Loading content')
  })

  it('uses shimmer animation for each card', () => {
    const { container } = render(<SkeletonCard count={2} />)
    const cards = container.querySelectorAll('.skeleton-card')
    cards.forEach(c => {
      expect(c.classList.contains('skeleton')).toBe(true)
    })
  })
})
