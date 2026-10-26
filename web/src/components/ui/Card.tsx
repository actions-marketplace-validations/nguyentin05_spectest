import type { ReactNode } from "react";

type CardProps = {
  title?: ReactNode;
  subtitle?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  id?: string;
};

export function Card({ title, subtitle, actions, children, className, id }: CardProps) {
  return (
    <section className={`st-card ${className ?? ""}`.trim()} id={id}>
      {(title || subtitle || actions) && (
        <header className="st-card__head">
          <div className="st-card__heading">
            {title && <h2 className="st-card__title">{title}</h2>}
            {subtitle && <p className="st-card__subtitle">{subtitle}</p>}
          </div>
          {actions && <div className="st-card__actions">{actions}</div>}
        </header>
      )}
      {children}
    </section>
  );
}
