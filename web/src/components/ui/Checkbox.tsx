import { forwardRef, type InputHTMLAttributes, type ReactNode } from "react";

type CheckboxProps = Omit<InputHTMLAttributes<HTMLInputElement>, "type"> & {
  children: ReactNode;
};

export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(function Checkbox(
  { children, className, ...props },
  ref,
) {
  return (
    <label className={`st-checkbox ${className ?? ""}`.trim()}>
      <input ref={ref} type="checkbox" {...props} />
      <span>{children}</span>
    </label>
  );
});
