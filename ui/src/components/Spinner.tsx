type SpinnerSize = 'sm' | 'md' | 'lg'

interface SpinnerProps {
  size?: SpinnerSize
  'aria-label'?: string
  className?: string
}

export function Spinner({ size = 'md', 'aria-label': ariaLabel = 'Loading', className = '' }: SpinnerProps) {
  return (
    <span
      role="status"
      aria-label={ariaLabel}
      className={`spinner spinner-${size} ${className}`.trim()}
    >
      <span className="spinner-inner" aria-hidden="true" />
    </span>
  )
}
