import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
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

  it('renders CardHeader with title and optional children', () => {
    render(
      <Card>
        <CardHeader title="My title">extra</CardHeader>
      </Card>
    )
    expect(screen.getByText('My title')).toBeInTheDocument()
    expect(screen.getByText('extra')).toBeInTheDocument()
  })
})
