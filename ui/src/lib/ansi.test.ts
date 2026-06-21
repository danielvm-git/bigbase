import { describe, it, expect } from 'vitest'
import { ansiToHtml } from './ansi'

describe('ansiToHtml', () => {
  it('returns plain text unchanged', () => {
    expect(ansiToHtml('Hello world')).toBe('Hello world')
  })

  it('strips unsupported escape sequences', () => {
    expect(ansiToHtml('\x1b[999mUnknown\x1b[0m text')).toBe('Unknown text')
  })

  it('converts foreground colors', () => {
    // Red: \x1b[31m
    expect(ansiToHtml('\x1b[31mError\x1b[0m')).toBe('<span style="color:#ff6b6b">Error</span>')
    // Green: \x1b[32m
    expect(ansiToHtml('\x1b[32mSuccess\x1b[0m')).toBe('<span style="color:#69db7c">Success</span>')
    // Blue: \x1b[34m
    expect(ansiToHtml('\x1b[34mInfo\x1b[0m')).toBe('<span style="color:#74c0fc">Info</span>')
    // Yellow: \x1b[33m
    expect(ansiToHtml('\x1b[33mWarning\x1b[0m')).toBe('<span style="color:#ffd43b">Warning</span>')
    // Cyan: \x1b[36m
    expect(ansiToHtml('\x1b[36mDebug\x1b[0m')).toBe('<span style="color:#66d9e8">Debug</span>')
    // Magenta: \x1b[35m
    expect(ansiToHtml('\x1b[35mTrace\x1b[0m')).toBe('<span style="color:#da77f2">Trace</span>')
  })

  it('converts bold text', () => {
    expect(ansiToHtml('\x1b[1mBold\x1b[0m')).toBe('<span style="font-weight:bold">Bold</span>')
  })

  it('converts dim text', () => {
    expect(ansiToHtml('\x1b[2mDim\x1b[0m')).toBe('<span style="opacity:0.6">Dim</span>')
  })

  it('strips reset-only sequences', () => {
    expect(ansiToHtml('\x1b[0m')).toBe('')
    expect(ansiToHtml('text\x1b[0mmore')).toBe('textmore')
  })

  it('handles multiple codes in one sequence', () => {
    // Bold + Red: \x1b[1;31m
    expect(ansiToHtml('\x1b[1;31mBold Red\x1b[0m'))
      .toBe('<span style="font-weight:bold;color:#ff6b6b">Bold Red</span>')
  })

  it('handles nested/multiple spans', () => {
    expect(ansiToHtml('\x1b[31mRed \x1b[32mGreen\x1b[0m after'))
      .toBe('<span style="color:#ff6b6b">Red </span><span style="color:#69db7c">Green</span> after')
  })

  it('handles npm-style output (real example)', () => {
    const input = '\x1b[1madded\x1b[0m 42 packages in 3s'
    expect(ansiToHtml(input)).toBe('<span style="font-weight:bold">added</span> 42 packages in 3s')
  })

  it('handles empty string', () => {
    expect(ansiToHtml('')).toBe('')
  })

  it('escapes HTML in non-ANSI text', () => {
    expect(ansiToHtml('<script>alert(1)</script>')).toBe('&lt;script&gt;alert(1)&lt;/script&gt;')
  })
})
