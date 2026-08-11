import type {
  ReactNode,
  HTMLAttributes,
  ButtonHTMLAttributes,
  MouseEventHandler,
} from 'react'

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode
  /**
   * Add a hover affordance (cursor + border + shadow). Decorative only by
   * default — the Card stays a non-interactive `<div>`.
   *
   * When an `onClick` handler is passed the Card renders a real `<button>`
   * (same className) so it is keyboard-accessible (Enter/Space) and exposed
   * to assistive tech without extra props. Pass `interactive` to also apply
   * the hover affordance.
   */
  interactive?: boolean
}

export function Card({
  children,
  className = '',
  interactive = false,
  onClick,
  ...rest
}: CardProps) {
  const cls = `card ${interactive ? 'card-interactive ' : ''}${className}`.trim()
  if (onClick) {
    // HTMLAttributes<HTMLDivElement> handlers are strictly typed against the
    // div element; convert them for the <button> boundary (React 19 types).
    const buttonProps = rest as unknown as ButtonHTMLAttributes<HTMLButtonElement>
    return (
      <button
        type="button"
        className={`${cls} btn-reset`}
        onClick={onClick as unknown as MouseEventHandler<HTMLButtonElement>}
        {...buttonProps}
      >
        {children}
      </button>
    )
  }
  return (
    <div className={cls} {...rest}>
      {children}
    </div>
  )
}

interface CardHeaderProps {
  title: string
  children?: ReactNode
  /** Heading level for the title (WCAG 1.3.1). Defaults to 2. */
  headingLevel?: 1 | 2 | 3 | 4
}

export function CardHeader({ title, children, headingLevel = 2 }: CardHeaderProps) {
  const Heading = `h${headingLevel}` as 'h1' | 'h2' | 'h3' | 'h4'
  return (
    <div className="card-header">
      <Heading className="card-title">{title}</Heading>
      {children}
    </div>
  )
}
