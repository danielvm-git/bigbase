import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ChoiceCard } from './ChoiceCard'

describe('ChoiceCard', () => {
  it('renders the title, description, and icon', () => {
    render(
      <ChoiceCard
        title="GitHub"
        description="Connect a repository"
        icon={<span data-testid="icon">GH</span>}
        onClick={() => {}}
      />
    )
    expect(screen.getByText('GitHub')).toBeInTheDocument()
    expect(screen.getByText('Connect a repository')).toBeInTheDocument()
    expect(screen.getByTestId('icon')).toBeInTheDocument()
  })

  it('calls onClick when clicked', () => {
    const onClick = vi.fn()
    render(
      <ChoiceCard
        title="GitHub"
        description="repo"
        icon={<span>x</span>}
        onClick={onClick}
      />
    )
    fireEvent.click(screen.getByText('GitHub'))
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('applies the selected class when selected is true', () => {
    const { container } = render(
      <ChoiceCard
        title="GitHub"
        description="repo"
        icon={<span>x</span>}
        selected
        onClick={() => {}}
      />
    )
    const card = container.firstElementChild
    expect(card).toHaveClass('choice-card')
    expect(card).toHaveClass('choice-card--selected')
  })

  it('does not apply the selected class when not selected', () => {
    const { container } = render(
      <ChoiceCard
        title="GitHub"
        description="repo"
        icon={<span>x</span>}
        onClick={() => {}}
      />
    )
    const card = container.firstElementChild
    expect(card).toHaveClass('choice-card')
    expect(card).not.toHaveClass('choice-card--selected')
  })

  it('renders an optional badge', () => {
    render(
      <ChoiceCard
        title="GitLab"
        description="repo"
        icon={<span>x</span>}
        badge="Beta"
        onClick={() => {}}
      />
    )
    expect(screen.getByText('Beta')).toBeInTheDocument()
  })
})
