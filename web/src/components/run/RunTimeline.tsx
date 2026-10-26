import type { RunStreamEvent } from "../../api/types";
import { RunEventItem } from "./RunEventItem";

type RunTimelineProps = {
  events: RunStreamEvent[];
  live?: boolean;
};

function markerTone(event: RunStreamEvent): string {
  if (event.type === "step_result") {
    return event.data.passed ? "pass" : "fail";
  }
  if (event.type === "scenario_complete") {
    return event.data.skipped ? "muted" : event.data.passed ? "pass" : "fail";
  }
  if (event.type === "complete") return "pass";
  if (event.type === "error") return "fail";
  if (event.type === "cancelled") return "muted";
  return "live";
}

export function RunTimeline({ events, live = false }: RunTimelineProps) {
  const visible = events.filter(
    (event) => event.type !== "connected" && event.type !== "idle",
  );

  if (visible.length === 0) {
    return <p className="st-timeline__empty">Chưa có sự kiện nào.</p>;
  }

  return (
    <ol
      className={`st-timeline ${live ? "is-live" : ""}`.trim()}
      aria-live={live ? "polite" : undefined}
      aria-label="Diễn tiến lần chạy"
    >
      {visible.map((event, index) => (
        <li
          key={`${event.type}-${index}`}
          className={`st-timeline__item st-timeline__item--${markerTone(event)}`}
        >
          <span className="st-timeline__marker" aria-hidden="true" />
          <RunEventItem event={event} />
        </li>
      ))}
    </ol>
  );
}
