import type { RunStreamEvent } from "../../api/types";
import { formatDurationMs } from "../../lib/format";
import { MethodPill } from "../data/MethodPill";
import { StatusBadge } from "../data/StatusBadge";

type RunTimelineProps = {
  events: RunStreamEvent[];
  live?: boolean;
};

function markerTone(event: RunStreamEvent): string {
  if (event.type === "step_result" || event.type === "scenario_complete") {
    return event.data.passed ? "pass" : "fail";
  }
  if (event.type === "complete") return "pass";
  if (event.type === "error") return "fail";
  if (event.type === "cancelled") return "muted";
  return "live";
}

function EventBody({ event }: { event: RunStreamEvent }) {
  switch (event.type) {
    case "start":
      return (
        <>
          <p className="st-timeline__title">Bắt đầu chạy suite</p>
          <p className="st-timeline__meta">
            {event.data.total_scenarios} kịch bản · {event.data.servers.join(", ")}
          </p>
        </>
      );
    case "scenario_start":
      return (
        <>
          <p className="st-timeline__title">{event.data.scenario_name}</p>
          <p className="st-timeline__meta">
            {event.data.setups_to_run.length > 0
              ? `Setup: ${event.data.setups_to_run.join(", ")}`
              : "Không dùng setup"}
          </p>
        </>
      );
    case "step_result":
      return (
        <>
          <div className="st-timeline__line">
            <MethodPill method={event.data.method} />
            <code className="st-timeline__call">{event.data.call}</code>
          </div>
          <div className="st-timeline__line">
            <StatusBadge value={event.data.passed ? "passed" : "failed"} />
            <span className="st-timeline__meta">
              {event.data.status_code} / {event.data.expected_code} ·{" "}
              {formatDurationMs(event.data.latency_ms)}
            </span>
          </div>
        </>
      );
    case "scenario_complete":
      return (
        <>
          <div className="st-timeline__line">
            <p className="st-timeline__title">Kết thúc {event.data.scenario_id}</p>
            <StatusBadge value={event.data.passed ? "passed" : "failed"} />
          </div>
          <p className="st-timeline__meta">{formatDurationMs(event.data.duration_ms)}</p>
        </>
      );
    case "complete":
      return (
        <>
          <p className="st-timeline__title">
            Hoàn tất · {event.data.passed_scenarios}/{event.data.total_scenarios} kịch bản đạt
          </p>
          <p className="st-timeline__meta">
            {event.data.failed_scenarios} lỗi · {event.data.skipped_scenarios} bỏ qua ·{" "}
            {formatDurationMs(event.data.duration_ms)}
          </p>
        </>
      );
    case "cancelled":
      return (
        <>
          <p className="st-timeline__title">Đã hủy phiên chạy</p>
          {event.message && <p className="st-timeline__meta">{event.message}</p>}
        </>
      );
    case "error":
      return (
        <>
          <p className="st-timeline__title">Lỗi phiên chạy</p>
          <p className="st-timeline__meta">{event.message}</p>
        </>
      );
  }
}

export function RunTimeline({ events, live = false }: RunTimelineProps) {
  if (events.length === 0) {
    return <p className="st-timeline__empty">Chưa có sự kiện nào.</p>;
  }

  return (
    <ol
      className={`st-timeline ${live ? "is-live" : ""}`.trim()}
      aria-live={live ? "polite" : undefined}
      aria-label="Diễn tiến lần chạy"
    >
      {events.map((event, index) => (
        <li
          key={`${event.type}-${index}`}
          className={`st-timeline__item st-timeline__item--${markerTone(event)}`}
        >
          <span className="st-timeline__marker" aria-hidden="true" />
          <EventBody event={event} />
        </li>
      ))}
    </ol>
  );
}
