// e88s01 AAA link contrast matrix — WCAG 2.2 1.4.6 (threshold 7.0).
// Covers default + all 13 accent themes:
//   - light-mode link (per-theme brandLink) on white            >= 7:1
//   - dark-mode link (per-theme brand300) on neutral-850        >= 7:1
// Values are parsed from ui/src/context/accentThemes.ts (single source of truth).
// Usage: node specs/epics/e88-aaa-closing/e88s01-contrast-matrix.mjs
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
// specs/epics/e88-aaa-closing/ -> repo root -> ui/src/context/accentThemes.ts
const repoRoot = join(here, '..', '..', '..')
const src = readFileSync(join(repoRoot, 'ui', 'src', 'context', 'accentThemes.ts'), 'utf8')

const lin = c => { c /= 255; return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4) }
const L = ([r, g, b]) => 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)
const C = (a, b) => { const x = L(a), y = L(b); return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05) }
const parseRgb = s => s.split(',').map(v => Number(v.trim()))

const WHITE = [255, 255, 255]
const N850 = [29, 29, 33]

// Extract per-theme entries from the ACCENT_THEMES array (one object per line).
const themes = []
for (const line of src.split('\n')) {
  const m = line.match(/id:\s*'([^']+)'.*?brand300:\s*'([^']+)'.*?brandLink:\s*'([^']+)'/)
  if (m) themes.push({ id: m[1], brand300: parseRgb(m[2]), brandLink: parseRgb(m[3]) })
}

let failures = 0
const rows = []
for (const t of themes) {
  const light = C(t.brandLink, WHITE)
  const dark = C(t.brand300, N850)
  const lightOk = light >= 7
  const darkOk = dark >= 7
  if (!lightOk || !darkOk) failures++
  rows.push(`${lightOk && darkOk ? 'PASS' : 'FAIL'} ${t.id.padEnd(10)} light link ${light.toFixed(2)}:1 on white | dark link ${dark.toFixed(2)}:1 on neutral-850`)
}

for (const r of rows) console.log(r)
console.log(`\n${themes.length} themes checked, ${failures} failures (threshold 7.0)`)
console.log(failures === 0 ? 'PASS' : `FAIL: ${failures} theme(s) below 7:1`)
process.exit(failures > 0 ? 1 : 0)
