import { useState, type ReactNode } from "react";
import { Link, useSearchParams } from "react-router-dom";
import type { WorkspaceItem } from "../api/types";
import { api } from "../api/endpoints";
import { ApiError } from "../api/errors";
import { useApi } from "../hooks/useApi";
import { withQuery } from "../lib/url";
import { MarkdownView } from "../components/markdown/MarkdownView";
import { Button } from "../components/ui/Button";
import { CopyButton } from "../components/ui/CopyButton";
import { EmptyState } from "../components/ui/EmptyState";
import { ErrorState } from "../components/ui/ErrorState";
import { LoadingBlock } from "../components/ui/LoadingBlock";

function fileName(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

function DocumentView({ path }: { path: string }) {
  const { data, error, loading, reload } = useApi(() => api.markdown(path), [path]);
  const [showRaw, setShowRaw] = useState(false);

  let content: ReactNode = null;

  if (loading) {
    content = <LoadingBlock label="Đang đọc tài liệu…" rows={4} />;
  } else if (error instanceof ApiError && (error.status === 400 || error.status === 404)) {
    content = (
      <EmptyState
        title="Không tìm thấy tài liệu"
        description={
          <>
            Không mở được <code>{path}</code>. Kiểm tra lại đường dẫn hoặc chọn tệp khác.
          </>
        }
        action={
          <Link className="st-btn st-btn--secondary" to="/docs">
            Chọn tài liệu khác
          </Link>
        }
      />
    );
  } else if (error) {
    content = <ErrorState error={error} onRetry={reload} />;
  } else if (data) {
    content = (
      <>
        <div className="st-docs__meta">
          <p className="st-docs__path">
            <code>{data.path}</code>
            <CopyButton value={data.path} label={`đường dẫn ${data.file_name}`} />
          </p>
          <Button
            variant="secondary"
            size="sm"
            aria-pressed={showRaw}
            onClick={() => setShowRaw((current) => !current)}
          >
            {showRaw ? "Xem bản đọc" : "Xem nội dung gốc"}
          </Button>
        </div>

        {showRaw ? (
          <textarea
            className="st-docs__raw"
            aria-label="Nội dung Markdown gốc"
            readOnly
            value={data.content}
          />
        ) : (
          <article className="st-docs__content">
            <MarkdownView content={data.content} />
          </article>
        )}
      </>
    );
  }

  return <>{content}</>;
}

function groupOf(item: WorkspaceItem): "prd" | "matrix" | "other" {
  if (item.type === "prd") return "prd";
  if (item.type === "matrix") return "matrix";
  return "other";
}

const GROUPS: { id: "prd" | "matrix" | "other"; label: string }[] = [
  { id: "prd", label: "Tài liệu yêu cầu" },
  { id: "matrix", label: "Báo cáo" },
  { id: "other", label: "Khác" },
];

function DocumentPicker() {
  const { data, error, loading, reload } = useApi(api.workspace);

  let content: ReactNode = null;

  if (loading) {
    content = <LoadingBlock label="Đang tải danh sách tài liệu…" rows={2} />;
  } else if (error) {
    content = <ErrorState error={error} onRetry={reload} />;
  } else if (data) {
    const documents = data.items.filter((item) => /\.(md|markdown)$/i.test(item.name));

    content =
      documents.length === 0 ? (
        <EmptyState
          title="Chưa có tài liệu Markdown"
          description="Thêm tệp .md vào workspace để đọc tại đây."
          action={
            <Link className="st-btn st-btn--secondary" to="/workspace">
              Mở workspace
            </Link>
          }
        />
      ) : (
        <div className="st-docs__picker">
          {GROUPS.map((group) => {
            const items = documents.filter((item) => groupOf(item) === group.id);
            if (items.length === 0) return null;

            return (
              <section key={group.id} className="st-docs__group" aria-labelledby={`docs-${group.id}`}>
                <h2 className="st-docs__group-title" id={`docs-${group.id}`}>
                  {group.label}
                </h2>
                <ul>
                  {items.map((item) => (
                    <li key={item.path}>
                      <Link to={withQuery("/docs", { path: item.name })}>{item.name}</Link>
                    </li>
                  ))}
                </ul>
              </section>
            );
          })}
        </div>
      );
  }

  return <>{content}</>;
}

export function DocsPage() {
  const [params] = useSearchParams();
  const path = params.get("path")?.trim() ?? "";

  return (
    <section className="st-page">
      <header className="st-page__head">
        {path && (
          <nav className="st-docs__crumb" aria-label="Đường dẫn">
            <Link to="/workspace">Tệp</Link>
            <span aria-hidden="true">/</span>
            <span aria-current="page">{fileName(path)}</span>
          </nav>
        )}
        <h1>Tài liệu</h1>
        <p className="st-page__subtitle">
          Đọc đặc tả và báo cáo Markdown ngay trong workspace.
        </p>
      </header>
      {path ? <DocumentView path={path} /> : <DocumentPicker />}
    </section>
  );
}
