import type { RunReportRecord } from "../../api/types";
import { formatDurationMs } from "../../lib/format";
import { JsonTree } from "../data/JsonTree";
import { MethodPill } from "../data/MethodPill";
import { StatusBadge } from "../data/StatusBadge";

function parsePayload(value: string): unknown {
  try {
    return JSON.parse(value) as unknown;
  } catch {
    return value;
  }
}

function Payload({ label, raw }: { label: string; raw: string }) {
  if (!raw) return <p className="st-report__empty">Không có dữ liệu.</p>;
  return <JsonTree value={parsePayload(raw)} name={label} />;
}

export function ReportTable({ records }: { records: RunReportRecord[] }) {
  if (records.length === 0) {
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
            {records.map((record, index) => (
              <tr key={`${record.scenario_id}-${record.step_index}-${index}`}>
                <td>
                  <span className="st-report__scenario">{record.scenario_name}</span>
                  <code className="st-report__id">{record.scenario_id}</code>
                </td>
                <td className="st-report__step">#{record.step_index}</td>
                <td>
                  <div className="st-report__call">
                    <MethodPill method={record.method} />
                    <code>{record.call}</code>
                  </div>
                  <details className="st-report__details">
                    <summary>Yêu cầu / Phản hồi</summary>
                    <div className="st-report__details-body">
                      <p className="st-report__url">
                        <span>URL</span>
                        <code>{record.url}</code>
                      </p>
                      <Payload label="request_payload" raw={record.request_payload} />
                      <Payload label="response_body" raw={record.response_body} />
                      {record.fail_reason && (
                        <p className="st-report__reason">
                          <strong>Lỗi:</strong> {record.fail_reason}
                        </p>
                      )}
                      {record.ai_diagnosis && (
                        <p className="st-report__reason">
                          <strong>Chẩn đoán AI:</strong> {record.ai_diagnosis}
                        </p>
                      )}
                    </div>
                  </details>
                </td>
                <td>
                  <span className="st-report__codes">
                    <code>{record.status_code}</code>
                    <span aria-hidden="true"> / </span>
                    <code>{record.expected_code}</code>
                  </span>
                </td>
                <td className="st-report__latency">{formatDurationMs(record.latency_ms)}</td>
                <td>
                  <StatusBadge value={record.passed ? "passed" : "failed"} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
