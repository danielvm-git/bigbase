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

  it('links the label to the input via htmlFor/id', () => {
    const { container } = render(<Input as="input" label="Email" name="email" />)
    const label = container.querySelector('.input-label')
    expect(label).toHaveAttribute('for', 'email')
    expect(screen.getByLabelText('Email')).toHaveAttribute('id', 'email')
  })

  it('falls back to id when name is absent for the label linkage', () => {
    const { container } = render(<Input as="input" label="Site name" id="site-name" />)
    expect(container.querySelector('.input-label')).toHaveAttribute('for', 'site-name')
    expect(screen.getByLabelText('Site name')).toHaveAttribute('id', 'site-name')
  })

  it('passes aria-label through to the input variant', () => {
    render(<Input as="input" name="log-search" aria-label="Search logs" />)
    expect(screen.getByLabelText('Search logs')).toHaveAttribute('id', 'log-search')
  })

  it('passes aria-label through to the select and textarea variants', () => {
    render(
      <Input as="select" name="runtime" aria-label="Runtime">
        <option value="javascript">JavaScript</option>
      </Input>
    )
    expect(screen.getByLabelText('Runtime')).toHaveAttribute('id', 'runtime')
    render(<Input as="textarea" name="query" aria-label="SQL query" />)
    expect(screen.getByLabelText('SQL query')).toHaveAttribute('id', 'query')
  })

  it('associates the error message with the input via aria-describedby and aria-invalid', () => {
    render(<Input as="input" label="Email" name="email" error="Email is required" />)
    const input = screen.getByLabelText('Email')
    const error = screen.getByText('Email is required')
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).toHaveAttribute('aria-describedby', 'email-error')
    expect(error).toHaveAttribute('id', 'email-error')
  })

  it('does not set aria-invalid or aria-describedby when there is no error or hint', () => {
    render(<Input as="input" label="Email" name="email" />)
    const input = screen.getByLabelText('Email')
    expect(input).not.toHaveAttribute('aria-invalid')
    expect(input).not.toHaveAttribute('aria-describedby')
  })

  it('associates the error message with the textarea and select variants', () => {
    render(<Input as="textarea" label="Query" name="query" error="Invalid SQL" />)
    const textarea = screen.getByLabelText('Query')
    expect(textarea).toHaveAttribute('aria-invalid', 'true')
    expect(textarea).toHaveAttribute('aria-describedby', 'query-error')
    expect(screen.getByText('Invalid SQL')).toHaveAttribute('id', 'query-error')

    render(
      <Input as="select" label="Runtime" name="runtime" error="Pick one">
        <option value="node">Node</option>
      </Input>
    )
    const select = screen.getByLabelText('Runtime')
    expect(select).toHaveAttribute('aria-invalid', 'true')
    expect(select).toHaveAttribute('aria-describedby', 'runtime-error')
    expect(screen.getByText('Pick one')).toHaveAttribute('id', 'runtime-error')
  })

  it('associates the hint via aria-describedby when there is no error', () => {
    render(<Input as="input" label="Name" name="name" hint="Visible to your team" />)
    const input = screen.getByLabelText('Name')
    expect(input).toHaveAttribute('aria-describedby', 'name-hint')
    expect(screen.getByText('Visible to your team')).toHaveAttribute('id', 'name-hint')
  })
})
