#!/usr/bin/env node
/**
 * Fails CI/preflight if production CSS bundle omits Sites layout rules.
 * Prevents recurrence of BUG-2026-06-04T112700 (unstyled choice-grid / repo-picker).
 */
import { readFileSync, readdirSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const distAssets = resolve(root, 'dist/assets')

let cssFiles
try {
  cssFiles = readdirSync(distAssets).filter(f => f.startsWith('index-') && f.endsWith('.css'))
} catch {
  console.error('verify-sites-css-in-dist: ui/dist/assets not found — run npm run build first')
  process.exit(1)
}

if (cssFiles.length === 0) {
  console.error('verify-sites-css-in-dist: no index-*.css in ui/dist/assets')
  process.exit(1)
}

const css = cssFiles
  .map(f => readFileSync(resolve(distAssets, f), 'utf8'))
  .join('\n')

const required = [
  'choice-grid',
  'choice-card--selected',
  'repo-picker',
  'repo-picker-item',
  'wizard-steps',
  'wizard-step--active',
  'site-grid',
]

const missing = required.filter(token => !css.includes(token))
if (missing.length > 0) {
  console.error('verify-sites-css-in-dist: built CSS missing:', missing.join(', '))
  console.error('  files:', cssFiles.join(', '))
  process.exit(1)
}

console.log('verify-sites-css-in-dist: OK (%s)', cssFiles.join(', '))
