import type { ReactNode } from "react";

type EmptyStateProps = {
  title: string;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
};

export function EmptyState({ title, description, action, className }: EmptyStateProps) {
  return (
    <div className={`st-empty ${className ?? ""}`.trim()} role="status">
      <span className="st-empty__mark" aria-hidden="true" />
      <h2 className="st-empty__title">{title}</h2>
      {description && <p className="st-empty__description">{description}</p>}
      {action && <div className="st-empty__action">{action}</div>}
    </div>
  );
}
