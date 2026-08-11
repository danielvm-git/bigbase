import type { ReactNode } from 'react'
import { Tooltip } from './Tooltip'

interface AbbrProps {
  /** Full expansion of the abbreviation; rendered as the `title` attribute (WCAG 3.1.4). */
  title: string
  /**
   * Optional longer definition for jargon terms (WCAG 3.1.3 / 3.1.5 mechanism).
   * When provided, the abbreviation is wrapped in a `Tooltip` (aria-describedby)
   * so the definition is available on hover and on keyboard focus.
   */
  definition?: string
  className?: string
  children: ReactNode
}

/**
 * Accessible abbreviation. Renders `<abbr title="expansion">` so every
 * abbreviation in the UI carries its expansion (WCAG 2.2 AA/AAA 3.1.4).
 *
 * For high-jargon terms, pass `definition` to wire the term through `Tooltip`:
 * the definition then becomes an on-demand mechanism reachable by mouse hover
 * and keyboard focus (3.1.3 unusual words, 3.1.5 reading level alternative).
 */
export function Abbr({ title, definition, className, children }: AbbrProps) {
  const abbr = (
    <abbr title={title} className={className} tabIndex={definition ? 0 : undefined}>
      {children}
    </abbr>
  )
  if (!definition) return abbr
  return <Tooltip content={definition}>{abbr}</Tooltip>
}
