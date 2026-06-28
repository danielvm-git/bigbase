import type { HTMLAttributes, ThHTMLAttributes, TdHTMLAttributes, ReactNode } from 'react'

interface TableProps extends HTMLAttributes<HTMLTableElement> {
  children: ReactNode
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

export function Table({ children, className = '', ...rest }: TableProps) {
  return (
    <div className="table-wrapper">
      <table className={`table ${className}`.trim()} {...rest}>
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
