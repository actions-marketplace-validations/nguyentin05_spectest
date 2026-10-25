import { cloneElement, isValidElement, type ReactElement, type ReactNode } from "react";

type FieldProps = {
  id: string;
  label: string;
  hint?: ReactNode;
  error?: string;
  children: ReactNode;
  className?: string;
};

export function Field({ id, label, hint, error, children, className }: FieldProps) {
  const hintId = hint ? `${id}-hint` : undefined;
  const errorId = error ? `${id}-error` : undefined;
  const describedBy = [hintId, errorId].filter(Boolean).join(" ") || undefined;

  const control = isValidElement(children)
    ? cloneElement(children as ReactElement<Record<string, unknown>>, {
        id,
        "aria-describedby": describedBy,
        "aria-invalid": error ? true : undefined,
      })
    : children;

  return (
    <div className={`st-field ${className ?? ""}`.trim()}>
      <label className="st-field__label" htmlFor={id}>
        {label}
      </label>
      {control}
      {hint && (
        <p className="st-field__hint" id={hintId}>
          {hint}
        </p>
      )}
      {error && (
        <p className="st-field__error" id={errorId} role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
