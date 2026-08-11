import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Card, CardHeader } from './Card'

describe('Card', () => {
  it('renders a basic card without interactive class by default', () => {
    const { container } = render(<Card>x</Card>)
    const card = container.firstElementChild
    expect(card).toHaveClass('card')
    expect(card).not.toHaveClass('card-interactive')
  })

  it('applies the card-interactive class when interactive is true', () => {
    const { container } = render(<Card interactive>x</Card>)
    const card = container.firstElementChild
    expect(card).toHaveClass('card-interactive')
    expect(card).toHaveClass('card')
  })

  it('forwards extra className alongside card', () => {
    const { container } = render(<Card className="custom-class">x</Card>)
    const card = container.firstElementChild
    expect(card).toHaveClass('card')
    expect(card).toHaveClass('custom-class')
  })

  it('renders a real button when onClick is provided', () => {
    const { container } = render(<Card onClick={() => {}}>x</Card>)
    const card = container.firstElementChild as HTMLElement
    expect(card.tagName).toBe('BUTTON')
    expect(card).toHaveClass('card')
    expect(card).toHaveAttribute('type', 'button')
  })

  it('activates onClick with Enter and Space keys (WCAG 2.1.1)', async () => {
    const onClick = vi.fn()
    const user = userEvent.setup()
    render(<Card onClick={onClick}>x</Card>)
    const card = screen.getByRole('button')
    card.focus()
    await user.keyboard('{Enter}')
    expect(onClick).toHaveBeenCalledTimes(1)
    await user.keyboard(' ')
    expect(onClick).toHaveBeenCalledTimes(2)
  })

  it('calls onClick on click when interactive usage', () => {
    const onClick = vi.fn()
    render(<Card interactive onClick={onClick}>x</Card>)
    screen.getByRole('button').click()
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('renders CardHeader with title and optional children', () => {
    render(
      <Card>
        <CardHeader title="My title">extra</CardHeader>
      </Card>
    )
    expect(screen.getByText('My title')).toBeInTheDocument()
    expect(screen.getByText('extra')).toBeInTheDocument()
  })

  it('renders CardHeader title as a level-2 heading by default (WCAG 1.3.1)', () => {
    render(
      <Card>
        <CardHeader title="My title" />
      </Card>
    )
    expect(screen.getByRole('heading', { name: 'My title', level: 2 })).toBeInTheDocument()
  })

  it('respects the headingLevel prop on CardHeader', () => {
    render(
      <Card>
        <CardHeader title="My title" headingLevel={3} />
      </Card>
    )
    expect(screen.getByRole('heading', { name: 'My title', level: 3 })).toBeInTheDocument()
  })
})
