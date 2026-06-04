import { describe, it, expect } from 'vitest'
import { timeAgo, siteDisplayUrl, mapDeployStatus } from './format'

describe('format helpers', () => {
  it('timeAgo returns relative labels', () => {
    const recent = new Date(Date.now() - 120000).toISOString()
    expect(timeAgo(recent)).toMatch(/ago|just now/)
  })

  it('siteDisplayUrl builds production subdomain', () => {
    expect(siteDisplayUrl('My App')).toBe('https://my-app.bigbase.click')
  })

  it('mapDeployStatus normalizes running to ready', () => {
    expect(mapDeployStatus('running')).toBe('ready')
    expect(mapDeployStatus('building')).toBe('building')
  })
})
