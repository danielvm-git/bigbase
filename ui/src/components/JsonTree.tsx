import { useId, useState } from 'react'

type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue }

interface JsonNodeProps {
  data: JsonValue
  depth: number
  maxDepth: number
  keyName?: string
}

function JsonNode({ data, depth, maxDepth, keyName }: JsonNodeProps) {
  const [open, setOpen] = useState(true)
  const regionId = useId()

  const label = keyName !== undefined ? (
    <><span className="json-tree-key">{keyName}</span><span className="json-tree-colon">: </span></>
  ) : null

  if (data === null) {
    return <div className="json-tree-node">{label}<span className="json-tree-null">null</span></div>
  }

  if (typeof data === 'boolean') {
    return <div className="json-tree-node">{label}<span className="json-tree-bool">{String(data)}</span></div>
  }

  if (typeof data === 'number') {
    return <div className="json-tree-node">{label}<span className="json-tree-number">{data}</span></div>
  }

  if (typeof data === 'string') {
    return <div className="json-tree-node">{label}<span className="json-tree-string">&quot;{data}&quot;</span></div>
  }

  const isArray = Array.isArray(data)
  const entries = isArray
    ? (data as JsonValue[]).map((v, i) => [String(i), v] as [string, JsonValue])
    : Object.entries(data as { [key: string]: JsonValue })

  const brackets = isArray ? ['[', ']'] : ['{', '}']
  const tooDeep = depth >= maxDepth
  // Disclosure-pattern accessible name: key (when present) + state + item
  // count, so screen readers announce which node and how much it contains.
  const toggleLabel = `${keyName !== undefined ? `${keyName}: ` : ''}${
    open ? 'collapse' : 'expand'
  } ${entries.length} items`

  return (
    <div className="json-tree-node">
      <button
        type="button"
        className="json-tree-toggle"
        aria-expanded={open}
        aria-controls={open ? regionId : undefined}
        aria-label={toggleLabel}
        onClick={() => setOpen(o => !o)}
      >
        {open ? '▾' : '▸'}
      </button>
      {label}
      <span className="json-tree-bracket">{brackets[0]}</span>
      {(!open || tooDeep) ? (
        <span className="json-tree-collapsed">…</span>
      ) : (
        <div className="json-tree-children" id={regionId}>
          {entries.map(([k, v]) => (
            <JsonNode
              key={k}
              data={v}
              depth={depth + 1}
              maxDepth={maxDepth}
              keyName={isArray ? undefined : k}
            />
          ))}
        </div>
      )}
      <span className="json-tree-bracket">{brackets[1]}</span>
    </div>
  )
}

interface JsonTreeProps {
  data: JsonValue
  maxDepth?: number
  className?: string
}

export function JsonTree({ data, maxDepth = Infinity, className = '' }: JsonTreeProps) {
  return (
    <div className={`json-tree ${className}`.trim()}>
      <JsonNode data={data} depth={0} maxDepth={maxDepth} />
    </div>
  )
}
