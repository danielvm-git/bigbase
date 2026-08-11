// e87s01 AAA contrast matrix — WCAG 2.2 1.4.6 (threshold 7.0), light + dark.
// Usage: node e87s01-contrast-matrix.mjs [--exceptions]
const showExceptions = process.argv.includes('--exceptions')

const lin = c => { c /= 255; return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4) }
const L = ([r, g, b]) => 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)
const C = (a, b) => { const x = L(a), y = L(b); return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05) }
const comp = (fg, base, a) => fg.map((v, i) => a * v + (1 - a) * base[i])

const WHITE = [255, 255, 255]
const N25 = [250, 250, 251]
const N40 = [244, 244, 247]
const N900 = [25, 25, 28]
const N850 = [29, 29, 33]
const N800 = [45, 45, 49]

const EMERALD = [16, 185, 129]
const AMBER = [245, 158, 11]
const RED = [239, 68, 68]
const BLUE = [59, 130, 246]
const INDIGO = [79, 70, 229]

const light = {
  bgs: {
    'bg-surface (white)': WHITE,
    'bg-default (neutral-25)': N25,
    'bg-surface-secondary (neutral-40)': N40,
    'brand-tint over white': comp(INDIGO, WHITE, 0.06),
    'success-bg over white': comp(EMERALD, WHITE, 0.12),
    'success-bg over neutral-25': comp(EMERALD, N25, 0.12),
    'warning-bg over white': comp(AMBER, WHITE, 0.12),
    'error-bg over white': comp(RED, WHITE, 0.08),
    'info-bg over white': comp(BLUE, WHITE, 0.10),
    'bg-accent (brand-600)': [67, 56, 202],
    'bg-accent-hover (brand-700)': [55, 48, 163],
  },
  pairs: [
    ['fg-primary (neutral-800)', [45, 45, 49], ['bg-surface (white)', 'bg-default (neutral-25)', 'bg-surface-secondary (neutral-40)']],
    ['fg-secondary (75,75,80)', [75, 75, 80], ['bg-surface (white)', 'bg-default (neutral-25)', 'bg-surface-secondary (neutral-40)']],
    ['fg-tertiary (75,75,80)', [75, 75, 80], ['bg-surface (white)', 'bg-default (neutral-25)', 'bg-surface-secondary (neutral-40)']],
    ['fg-accent (brand-600)', [67, 56, 202], ['bg-surface (white)', 'bg-default (neutral-25)', 'bg-surface-secondary (neutral-40)', 'brand-tint over white']],
    ['success-fg (4,78,58)', [4, 78, 58], ['success-bg over white', 'success-bg over neutral-25']],
    ['warning-fg (120,53,15)', [120, 53, 15], ['warning-bg over white']],
    ['error-fg (120,20,20)', [120, 20, 20], ['error-bg over white']],
    ['info-fg (30,64,175)', [30, 64, 175], ['info-bg over white']],
    ['fg-on-accent (white)', [255, 255, 255], ['bg-accent (brand-600)', 'bg-accent-hover (brand-700)']],
  ],
}

const dark = {
  bgs: {
    'bg-default (neutral-900)': N900,
    'bg-surface (neutral-850)': N850,
    'bg-surface-secondary (neutral-800)': N800,
    'brand-tint over neutral-850': comp(INDIGO, N850, 0.06),
    'success-bg over neutral-850': comp(EMERALD, N850, 0.12),
    'warning-bg over neutral-850': comp(AMBER, N850, 0.12),
    'error-bg over neutral-850': comp(RED, N850, 0.08),
    'info-bg over neutral-850': comp(BLUE, N850, 0.10),
    'bg-accent (brand-600)': [67, 56, 202],
  },
  pairs: [
    ['fg-primary (neutral-25)', [250, 250, 251], ['bg-default (neutral-900)', 'bg-surface (neutral-850)', 'bg-surface-secondary (neutral-800)']],
    ['fg-secondary (neutral-250)', [195, 195, 198], ['bg-default (neutral-900)', 'bg-surface (neutral-850)', 'bg-surface-secondary (neutral-800)']],
    ['fg-tertiary (neutral-300)', [173, 173, 176], ['bg-default (neutral-900)', 'bg-surface (neutral-850)']],
    ['fg-accent (brand-300)', [165, 180, 252], ['bg-default (neutral-900)', 'bg-surface (neutral-850)', 'brand-tint over neutral-850']],
    ['success-fg (110,231,183)', [110, 231, 183], ['success-bg over neutral-850']],
    ['warning-fg (252,211,77)', [252, 211, 77], ['warning-bg over neutral-850']],
    ['error-fg (252,165,165)', [252, 165, 165], ['error-bg over neutral-850']],
    ['info-fg (147,197,253)', [147, 197, 253], ['info-bg over neutral-850']],
    ['fg-on-accent (white)', [255, 255, 255], ['bg-accent (brand-600)']],
  ],
}

let failures = 0
const rows = []
for (const [modeName, mode] of [['LIGHT', light], ['DARK', dark]]) {
  for (const [fgName, fg, bgNames] of mode.pairs) {
    for (const bgName of bgNames) {
      const ratio = C(fg, mode.bgs[bgName])
      const ok = ratio >= 7
      if (!ok) failures++
      rows.push(`${ok ? 'PASS' : 'FAIL'} ${modeName}  ${fgName.padEnd(26)} on ${bgName.padEnd(30)} ${ratio.toFixed(2)}:1`)
    }
  }
}
for (const r of rows) console.log(r)
console.log(`\n${rows.length} pairs checked, ${failures} failures (threshold 7.0)`)

if (showExceptions) {
  console.log('\n--- Known component/state pairs (documented, not token pairs) ---')
  const pairs = [
    ['light', '.btn-danger white on --error', [255, 255, 255], RED],
    ['light', '.input-error-text --error on white', RED, WHITE],
    ['dark', '.input-error-text --error on neutral-850', RED, N850],
    ['light', 'wizard done white on --success', [255, 255, 255], EMERALD],
    ['dark', 'wizard-step-num fg-tertiary on bg-subtle', [173, 173, 176], N800],
    ['light', '.sql-textarea::placeholder neutral-300 on neutral-900', [173, 173, 176], N900],
    ['dark', '.sql-textarea::placeholder neutral-300 on neutral-900', [173, 173, 176], N900],
    ['light', 'wizard-step-num fg-tertiary on bg-subtle(neutral-40)', [75, 75, 80], N40],
  ]
  for (const [mode, name, fg, bg] of pairs) console.log(`  ${mode.padEnd(5)} ${name.padEnd(48)} ${C(fg, bg).toFixed(2)}:1`)
}
process.exit(failures > 0 ? 1 : 0)
