import { Link } from "react-router-dom";
import type { RunReport, RunStreamEvent } from "../api/types";
import { api } from "../api/endpoints";
import { ApiError, isMissingEndpoint } from "../api/errors";
import { useApi } from "../hooks/useApi";
import { formatDateTime, formatDurationMs } from "../lib/format";
import { Badge } from "../components/ui/Badge";
import { CapabilityNotice } from "../components/ui/CapabilityNotice";
import { Card } from "../components/ui/Card";
import { EmptyState } from "../components/ui/EmptyState";
import { ErrorState } from "../components/ui/ErrorState";
import { LoadingBlock } from "../components/ui/LoadingBlock";
import { StatGrid } from "../components/data/StatGrid";
import { StatusBadge } from "../components/data/StatusBadge";
import { RunSummary } from "../components/run/RunSummary";
import { RunTimeline } from "../components/run/RunTimeline";

function SectionFailure({
  error,
  feature,
  onRetry,
}: {
  error: Error;
  feature: string;
  onRetry: () => void;
}) {
  if (isMissingEndpoint(error)) {
    return <CapabilityNotice feature={feature} error={error} />;
  }
  return <ErrorState error={error} onRetry={onRetry} />;
}

function HealthStrip() {
  const { data, error, loading, reload } = useApi(api.health);

  if (loading) return <LoadingBlock label="Đang kiểm tra máy chủ…" rows={1} />;
  if (error) {
    return (
      <SectionFailure error={error} feature="trạng thái máy chủ" onRetry={reload} />
    );
  }
  if (!data) return null;

  return (
    <section className="st-dashboard__status" aria-label="Trạng thái máy chủ">
      <StatusBadge value={data.status === "ok" ? "ready" : "down"} />
      <p className="st-dashboard__status-label">
        Thư mục làm việc <code className="st-dashboard__workdir">{data.work_dir}</code>
      </p>
      <p className="st-dashboard__muted">Cập nhật {formatDateTime(data.time)}</p>
    </section>
  );
}

function QuickRunCard() {
  const { data, error, loading, reload } = useApi(api.report);

  let body;
  if (loading) {
    body = <LoadingBlock rows={2} />;
  } else if (error instanceof ApiError && error.status === 404) {
    body = (
      <p className="st-dashboard__muted">
        Chưa có lần chạy nào. Mở bàn chạy để kiểm thử suite hiện tại.
      </p>
    );
  } else if (error) {
    body = <SectionFailure error={error} feature="báo cáo lần chạy" onRetry={reload} />;
  } else if (data) {
    body = <RunSummary report={data} />;
  }

  return (
    <Card
      title="Chạy thử"
      subtitle="Kiểm thử toàn bộ kịch bản trên môi trường thật."
      actions={
        <Link className="st-btn st-btn--primary" to="/run">
          Mở bàn chạy
        </Link>
      }
    >
      {body}
    </Card>
  );
}

function PlanSection() {
  const { data, error, loading, reload } = useApi(api.plan);

  if (loading) return <LoadingBlock label="Đang đọc spectest.yaml…" rows={2} />;

  if (error instanceof ApiError && error.status === 404) {
    return (
      <EmptyState
        title="Chưa có spectest.yaml"
        description="Hãy sinh kịch bản đầu tiên hoặc chọn file khác."
        action={
          <Link className="st-btn st-btn--secondary" to="/generate">
            Sinh kịch bản đầu tiên
          </Link>
        }
      />
    );
  }

  if (error) {
    return <SectionFailure error={error} feature="kịch bản" onRetry={reload} />;
  }

  if (!data) return null;

  return (
    <section className="st-dashboard__section" aria-labelledby="dashboard-plan">
      <div className="st-dashboard__section-head">
        <h2 id="dashboard-plan">Quy mô suite</h2>
        <p className="st-dashboard__muted">
          {data.path} · phiên bản {data.version}
        </p>
      </div>
      <StatGrid
        items={[
          { id: "scenarios", label: "Kịch bản", value: data.stats.total_scenarios },
          { id: "setups", label: "Setup", value: data.stats.total_setups },
          { id: "steps", label: "Bước", value: data.stats.total_steps },
          { id: "locked", label: "Đã khóa", value: data.stats.locked_scenarios },
        ]}
      />
      <div className="st-dashboard__scenarios">
        <span className="st-dashboard__muted">Kịch bản:</span>
        {data.scenarios.map((scenario) => (
          <Badge key={scenario.id} tone={scenario.locked ? "neutral" : "brand"}>
            {scenario.name}
          </Badge>
        ))}
      </div>
    </section>
  );
}

function reportEvents(report: RunReport): RunStreamEvent[] {
  return report.records.map((record) => ({
    type: "step_result",
    timestamp: report.start_time,
    data: {
      scenario_id: record.scenario_id,
      call: record.call,
      method: record.method,
      url: record.url,
      status_code: record.status_code,
      expected_code: record.expected_code,
      latency_ms: record.latency_ms,
      passed: record.passed,
    },
  }));
}

function LastRunSection() {
  const { data, error, loading, reload } = useApi(api.report);

  if (loading || (error instanceof ApiError && error.status === 404)) return null;
  if (error) {
    return <SectionFailure error={error} feature="báo cáo lần chạy" onRetry={reload} />;
  }
  if (!data) return null;

  return (
    <section className="st-dashboard__section" aria-labelledby="dashboard-last-run">
      <div className="st-dashboard__section-head">
        <h2 id="dashboard-last-run">Lần chạy gần nhất</h2>
        <p className="st-dashboard__muted">
          {formatDateTime(data.start_time)} · {formatDurationMs(data.duration_ms)}
        </p>
      </div>
      <RunTimeline events={reportEvents(data).slice(-8)} />
    </section>
  );
}

function HistorySection() {
  const { data, error, loading, reload } = useApi(api.history);

  let body;
  if (loading) {
    body = <LoadingBlock rows={3} />;
  } else if (error instanceof ApiError && error.status === 404) {
    body = <p className="st-dashboard__muted">Chưa có lịch sử thay đổi</p>;
  } else if (error) {
    body = <SectionFailure error={error} feature="lịch sử thay đổi" onRetry={reload} />;
  } else if (!data || data.updates.length === 0) {
    body = <p className="st-dashboard__muted">Chưa có lịch sử thay đổi</p>;
  } else {
    const updates = [...data.updates].sort((left, right) => right.n - left.n).slice(0, 3);
    body = (
      <ol className="st-dashboard__list">
        {updates.map((update) => (
          <li key={update.n} className="st-dashboard__item">
            <p className="st-dashboard__item-title">{update.note}</p>
            <p className="st-dashboard__muted">
              {formatDateTime(update.at)} · {update.by}
            </p>
          </li>
        ))}
      </ol>
    );
  }

  return (
    <Card
      title="Thay đổi gần đây"
      actions={<Link to="/history">Xem tất cả</Link>}
    >
      {body}
    </Card>
  );
}

function LogsSection() {
  const { data, error, loading, reload } = useApi(api.logs);

  let body;
  if (loading) {
    body = <LoadingBlock rows={3} />;
  } else if (error) {
    body = <SectionFailure error={error} feature="nhật ký" onRetry={reload} />;
  } else if (!data || data.logs.length === 0) {
    body = <p className="st-dashboard__muted">Chưa có dòng nhật ký</p>;
  } else {
    body = (
      <ol className="st-dashboard__logs" aria-label="Năm dòng nhật ký gần nhất">
        {data.logs.slice(-5).map((line, index) => (
          <li key={`${index}-${line}`}>{line}</li>
        ))}
      </ol>
    );
  }

  return (
    <Card title="Nhật ký" subtitle="5 dòng gần nhất">
      {body}
    </Card>
  );
}

export function DashboardPage() {
  return (
    <section className="st-page">
      <header className="st-page__head">
        <h1>Tổng quan</h1>
        <p className="st-page__subtitle">
          Theo dõi suite, lần chạy gần nhất và thay đổi đặc tả.
        </p>
      </header>

      <HealthStrip />

      <div className="st-dashboard__grid">
        <div className="st-dashboard__main">
          <QuickRunCard />
          <PlanSection />
          <LastRunSection />
        </div>
        <aside className="st-dashboard__side" aria-label="Hoạt động gần đây">
          <HistorySection />
          <LogsSection />
        </aside>
      </div>
    </section>
  );
}
