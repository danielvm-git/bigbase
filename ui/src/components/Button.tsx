import type { ButtonHTMLAttributes, AnchorHTMLAttributes, ReactNode } from 'react'

type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'link' | 'ghost'
type ButtonSize = 'sm' | 'md' | 'block'

interface ButtonBase {
  variant?: ButtonVariant
  size?: ButtonSize
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
  const { variant = 'primary', size = 'md', children, className = '' } = props
  const classes = ['btn', variantClass[variant], size === 'sm' ? 'btn-sm' : '', size === 'block' ? 'btn-block' : '', className]
    .filter(Boolean)
    .join(' ')

  if (props.as === 'a') {
    const { as: _, ...anchorProps } = props as ButtonAsLink
    return (
      <a className={classes} {...anchorProps}>
        {children}
      </a>
    )
  }

  const { as: _, ...buttonProps } = props as ButtonAsButton
  return (
    <button className={classes} {...buttonProps}>
      {children}
    </button>
  )
}
