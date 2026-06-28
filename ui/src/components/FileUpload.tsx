import { useRef, useState, type DragEvent } from 'react'

interface FileUploadProps {
  onFiles: (files: File[]) => void
  accept?: string
  multiple?: boolean
  maxSizeMb?: number
  label?: string
  className?: string
}

export function FileUpload({ onFiles, accept, multiple = false, maxSizeMb, label, className = '' }: FileUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function validate(files: File[]): File[] | null {
    if (maxSizeMb) {
      const maxBytes = maxSizeMb * 1024 * 1024
      const oversized = files.filter(f => f.size > maxBytes)
      if (oversized.length > 0) {
        setError(`File too large. Max size: ${maxSizeMb} MB.`)
        return null
      }
    }
    setError(null)
    return files
  }

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? [])
    const valid = validate(files)
    if (valid) onFiles(valid)
    e.target.value = ''
  }

  function handleDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setDragging(false)
    const files = Array.from(e.dataTransfer.files)
    const valid = validate(files)
    if (valid) onFiles(valid)
  }

  return (
    <div
      className={`file-upload ${dragging ? 'file-upload-dragging' : ''} ${className}`.trim()}
      onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
      onDragLeave={() => setDragging(false)}
      onDrop={handleDrop}
    >
      <input
        ref={inputRef}
        type="file"
        accept={accept}
        multiple={multiple}
        className="file-upload-input"
        style={{ position: 'absolute', opacity: 0, pointerEvents: 'none' }}
        onChange={handleChange}
        aria-hidden="true"
        tabIndex={-1}
      />
      {label && <p className="file-upload-label">{label}</p>}
      <button
        type="button"
        className="btn btn-secondary btn-sm"
        onClick={() => inputRef.current?.click()}
        aria-label="Upload or browse files"
      >
        Browse files
      </button>
      {accept && <p className="file-upload-hint">Accepted: {accept}</p>}
      {maxSizeMb && <p className="file-upload-hint">Max size: {maxSizeMb} MB</p>}
      {error && <p role="alert" className="input-error-message">{error}</p>}
    </div>
  )
}
