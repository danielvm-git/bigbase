import { readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, it, expect } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const sitesCss = readFileSync(resolve(here, 'sites.css'), 'utf8')
const indexCss = readFileSync(resolve(here, '../index.css'), 'utf8')

/** Selectors required for Sites / Create-site layout — regression guard for missing CSS port. */
const REQUIRED_SITES_SELECTORS = [
  '.choice-grid',
  '.choice-card--selected',
  '.choice-card--disabled',
  '.wizard-steps',
  '.wizard-step--active',
  '.wizard-step--done',
  '.wizard-step-line',
  '.repo-picker',
  '.repo-picker-item--selected',
  '.site-grid',
  '.site-card-thumb',
] as const

describe('sites.css source contract', () => {
  it.each(REQUIRED_SITES_SELECTORS)('defines %s', selector => {
    expect(sitesCss).toContain(selector)
  })

  it('choice-grid uses display grid', () => {
    expect(sitesCss).toMatch(/\.choice-grid\s*\{[^}]*display:\s*grid/s)
  })

  it('repo-picker stacks items vertically', () => {
    expect(sitesCss).toMatch(/\.repo-picker\s*\{[^}]*flex-direction:\s*column/s)
  })

  it('wizard-steps is a horizontal flex list without browser numbering', () => {
    expect(sitesCss).toMatch(/\.wizard-steps\s*\{[^}]*display:\s*flex/s)
    expect(sitesCss).toMatch(/\.wizard-steps\s*\{[^}]*list-style:\s*none/s)
  })
})

describe('index.css wiring', () => {
  it('imports sites.css so Vite bundles Sites component styles', () => {
    expect(indexCss).toContain("./styles/sites.css")
  })
})
