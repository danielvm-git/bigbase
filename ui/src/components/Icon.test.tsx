import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { Icon, type IconName } from './Icon'

// Names that must exist per the prototype-fidelity gap (G19).
const EXPECTED_ICONS: IconName[] = [
  // Original 16 still present (smoke check for regressions).
  'layout-dashboard',
  'rocket',
  'box',
  'database',
  'terminal',
  'hard-drive',
  'users',
  'mail',
  'git-branch',
  'git-pull-request',
  'activity',
  'settings',
  'hammer',
  'radio',
  'moon',
  'sun',
  // 15 new names from B4.3 (plan section 4.3 "Where used" column).
  'check',
  'chevron-down',
  'x',
  'plus',
  'search',
  'trash-2',
  'pencil',
  'refresh-cw',
  'arrow-right',
  'external-link',
  'download',
  'upload',
  'copy',
  'more-horizontal',
  'github',
]

describe('Icon', () => {
  it('renders an SVG for every IconName', () => {
    for (const name of EXPECTED_ICONS) {
      const { container } = render(<Icon name={name} />)
      const svg = container.querySelector('svg')
      expect(svg, `Icon "${name}" must render an svg element`).toBeInTheDocument()
    }
  })

  it('exposes 31 icon names (16 original + 15 new, closing G19)', () => {
    // B4.3 added 15 new names to the IconName union.
    // 16 original + 15 new = 31 total.
    expect(EXPECTED_ICONS.length).toBe(31)
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
