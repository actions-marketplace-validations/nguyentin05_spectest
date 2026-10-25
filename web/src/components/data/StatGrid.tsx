import type { ReactNode } from "react";

export type StatItem = {
  id: string;
  label: string;
  value: ReactNode;
  hint?: string;
};

export function StatGrid({ items }: { items: StatItem[] }) {
  return (
    <dl className="st-stat-grid">
      {items.map((item) => (
        <div key={item.id} className="st-stat" data-testid={`stat-${item.id}`}>
          <dt className="st-stat__label">{item.label}</dt>
          <dd className="st-stat__value">{item.value}</dd>
          {item.hint && <dd className="st-stat__hint">{item.hint}</dd>}
        </div>
      ))}
    </dl>
  );
}
