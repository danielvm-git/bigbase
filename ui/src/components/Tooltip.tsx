import { useState, useId, cloneElement, isValidElement, type ReactNode } from 'react'

type TooltipPlacement = 'top' | 'bottom' | 'left' | 'right'

interface TooltipProps {
  content: ReactNode
  children: ReactNode
  placement?: TooltipPlacement
  className?: string
}

export function Tooltip({ content, children, placement = 'top', className = '' }: TooltipProps) {
  const [visible, setVisible] = useState(false)
  const id = useId()

  function cloneWithAriaDescribedBy(child: ReactNode) {
    if (!isValidElement(child)) return child
    const el = child as React.ReactElement<React.HTMLAttributes<HTMLElement>>
    return cloneElement(el, {
      'aria-describedby': id,
      onMouseEnter: (e: React.MouseEvent<HTMLElement>) => { setVisible(true); el.props.onMouseEnter?.(e) },
      onMouseLeave: (e: React.MouseEvent<HTMLElement>) => { setVisible(false); el.props.onMouseLeave?.(e) },
      onFocus: (e: React.FocusEvent<HTMLElement>) => { setVisible(true); el.props.onFocus?.(e) },
      onBlur: (e: React.FocusEvent<HTMLElement>) => { setVisible(false); el.props.onBlur?.(e) },
    })
  }

  return (
    <span className={`tooltip-wrapper tooltip-${placement} ${className}`.trim()}>
      {cloneWithAriaDescribedBy(children as ReactNode) as ReactNode}
      <span
        id={id}
        role="tooltip"
        className="tooltip-content"
        style={{ visibility: visible ? 'visible' : 'hidden' }}
      >
        {content}
      </span>
    </span>
  )
}
