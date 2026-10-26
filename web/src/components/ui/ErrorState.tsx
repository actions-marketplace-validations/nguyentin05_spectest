import { Button } from "./Button";

type ErrorStateProps = {
  title?: string;
  message?: string;
  error?: unknown;
  onRetry?: () => void;
};

export function ErrorState({ title = "Không tải được dữ liệu", message, error, onRetry }: ErrorStateProps) {
  const detail = message ?? (error instanceof Error ? error.message : undefined);

  return (
    <div className="st-error" role="alert">
      <p className="st-error__title">{title}</p>
      {detail && <p className="st-error__message">{detail}</p>}
      {onRetry && (
        <Button variant="secondary" size="sm" onClick={onRetry}>
          Thử lại
        </Button>
      )}
    </div>
  );
}
