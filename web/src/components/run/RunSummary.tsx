import type { RunReport } from "../../api/types";
import { formatDateTime, formatDurationMs } from "../../lib/format";
import { StatusBadge } from "../data/StatusBadge";

export function RunSummary({ report }: { report: RunReport }) {
  return (
    <div className="st-run-summary" aria-label="Tóm tắt lần chạy">
      <StatusBadge value={report.status} />
      <div className="st-run-summary__grid">
        <div className="st-run-summary__metric">
          <span className="st-run-summary__label">Kịch bản đạt</span>
          <span className="st-run-summary__value">
            {report.passed_scenarios}/{report.total_scenarios}
          </span>
        </div>
        <div className="st-run-summary__metric">
          <span className="st-run-summary__label">Lỗi</span>
          <span className="st-run-summary__value">{report.failed_scenarios}</span>
        </div>
        <div className="st-run-summary__metric">
          <span className="st-run-summary__label">Bỏ qua</span>
          <span className="st-run-summary__value">{report.skipped_scenarios}</span>
        </div>
        <div className="st-run-summary__metric">
          <span className="st-run-summary__label">Thời lượng</span>
          <span className="st-run-summary__value">{formatDurationMs(report.duration_ms)}</span>
        </div>
      </div>
      <p className="st-run-summary__meta">Bắt đầu {formatDateTime(report.start_time)}</p>
    </div>
  );
}
