/**
 * Minimal ANSI escape code to HTML converter.
 * Handles 16 foreground colors, bold, and dim.
 * Unrecognized sequences are stripped silently.
 * HTML entities in plain text are escaped.
 */

type StyleState = {
  bold: boolean
  dim: boolean
  fg: string | null
}

const FG_COLORS: Record<string, string> = {
  '30': '#212529', // black
  '31': '#ff6b6b', // red
  '32': '#69db7c', // green
  '33': '#ffd43b', // yellow
  '34': '#74c0fc', // blue
  '35': '#da77f2', // magenta
  '36': '#66d9e8', // cyan
  '37': '#e6edf3', // white (bright)
  '90': '#868e96', // bright black (gray)
  '91': '#ff8787', // bright red
  '92': '#8ce99a', // bright green
  '93': '#ffe066', // bright yellow
  '94': '#91a7ff', // bright blue
  '95': '#e599f7', // bright magenta
  '96': '#99e9f2', // bright cyan
  '97': '#f8f9fa', // bright white
}

const ANSI_RE = /\x1b\[([\d;]*)m/g

function styleToString(s: StyleState): string {
  const parts: string[] = []
  if (s.bold) parts.push('font-weight:bold')
  if (s.dim) parts.push('opacity:0.6')
  if (s.fg) parts.push(`color:${s.fg}`)
  return parts.join(';')
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export function ansiToHtml(input: string): string {
  if (!input) return ''

  const parts: string[] = []
  let lastIndex = 0
  let openSpan = false
  let currentStyle: StyleState = { bold: false, dim: false, fg: null }

  const closeSpan = () => {
    if (openSpan) {
      parts.push('</span>')
      openSpan = false
    }
  }

  const openSpanTag = (style: StyleState) => {
    const s = styleToString(style)
    if (s) {
      parts.push(`<span style="${s}">`)
      openSpan = true
    }
  }

  let match: RegExpExecArray | null
  ANSI_RE.lastIndex = 0

  while ((match = ANSI_RE.exec(input)) !== null) {
    // Append text before this escape sequence
    if (match.index > lastIndex) {
      const text = input.slice(lastIndex, match.index)
      parts.push(escapeHtml(text))
    }

    const codes = match[1] ? match[1].split(';').map(Number) : [0]

    if (codes.includes(0) || codes.length === 1 && codes[0] === 0) {
      // Reset
      closeSpan()
      currentStyle = { bold: false, dim: false, fg: null }
    } else {
      const newStyle: StyleState = { ...currentStyle }

      for (const code of codes) {
        if (code === 1) {
          newStyle.bold = true
          newStyle.dim = false
        } else if (code === 2) {
          newStyle.dim = true
          newStyle.bold = false
        } else if (code === 22) {
          newStyle.bold = false
          newStyle.dim = false
        } else if (code >= 30 && code <= 37 || code >= 90 && code <= 97) {
          newStyle.fg = FG_COLORS[String(code)] || null
        } else if (code === 39) {
          newStyle.fg = null
        }
      }

      const newStyleStr = styleToString(newStyle)
      const oldStyleStr = styleToString(currentStyle)

      if (newStyleStr !== oldStyleStr) {
        closeSpan()
        currentStyle = newStyle
        openSpanTag(currentStyle)
      }
    }

    lastIndex = ANSI_RE.lastIndex
  }

  // Append remaining text after last escape sequence
  if (lastIndex < input.length) {
    parts.push(escapeHtml(input.slice(lastIndex)))
  }

  closeSpan()

  return parts.join('')
}
