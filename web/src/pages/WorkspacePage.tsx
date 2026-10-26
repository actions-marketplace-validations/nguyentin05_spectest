import { useState, type ReactNode } from "react";
import type { WorkspaceItemType } from "../api/types";
import { api } from "../api/endpoints";
import { isMissingEndpoint } from "../api/errors";
import { useApi } from "../hooks/useApi";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import { CapabilityNotice } from "../components/ui/CapabilityNotice";
import { Button } from "../components/ui/Button";
import { EmptyState } from "../components/ui/EmptyState";
import { ErrorState } from "../components/ui/ErrorState";
import { LoadingBlock } from "../components/ui/LoadingBlock";
import { SearchInput } from "../components/ui/SearchInput";
import { FileTable } from "../components/workspace/FileTable";
import { UploadPanel } from "../components/workspace/UploadPanel";

type FilterId = "all" | WorkspaceItemType;

const FILTERS: { id: FilterId; label: string }[] = [
  { id: "all", label: "Tất cả" },
  { id: "openapi", label: "Đặc tả API" },
  { id: "prd", label: "Yêu cầu" },
  { id: "suite", label: "Kịch bản" },
  { id: "history", label: "Lịch sử" },
  { id: "matrix", label: "Báo cáo" },
  { id: "other", label: "Khác" },
];

export function WorkspacePage() {
  const { data, error, loading, reload } = useApi(api.workspace);
  const [filter, setFilter] = useState<FilterId>("all");
  const [search, setSearch] = useState("");
  const query = useDebouncedValue(search.trim().toLowerCase());

  let content: ReactNode = null;

  if (loading) {
    content = <LoadingBlock label="Đang quét workspace…" rows={3} />;
  } else if (error) {
    content = isMissingEndpoint(error) ? (
      <CapabilityNotice feature="duyệt tệp" error={error} />
    ) : (
      <ErrorState error={error} onRetry={reload} />
    );
  } else if (data) {
    const items = data.items.filter((item) => {
      if (filter !== "all" && item.type !== filter) return false;
      if (!query) return true;
      return (
        item.name.toLowerCase().includes(query) || item.path.toLowerCase().includes(query)
      );
    });

    content = (
      <>
        <p className="st-workspace__current">
          <span>Thư mục:</span>
          <code>{data.current_dir}</code>
        </p>

        {data.items.length === 0 ? (
          <EmptyState
            title="Chưa có tệp nào"
            description="Tải đặc tả API, yêu cầu hoặc kịch bản để bắt đầu."
            action={
              <a className="st-btn st-btn--secondary" href="#upload">
                Tải tệp lên
              </a>
            }
          />
        ) : (
          <>
            <div className="st-workspace__filters" role="group" aria-label="Lọc theo loại tệp">
              {FILTERS.map((option) => (
                <Button
                  key={option.id}
                  variant={filter === option.id ? "primary" : "secondary"}
                  size="sm"
                  aria-pressed={filter === option.id}
                  onClick={() => setFilter(option.id)}
                >
                  {option.label}
                </Button>
              ))}
            </div>

            <div className="st-workspace__toolbar">
              <SearchInput
                aria-label="Tìm tệp"
                placeholder="Tìm theo tên hoặc đường dẫn…"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
              />
              <span className="st-workspace__count">
                {items.length} / {data.items.length} tệp
              </span>
            </div>

            <FileTable items={items} />
          </>
        )}

        <UploadPanel onUploaded={reload} />
      </>
    );
  }

  return (
    <section className="st-page">
      <header className="st-page__head">
        <h1>Tệp</h1>
        <p className="st-page__subtitle">
          Duyệt đặc tả, yêu cầu, kịch bản và lịch sử trong workspace.
        </p>
      </header>
      {content}
    </section>
  );
}
