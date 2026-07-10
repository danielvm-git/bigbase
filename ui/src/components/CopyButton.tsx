import { useState } from 'react'

interface CopyButtonProps {
  value: string
  label?: string
  className?: string
}

export function CopyButton({ value, label, className = '' }: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy to clipboard:', err)
      alert('Failed to copy to clipboard. Please try again.')
    }
  }

  return (
    <button
      type="button"
      className={`btn btn-ghost btn-sm copy-button${copied ? ' copy-button-copied' : ''} ${className}`.trim()}
      onClick={handleCopy}
      aria-label={copied ? 'Copied!' : 'Copy to clipboard'}
    >
      {label && <span className="copy-button-label">{label}</span>}
      <span className="copy-button-icon" aria-hidden="true">
        {copied ? '✓' : '⧉'}
      </span>
      {copied && <span className="copy-button-feedback">Copied!</span>}
    </button>
  )
}
