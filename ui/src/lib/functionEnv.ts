export type FunctionEnv = Record<string, string>

export function formatFunctionEnv(env: FunctionEnv | undefined): string {
  return JSON.stringify(env ?? {}, null, 2)
}

export function parseFunctionEnv(
  text: string,
): { ok: true; env: FunctionEnv } | { ok: false; error: string } {
  try {
    const parsed: unknown = JSON.parse(text || '{}')
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return { ok: false, error: 'Env must be a JSON object' }
    }
    const env: FunctionEnv = {}
    for (const [key, value] of Object.entries(parsed)) {
      if (typeof value !== 'string') {
        return { ok: false, error: `Env value for "${key}" must be a string` }
      }
      env[key] = value
    }
    return { ok: true, env }
  } catch {
    return { ok: false, error: 'Env must be valid JSON' }
  }
}

export interface FunctionRecord {
  id: string
  name: string
  runtime: string
  source: string
  trigger: string
  schedule: string
  env: FunctionEnv
  timeout: number
  created_at: string
}

export interface FunctionFormFields {
  name: string
  runtime: string
  source: string
  trigger: string
  schedule: string
  env: string
  timeout: number
}

export function functionPayloadFromForm(
  form: FunctionFormFields,
): { ok: true; payload: Omit<FunctionRecord, 'id' | 'created_at'> } | { ok: false; error: string } {
  const envResult = parseFunctionEnv(form.env)
  if (!envResult.ok) return envResult
  return {
    ok: true,
    payload: {
      name: form.name,
      runtime: form.runtime,
      source: form.source,
      trigger: form.trigger,
      schedule: form.schedule,
      env: envResult.env,
      timeout: form.timeout,
    },
  }
}

export function formEnvFromRecord(env: FunctionEnv | string | undefined): string {
  if (typeof env === 'string') return env
  return formatFunctionEnv(env)
}
