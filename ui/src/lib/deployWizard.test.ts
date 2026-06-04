import { describe, it, expect } from 'vitest'
import { deployWizardTitle } from './deployWizard'

describe('deployWizardTitle', () => {
  it('shows building while deploying', () => {
    expect(deployWizardTitle(true, 'pending')).toBe('Building your site…')
  })

  it('shows building for pending or building status', () => {
    expect(deployWizardTitle(false, 'building')).toBe('Building your site…')
    expect(deployWizardTitle(false, 'pending')).toBe('Building your site…')
  })

  it('shows failure headline when status is failed', () => {
    expect(deployWizardTitle(false, 'failed')).toBe('Deploy failed')
  })

  it('shows live only when running and not deploying', () => {
    expect(deployWizardTitle(false, 'running')).toBe('Your site is live')
  })
})
