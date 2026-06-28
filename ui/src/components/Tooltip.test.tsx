import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Tooltip } from './Tooltip'

describe('Tooltip', () => {
  it('renders children', () => {
    render(<Tooltip content="Hint text"><button>Hover me</button></Tooltip>)
    expect(screen.getByRole('button', { name: 'Hover me' })).toBeInTheDocument()
  })

  it('tooltip content is hidden initially', () => {
    render(<Tooltip content="Hint text"><button>Hover me</button></Tooltip>)
    expect(screen.queryByText('Hint text')).not.toBeVisible()
  })

  it('shows tooltip on mouse enter', () => {
    render(<Tooltip content="Hint text"><button>Hover me</button></Tooltip>)
    fireEvent.mouseEnter(screen.getByRole('button'))
    expect(screen.getByRole('tooltip')).toBeVisible()
    expect(screen.getByText('Hint text')).toBeVisible()
  })

  it('hides tooltip on mouse leave', () => {
    render(<Tooltip content="Hint text"><button>Hover me</button></Tooltip>)
    fireEvent.mouseEnter(screen.getByRole('button'))
    fireEvent.mouseLeave(screen.getByRole('button'))
    expect(screen.queryByText('Hint text')).not.toBeVisible()
  })

  it('shows tooltip on focus', () => {
    render(<Tooltip content="Focus hint"><button>Focus me</button></Tooltip>)
    fireEvent.focus(screen.getByRole('button'))
    expect(screen.getByRole('tooltip')).toBeVisible()
  })

  it('hides tooltip on blur', () => {
    render(<Tooltip content="Focus hint"><button>Focus me</button></Tooltip>)
    fireEvent.focus(screen.getByRole('button'))
    fireEvent.blur(screen.getByRole('button'))
    expect(screen.queryByText('Focus hint')).not.toBeVisible()
  })

  it('has aria-describedby linking child to tooltip', () => {
    render(<Tooltip content="Description"><button>Trigger</button></Tooltip>)
    const btn = screen.getByRole('button')
    const tooltipId = btn.getAttribute('aria-describedby')
    expect(tooltipId).toBeTruthy()
    expect(document.getElementById(tooltipId!)).toBeTruthy()
  })

  it('applies placement class', () => {
    render(<Tooltip content="Bottom tip" placement="bottom"><button>T</button></Tooltip>)
    const wrapper = screen.getByRole('button').closest('.tooltip-wrapper')
    expect(wrapper?.className).toContain('tooltip-bottom')
  })
})
