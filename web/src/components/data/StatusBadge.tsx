export type StatusValue =
  | "passed"
  | "failed"
  | "skipped"
  | "running"
  | "cancelled"
  | "ready"
  | "down";

const LABELS: Record<StatusValue, string> = {
  passed: "Đạt",
  failed: "Lỗi",
  skipped: "Bỏ qua",
  running: "Đang chạy",
  cancelled: "Đã hủy",
  ready: "Sẵn sàng",
  down: "Không kết nối",
};

export function StatusBadge({ value }: { value: StatusValue }) {
  return (
    <span className={`st-status st-status--${value}`}>
      <span className="st-status__dot" aria-hidden="true" />
      {LABELS[value]}
    </span>
  );
}
