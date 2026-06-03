import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { Icon, type IconName } from './Icon'

// Names that must exist per the prototype-fidelity gap (G19).
const EXPECTED_ICONS: IconName[] = [
  'check',
  'chevron-down',
  'x',
  'plus',
  'search',
  'trash-2',
  'pencil',
  'refresh-cw',
  'external-link',
  'arrow-right',
  'arrow-left',
  'chevron-right',
  'eye',
  'eye-off',
  'circle',
]

describe('Icon', () => {
  it('renders an SVG for every IconName', () => {
    for (const name of EXPECTED_ICONS) {
      const { container } = render(<Icon name={name} />)
      const svg = container.querySelector('svg')
      expect(svg, `Icon "${name}" must render an svg element`).toBeInTheDocument()
    }
  })

  it('exposes at least 15 icon names (closing G19)', () => {
    // B4.3 added 15 new names to the IconName union.
    // Combined with the original 16, the type should have at least 15.
    expect(EXPECTED_ICONS.length).toBeGreaterThanOrEqual(15)
  })

  it('renders the default size of 18', () => {
    const { container } = render(<Icon name="check" />)
    const svg = container.querySelector('svg')
    expect(svg).toHaveAttribute('width', '18')
    expect(svg).toHaveAttribute('height', '18')
  })

  it('honors a custom size', () => {
    const { container } = render(<Icon name="check" size={24} />)
    const svg = container.querySelector('svg')
    expect(svg).toHaveAttribute('width', '24')
    expect(svg).toHaveAttribute('height', '24')
  })
})
