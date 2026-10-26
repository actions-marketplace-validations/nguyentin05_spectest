import type { RunCaseResult } from "../../api/types";
import { formatDurationMs } from "../../lib/format";
import { JsonTree } from "../data/JsonTree";
import { MethodPill } from "../data/MethodPill";
import { StatusBadge } from "../data/StatusBadge";

function resultTone(status: RunCaseResult["status"]) {
  if (status === "PASS") return "passed";
  if (status === "FAIL") return "failed";
  return "skipped";
}

export function ReportTable({ results }: { results: RunCaseResult[] }) {
  if (results.length === 0) {
    return <p className="st-report__empty">Báo cáo chưa có bản ghi nào.</p>;
  }

  return (
    <div className="st-report">
      <div className="st-report__scroll">
        <table className="st-report__table" aria-label="Chi tiết báo cáo">
          <thead>
            <tr>
              <th>Kịch bản</th>
              <th>Bước</th>
              <th>Gọi</th>
              <th>Trạng thái</th>
              <th>Thời gian</th>
              <th>Kết quả</th>
            </tr>
          </thead>
          <tbody>
            {results.map((result, index) => (
              <tr key={`${result.scenario_id}-${result.step_id ?? index}`}>
                <td>
                  <code className="st-report__id">{result.scenario_id}</code>
                </td>
                <td className="st-report__step">
                  {result.step_id ?? result.case_id ?? "—"}
                </td>
                <td>
                  <div className="st-report__call">
                    <MethodPill method={result.method} />
                    <code>{result.call ?? result.endpoint ?? "—"}</code>
                  </div>
                  <details className="st-report__details">
                    <summary>Chi tiết bước</summary>
                    <div className="st-report__details-body">
                      <p className="st-report__url">
                        <span>URL</span>
                        <code>{result.url ?? "—"}</code>
                      </p>
                      {result.saved_vars &&
                        Object.keys(result.saved_vars).length > 0 && (
                          <JsonTree value={result.saved_vars} name="saved_vars" />
                        )}
                      {result.error && (
                        <p className="st-report__reason">
                          <strong>Lỗi:</strong> {result.error}
                        </p>
                      )}
                    </div>
                  </details>
                </td>
                <td>
                  <span className="st-report__codes">
                    <code>{result.http_status}</code>
                    <span aria-hidden="true"> / </span>
                    <code>{result.expected_http_status}</code>
                  </span>
                </td>
                <td className="st-report__latency">
                  {formatDurationMs(result.latency_ms)}
                </td>
                <td>
                  <StatusBadge value={resultTone(result.status)} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
