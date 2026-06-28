interface CodeBlockProps {
  code: string
  language?: string
  title?: string
  showLineNumbers?: boolean
  maxHeight?: string
  className?: string
}

export function CodeBlock({
  code,
  language,
  title,
  showLineNumbers = false,
  maxHeight,
  className = '',
}: CodeBlockProps) {
  const lines = code.split('\n')

  return (
    <div className={`code-block ${className}`.trim()}>
      {(title || language) && (
        <div className="code-block-header">
          {title && <span className="code-block-title">{title}</span>}
          {language && <span className="code-block-language">{language}</span>}
        </div>
      )}
      <pre
        className="code-block-pre"
        style={maxHeight ? { maxHeight, overflowY: 'auto' } : undefined}
      >
        {showLineNumbers ? (
          <code className="code-block-code">
            {lines.map((line, i) => (
              <span key={i} className="code-block-line">
                <span className="code-block-line-number">{i + 1}</span>
                <span className="code-block-line-content">{line}</span>
              </span>
            ))}
          </code>
        ) : (
          <code className="code-block-code">{code}</code>
        )}
      </pre>
    </div>
  )
}
