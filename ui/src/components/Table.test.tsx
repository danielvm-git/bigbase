import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { Table, TableHead, TableBody, TableRow, TableCell } from './Table'

function BasicTable() {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell as="th">Name</TableCell>
          <TableCell as="th">Status</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        <TableRow>
          <TableCell>Alice</TableCell>
          <TableCell>Active</TableCell>
        </TableRow>
        <TableRow>
          <TableCell>Bob</TableCell>
          <TableCell>Inactive</TableCell>
        </TableRow>
      </TableBody>
    </Table>
  )
}

describe('Table', () => {
  it('renders a table element', () => {
    render(<BasicTable />)
    expect(screen.getByRole('table')).toBeInTheDocument()
  })

  it('renders column headers', () => {
    render(<BasicTable />)
    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Status' })).toBeInTheDocument()
  })

  it('renders data rows', () => {
    render(<BasicTable />)
    expect(screen.getByRole('cell', { name: 'Alice' })).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: 'Bob' })).toBeInTheDocument()
  })

  it('renders correct row count', () => {
    render(<BasicTable />)
    expect(screen.getAllByRole('row')).toHaveLength(3)
  })

  it('applies custom className to Table', () => {
    render(<Table className="my-table"><TableBody><TableRow><TableCell>x</TableCell></TableRow></TableBody></Table>)
    expect(screen.getByRole('table').className).toContain('my-table')
  })

  it('renders a visually-hidden caption as the first child of the table', () => {
    render(<Table caption="Deploy keys"><TableBody><TableRow><TableCell>x</TableCell></TableRow></TableBody></Table>)
    const table = screen.getByRole('table', { name: 'Deploy keys' })
    expect(table.querySelector('caption')).toBeInTheDocument()
    // caption must be the first child of <table>
    expect(table.firstElementChild?.tagName).toBe('CAPTION')
  })

  it('exposes the scrollable wrapper as a labelled region when a caption is given', () => {
    const { container } = render(<Table caption="Build cache"><TableBody><TableRow><TableCell>x</TableCell></TableRow></TableBody></Table>)
    const wrapper = container.querySelector('.table-wrapper')
    expect(wrapper).toHaveAttribute('role', 'region')
    expect(wrapper).toHaveAttribute('aria-label', 'Build cache')
  })

  it('does not add an unnamed region when no caption is given', () => {
    const { container } = render(<Table><TableBody><TableRow><TableCell>x</TableCell></TableRow></TableBody></Table>)
    expect(container.querySelector('.table-wrapper')).not.toHaveAttribute('role')
  })

  it('defaults th scope to col', () => {
    render(<BasicTable />)
    expect(screen.getByRole('columnheader', { name: 'Name' })).toHaveAttribute('scope', 'col')
  })

  it('honors a caller-provided scope on th', () => {
    render(
      <Table>
        <TableHead><TableRow><TableCell as="th" scope="row">Name</TableCell></TableRow></TableHead>
        <TableBody><TableRow><TableCell>Alice</TableCell></TableRow></TableBody>
      </Table>
    )
    expect(screen.getByRole('rowheader', { name: 'Name' })).toBeInTheDocument()
    expect(screen.getByRole('rowheader', { name: 'Name' })).toHaveAttribute('scope', 'row')
  })

  it('does not set scope on td cells', () => {
    render(<BasicTable />)
    expect(screen.getByRole('cell', { name: 'Alice' })).not.toHaveAttribute('scope')
  })
})
