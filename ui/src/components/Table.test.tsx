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
})
