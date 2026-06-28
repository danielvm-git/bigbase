import { useState, type KeyboardEvent } from 'react'

interface TagInputProps {
  tags: string[]
  onChange: (tags: string[]) => void
  placeholder?: string
  maxTags?: number
  className?: string
}

export function TagInput({ tags, onChange, placeholder, maxTags, className = '' }: TagInputProps) {
  const [input, setInput] = useState('')

  function addTag(value: string) {
    const trimmed = value.trim().replace(/,+$/, '')
    if (!trimmed) return
    if (tags.includes(trimmed)) return
    if (maxTags && tags.length >= maxTags) return
    onChange([...tags, trimmed])
    setInput('')
  }

  function removeTag(index: number) {
    onChange(tags.filter((_, i) => i !== index))
  }

  function handleKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addTag(input)
      return
    }
    if (e.key === 'Backspace' && input === '' && tags.length > 0) {
      removeTag(tags.length - 1)
    }
  }

  return (
    <div className={`tag-input ${className}`.trim()}>
      <div className="tag-input-tags">
        {tags.map((tag, i) => (
          <span key={tag} className="tag">
            {tag}
            <button
              type="button"
              className="tag-remove"
              aria-label={`Remove ${tag}`}
              onClick={() => removeTag(i)}
            >
              ×
            </button>
          </span>
        ))}
      </div>
      <input
        type="text"
        className="tag-input-field"
        value={input}
        onChange={e => setInput(e.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={() => addTag(input)}
        placeholder={tags.length === 0 ? placeholder : undefined}
        disabled={!!(maxTags && tags.length >= maxTags)}
      />
    </div>
  )
}
