import { describe, it, expect } from 'vitest'
import { formatFunctionEnv, parseFunctionEnv, functionPayloadFromForm } from './functionEnv'

describe('functionEnv', () => {
  it('formats env object for textarea display', () => {
    expect(formatFunctionEnv({ KEY: 'val' })).toBe('{\n  "KEY": "val"\n}')
  })

  it('parses valid env JSON object', () => {
    const result = parseFunctionEnv('{"A":"1"}')
    expect(result).toEqual({ ok: true, env: { A: '1' } })
  })

  it('rejects non-object env JSON', () => {
    expect(parseFunctionEnv('[]').ok).toBe(false)
  })

  it('builds API payload with object env from form string', () => {
    const result = functionPayloadFromForm({
      name: 'fn',
      runtime: 'javascript',
      source: 'code',
      trigger: 'http',
      schedule: '',
      env: '{"FOO":"bar"}',
      timeout: 30,
    })
    expect(result.ok).toBe(true)
    if (result.ok) {
      expect(result.payload.env).toEqual({ FOO: 'bar' })
    }
  })
})
