import { describe, it, expect } from 'vitest'
import { createElement, type ReactElement } from 'react'
import { render, screen } from '@testing-library/react'
import { Input } from './Input'

describe('Input', () => {
  it('renders a prefix slot inside the input group', () => {
    const prefixIcon: ReactElement = createElement(
      'span',
      { 'data-testid': 'prefix-icon' },
      '@'
    )
    const { container } = render(
      <Input as="input" label="Search" prefix={prefixIcon} name="search" />
    )
    expect(screen.getByTestId('prefix-icon')).toBeInTheDocument()
    expect(container.querySelector('.input-group')).toBeInTheDocument()
    expect(container.querySelector('.input-with-prefix')).toBeInTheDocument()
  })

  it('applies input-mono class when mono is true', () => {
    render(<Input as="input" label="SQL" mono name="sql" />)
    expect(screen.getByLabelText('SQL')).toHaveClass('input-mono')
  })

  it('does not apply input-mono class by default', () => {
    render(<Input as="input" label="Name" name="name" />)
    expect(screen.getByLabelText('Name')).not.toHaveClass('input-mono')
  })

  it('renders prefix with the input-mono class as well', () => {
    const prefixEl: ReactElement = createElement(
      'span',
      { 'data-testid': 'prefix' },
      '$'
    )
    render(
      <Input as="input" label="Token" mono prefix={prefixEl} name="token" />
    )
    expect(screen.getByLabelText('Token')).toHaveClass('input-mono')
    expect(screen.getByTestId('prefix')).toBeInTheDocument()
  })
})
