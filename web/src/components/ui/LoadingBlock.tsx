type LoadingBlockProps = {
  label?: string;
  rows?: number;
};

export function LoadingBlock({ label = "Đang tải…", rows = 3 }: LoadingBlockProps) {
  return (
    <div className="st-loading" role="status" aria-live="polite">
      <span className="st-loading__label">{label}</span>
      <div className="st-loading__bars" aria-hidden="true">
        {Array.from({ length: rows }, (_, index) => (
          <span key={index} className="st-loading__bar" />
        ))}
      </div>
    </div>
  );
}
