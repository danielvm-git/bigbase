import type { ButtonHTMLAttributes, AnchorHTMLAttributes, ReactNode } from 'react'

type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'link' | 'ghost'
type ButtonSize = 'sm' | 'md' | 'block'
/** Density opt-out for the 2.5.5 spacing exception (e87s03):
 * `compact` drops the 44px min-height for row-level text actions
 * (e.g. table `.actions-cell`). Default keeps ≥44×44 targets. */
type ButtonDensity = 'default' | 'compact'

interface ButtonBase {
  variant?: ButtonVariant
  size?: ButtonSize
  density?: ButtonDensity
  children: ReactNode
}

type ButtonAsButton = ButtonBase & ButtonHTMLAttributes<HTMLButtonElement> & { as?: 'button' }
type ButtonAsLink = ButtonBase & AnchorHTMLAttributes<HTMLAnchorElement> & { as: 'a'; href: string }
type ButtonProps = ButtonAsButton | ButtonAsLink

const variantClass: Record<ButtonVariant, string> = {
  primary: 'btn-primary',
  secondary: 'btn-secondary',
  danger: 'btn-danger',
  link: 'btn-link',
  ghost: 'btn-ghost',
}

export function Button(props: ButtonProps) {
  const { variant = 'primary', size = 'md', density = 'default', children, className = '' } = props
  const classes = ['btn', variantClass[variant], size === 'sm' ? 'btn-sm' : '', size === 'block' ? 'btn-block' : '', density === 'compact' ? 'btn-compact' : '', className]
    .filter(Boolean)
    .join(' ')

  if (props.as === 'a') {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { as: _as, ...anchorProps } = props as ButtonAsLink
    return (
      <a className={classes} {...anchorProps}>
        {children}
      </a>
    )
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { as: _as, ...buttonProps } = props as ButtonAsButton
  return (
    <button className={classes} {...buttonProps}>
      {children}
    </button>
  )
}
