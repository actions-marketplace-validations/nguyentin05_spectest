import type { ReactNode } from "react";

export type BadgeTone = "neutral" | "brand" | "pass" | "fail" | "live" | "skip";

type BadgeProps = {
  children: ReactNode;
  tone?: BadgeTone;
  mono?: boolean;
  className?: string;
  title?: string;
};

export function Badge({ children, tone = "neutral", mono = false, className, title }: BadgeProps) {
  return (
    <span
      className={`st-badge st-badge--${tone} ${mono ? "st-badge--mono" : ""} ${className ?? ""}`.trim()}
      title={title}
    >
      {children}
    </span>
  );
}
