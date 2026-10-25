import type { ReactNode } from "react";

export type TimelineTone = "brand" | "pass" | "fail" | "live" | "muted";

export type TimelineItem = {
  id: string;
  tone?: TimelineTone;
  children: ReactNode;
};

type TimelineProps = {
  items: TimelineItem[];
  label?: string;
};

export function Timeline({ items, label }: TimelineProps) {
  return (
    <ol className="st-timeline" aria-label={label}>
      {items.map((item) => (
        <li
          key={item.id}
          className={`st-timeline__item${item.tone ? ` st-timeline__item--${item.tone}` : ""}`}
        >
          <span className="st-timeline__marker" aria-hidden="true" />
          {item.children}
        </li>
      ))}
    </ol>
  );
}
