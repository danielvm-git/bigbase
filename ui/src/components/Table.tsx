import type { HTMLAttributes, ThHTMLAttributes, TdHTMLAttributes, ReactNode } from 'react'

interface TableProps extends HTMLAttributes<HTMLTableElement> {
  children: ReactNode
  /** Accessible name for the table, rendered as a visually-hidden <caption>. */
  caption?: string
}

interface TableSectionProps extends HTMLAttributes<HTMLTableSectionElement> {
  children: ReactNode
}

interface TableRowProps extends HTMLAttributes<HTMLTableRowElement> {
  children: ReactNode
}

interface TableCellProps extends TdHTMLAttributes<HTMLTableCellElement> {
  children?: ReactNode
  as?: 'td' | 'th'
  scope?: ThHTMLAttributes<HTMLTableCellElement>['scope']
}

export function Table({ children, className = '', caption, ...rest }: TableProps) {
  // The wrapper scrolls horizontally on small screens; expose it as a labelled
  // region so the scrollable area is announced (WCAG 1.4.10 Reflow).
  const regionProps = caption
    ? { role: 'region' as const, 'aria-label': caption }
    : {}
  return (
    <div className="table-wrapper" {...regionProps}>
      <table className={`table ${className}`.trim()} {...rest}>
        {caption && <caption className="visually-hidden">{caption}</caption>}
        {children}
      </table>
    </div>
  )
}

export function TableHead({ children, className = '', ...rest }: TableSectionProps) {
  return (
    <thead className={`table-head ${className}`.trim()} {...rest}>
      {children}
    </thead>
  )
}

export function TableBody({ children, className = '', ...rest }: TableSectionProps) {
  return (
    <tbody className={`table-body ${className}`.trim()} {...rest}>
      {children}
    </tbody>
  )
}

export function TableRow({ children, className = '', ...rest }: TableRowProps) {
  return (
    <tr className={`table-row ${className}`.trim()} {...rest}>
      {children}
    </tr>
  )
}

export function TableCell({ children, as: Tag = 'td', scope, className = '', ...rest }: TableCellProps) {
  const isHeader = Tag === 'th'
  return (
    <Tag
      className={`table-cell${isHeader ? ' table-cell-header' : ''} ${className}`.trim()}
      scope={isHeader ? (scope ?? 'col') : undefined}
      {...(rest as TdHTMLAttributes<HTMLTableCellElement>)}
    >
      {children}
    </Tag>
  )
}
