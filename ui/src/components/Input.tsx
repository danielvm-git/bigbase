import type {
  InputHTMLAttributes,
  TextareaHTMLAttributes,
  SelectHTMLAttributes,
  ReactElement,
} from 'react'

interface CommonProps {
  label?: string
  error?: string
  hint?: string
  /** Render the input value in a monospace font (SQL editors, env vars). */
  mono?: boolean
}

type InputAsInput = Omit<CommonProps, 'prefix'> &
  Omit<InputHTMLAttributes<HTMLInputElement>, 'prefix'> & {
    as?: 'input'
    /** Icon or short text rendered inside the input field, before the value. */
    prefix?: ReactElement
  }

type InputAsTextarea = CommonProps &
  TextareaHTMLAttributes<HTMLTextAreaElement> & { as: 'textarea' }

type InputAsSelect = CommonProps &
  SelectHTMLAttributes<HTMLSelectElement> & { as: 'select' }

type InputProps = InputAsInput | InputAsTextarea | InputAsSelect

export function Input(props: InputProps) {
  const { label, error, hint, mono = false, className = '', ...rest } = props
  const id = props.id || props.name
  const inputClass =
    `input ${error ? 'input-error ' : ''}${mono ? 'input-mono ' : ''}${className}`.trim()

  // `prefix` is only valid for the input variant; we render it as a
  // sibling element above the field rather than as an HTML attribute
  // (and we have to Omit it from the spread because React 19's
  // InputHTMLAttributes includes a native `prefix: string` HTML
  // attribute that would type-conflict with our `prefix?: ReactElement`).
  // textarea and select variants don't carry `prefix` in their type.
  const isInputVariant = props.as !== 'textarea' && props.as !== 'select'
  const prefix = isInputVariant ? props.prefix : undefined

  // Strip `prefix` from the rest object before spreading onto the
  // native <input>. The `_` rename flags the intent (this is the
  // standard "I know I'm dropping this key" convention); the eslint
  // disable is the project's own noUnusedLocals not honouring that
  // convention yet. Tracked to lift when the lint config catches up.
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { prefix: _stripPrefix, ...inputAttrs } =
    (rest as InputAsInput & { prefix?: ReactElement })
  void _stripPrefix

  const inputElement =
    props.as === 'textarea' ? (
      <textarea id={id} className={inputClass} {...(rest as InputAsTextarea)} />
    ) : props.as === 'select' ? (
      <select id={id} className={inputClass} {...(rest as InputAsSelect)}>
        {(rest as InputAsSelect).children}
      </select>
    ) : (
      <input id={id} className={inputClass} {...inputAttrs} />
    )

  return (
    <div className="input-group">
      {label && (
        <label htmlFor={id} className="input-label">
          {label}
        </label>
      )}
      {prefix ? (
        <div className="input-with-prefix">
          <span className="input-prefix" aria-hidden="true">
            {prefix}
          </span>
          {inputElement}
        </div>
      ) : (
        inputElement
      )}
      {hint && !error && <span className="input-hint">{hint}</span>}
      {error && <span className="input-error-text">{error}</span>}
    </div>
  )
}
