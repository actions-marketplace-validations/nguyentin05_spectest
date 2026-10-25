import { useEffect, useRef, useState, type ReactNode } from "react";
import { useSearchParams } from "react-router-dom";
import type { RunStreamEvent } from "../api/types";
import { useRunStream } from "../hooks/useRunStream";
import { formatDurationMs } from "../lib/format";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { RunForm } from "../components/run/RunForm";
import { RunTimeline } from "../components/run/RunTimeline";

type StartEvent = Extract<RunStreamEvent, { type: "start" }>;
type EndEvent = Extract<RunStreamEvent, { type: "complete" | "cancelled" }>;

function useElapsed(events: RunStreamEvent[]): string {
  const startEvent = events.find(
    (event): event is StartEvent => event.type === "start",
  );
  const endEvent = [...events]
    .reverse()
    .find(
      (event): event is EndEvent =>
        event.type === "complete" || event.type === "cancelled",
    );
  const [now, setNow] = useState(() => Date.now());
  const running = Boolean(startEvent && !endEvent);

  useEffect(() => {
    if (!running) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [running]);

  if (!startEvent) return "0 ms";
  const start = Date.parse(startEvent.timestamp);
  const end = endEvent ? Date.parse(endEvent.timestamp) : now;
  if (Number.isNaN(start) || Number.isNaN(end)) return "0 ms";
  return formatDurationMs(Math.max(0, end - start));
}

function scenarioCounts(events: RunStreamEvent[]): {
  done: number;
  passed: number;
  failed: number;
} {
  let done = 0;
  let passed = 0;

  for (const event of events) {
    if (event.type !== "scenario_complete") continue;
    done += 1;
    if (event.data.passed) passed += 1;
  }

  return { done, passed, failed: done - passed };
}

export function RunPage() {
  const [params, setParams] = useSearchParams();
  const runId = params.get("run") ?? "";
  const { events, connected, error, start, stop, clear } = useRunStream();
  const streamRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!runId) return;
    clear();
    start();
    return () => stop();
  }, [runId, clear, start, stop]);

  useEffect(() => {
    const element = streamRef.current;
    if (!element) return;
    const distance = element.scrollHeight - element.scrollTop - element.clientHeight;
    if (distance < 120) {
      element.scrollTop = element.scrollHeight;
    }
  }, [events]);

  const startEvent = events.find(
    (event): event is StartEvent => event.type === "start",
  );
  const total = startEvent?.data.total_scenarios ?? 0;
  const { done, passed, failed } = scenarioCounts(events);
  const elapsed = useElapsed(events);

  let live: ReactNode = null;
  if (runId) {
    live = (
      <section className="st-run-live" aria-label="Phiên chạy trực tiếp">
        <header className="st-run-live__head">
          <div>
            <h2 className="st-run-live__title">Phiên chạy trực tiếp</h2>
            <p className="st-run-live__id">
              <code>{runId}</code>
            </p>
          </div>
          <Badge tone={error ? "fail" : connected ? "live" : "skip"}>
            {error ? "Mất kết nối" : connected ? "Đang chạy" : "Đã dừng"}
          </Badge>
        </header>

        <div className="st-run-live__stats" aria-label="Tiến độ phiên chạy">
          <span>
            <strong>
              {done}/{total}
            </strong>{" "}
            kịch bản
          </span>
          <span>
            Đạt <strong>{passed}</strong>
          </span>
          <span>
            Lỗi <strong>{failed}</strong>
          </span>
          <span>
            Thời gian <strong>{elapsed}</strong>
          </span>
        </div>

        {error && (
          <div className="st-run-live__reconnect" role="alert">
            <div>
              <p>Mất kết nối với phiên chạy</p>
              <p className="st-run-live__hint">
                Các sự kiện đã nhận vẫn được giữ lại.
              </p>
            </div>
            <Button variant="secondary" size="sm" onClick={start}>
              Kết nối lại
            </Button>
          </div>
        )}

        <div className="st-run-live__stream" ref={streamRef}>
          <RunTimeline events={events} live={connected && !error} />
        </div>
      </section>
    );
  }

  return (
    <section className="st-page">
      <header className="st-page__head">
        <h1>Chạy thử</h1>
        <p className="st-page__subtitle">
          Chạy suite kiểm thử và theo dõi từng bước theo thời gian thực.
        </p>
      </header>

      <RunForm
        onStarted={(id) => setParams({ run: id }, { replace: true })}
      />

      {live}
    </section>
  );
}
