import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Tabs } from './Tabs'

const tabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'settings', label: 'Settings' },
  { id: 'logs', label: 'Logs' },
]

function renderTabs(active = 'overview', onChange = vi.fn()) {
  const utils = render(
    <div>
      <Tabs tabs={tabs} active={active} onChange={onChange} />
      <div id="panel-overview" role="tabpanel" aria-labelledby="tab-overview">Overview content</div>
      <div id="panel-settings" role="tabpanel" aria-labelledby="tab-settings">Settings content</div>
      <div id="panel-logs" role="tabpanel" aria-labelledby="tab-logs">Logs content</div>
    </div>
  )
  return { ...utils, onChange }
}

describe('Tabs', () => {
  it('renders a tablist container with tab roles', () => {
    renderTabs()
    expect(screen.getByRole('tablist')).toBeInTheDocument()
    expect(screen.getAllByRole('tab')).toHaveLength(tabs.length)
    tabs.forEach((tab) => {
      expect(screen.getByRole('tab', { name: tab.label })).toHaveAttribute('id', `tab-${tab.id}`)
    })
  })

  it('marks the active tab with aria-selected and roving tabindex', () => {
    renderTabs('settings')
    const settings = screen.getByRole('tab', { name: 'Settings' })
    expect(settings).toHaveAttribute('aria-selected', 'true')
    expect(settings).toHaveAttribute('tabindex', '0')
    for (const tab of tabs.filter(t => t.id !== 'settings')) {
      const el = screen.getByRole('tab', { name: tab.label })
      expect(el).toHaveAttribute('aria-selected', 'false')
      expect(el).toHaveAttribute('tabindex', '-1')
    }
  })

  it('links tabs to their panels via aria-controls and aria-labelledby', () => {
    renderTabs()
    const overviewTab = screen.getByRole('tab', { name: 'Overview' })
    expect(overviewTab).toHaveAttribute('aria-controls', 'panel-overview')
    const overviewPanel = document.getElementById('panel-overview')
    expect(overviewPanel).toHaveAttribute('role', 'tabpanel')
    expect(overviewPanel).toHaveAttribute('aria-labelledby', 'tab-overview')
  })

  it('moves selection and focus on ArrowRight', () => {
    const { onChange } = renderTabs('overview')
    fireEvent.keyDown(screen.getByRole('tablist'), { key: 'ArrowRight' })
    expect(onChange).toHaveBeenCalledWith('settings')
    expect(document.activeElement).toBe(screen.getByRole('tab', { name: 'Settings' }))
  })

  it('moves selection on ArrowLeft', () => {
    const { onChange } = renderTabs('settings')
    fireEvent.keyDown(screen.getByRole('tablist'), { key: 'ArrowLeft' })
    expect(onChange).toHaveBeenCalledWith('overview')
    expect(document.activeElement).toBe(screen.getByRole('tab', { name: 'Overview' }))
  })

  it('wraps around at the ends and jumps with Home/End', () => {
    const { onChange } = renderTabs('logs')
    const tablist = screen.getByRole('tablist')
    fireEvent.keyDown(tablist, { key: 'ArrowRight' })
    expect(onChange).toHaveBeenCalledWith('overview')
    fireEvent.keyDown(tablist, { key: 'Home' })
    expect(onChange).toHaveBeenCalledWith('overview')
    fireEvent.keyDown(tablist, { key: 'End' })
    expect(onChange).toHaveBeenCalledWith('logs')
  })

  it('activates a tab on click', () => {
    const { onChange } = renderTabs('overview')
    fireEvent.click(screen.getByRole('tab', { name: 'Logs' }))
    expect(onChange).toHaveBeenCalledWith('logs')
  })
})
