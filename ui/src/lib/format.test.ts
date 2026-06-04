import { describe, it, expect } from 'vitest'
import { timeAgo, siteDisplayUrl, mapDeployStatus, fmtUptime, fmtMemoryMB, memoryBarPercent } from './format'

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

  it('fmtUptime formats days and hours', () => {
    const seconds = 14 * 86400 + 6 * 3600
    expect(fmtUptime(seconds)).toBe('14d 6h')
  })

  it('fmtMemoryMB rounds megabytes', () => {
    expect(fmtMemoryMB(512)).toBe('512 MB')
  })

  it('memoryBarPercent caps at 100', () => {
    expect(memoryBarPercent(2048)).toBe(100)
  })
})
