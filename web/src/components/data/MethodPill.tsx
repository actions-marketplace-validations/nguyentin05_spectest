export function MethodPill({ method }: { method: string }) {
  const upper = method.toUpperCase();
  const tone = ["GET", "POST", "PUT", "PATCH", "DELETE"].includes(upper)
    ? upper.toLowerCase()
    : "other";

  return <span className={`st-method st-method--${tone}`}>{upper}</span>;
}
