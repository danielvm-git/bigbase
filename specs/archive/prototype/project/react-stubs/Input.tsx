import React, { forwardRef, useId } from 'react';

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  /** Field label (rendered as a real <label> tied to the control). */
  label?: string;
  /** Inline error message — sets aria-invalid and the error styling. */
  error?: string;
  /** Helper text shown beneath the field when there is no error. */
  hint?: string;
  /** Non-editable affix joined to the left edge (e.g. "POST", "bigbase.local/"). */
  prefix?: string;
  /** Monospace the input value (URLs, SHAs, keys). */
  mono?: boolean;
}

/**
 * Input — text field with label, hint, and inline error. Error surfaces as
 * adjacent text (never color or placeholder alone) and wires aria-invalid +
 * aria-describedby for screen readers. Focus draws the accent border + ring.
 */
export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { label, error, hint, prefix, mono = false, id, className = '', ...rest },
  ref,
) {
  const autoId = useId();
  const inputId = id ?? autoId;
  const describedBy = error ? `${inputId}-err` : hint ? `${inputId}-hint` : undefined;
  const cls = ['input', mono ? 'mono' : '', error ? 'input-error' : '', className].filter(Boolean).join(' ');

  const control = (
    <input
      ref={ref}
      id={inputId}
      className={cls}
      aria-invalid={error ? true : undefined}
      aria-describedby={describedBy}
      {...rest}
    />
  );

  return (
    <div className="input-group">
      {label && <label className="input-label" htmlFor={inputId}>{label}</label>}
      {prefix
        ? <div className="input-with-prefix"><span className="input-prefix">{prefix}</span>{control}</div>
        : control}
      {error
        ? <div className="input-error-text" id={`${inputId}-err`} role="alert">{error}</div>
        : hint
          ? <div className="input-hint" id={`${inputId}-hint`}>{hint}</div>
          : null}
    </div>
  );
});
