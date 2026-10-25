import type { WorkspaceItemType } from "../../api/types";

const LABELS: Record<WorkspaceItemType, string> = {
  openapi: "Đặc tả API",
  prd: "Yêu cầu",
  suite: "Kịch bản",
  history: "Lịch sử",
  matrix: "Ma trận",
  other: "Khác",
};

export function TypeBadge({ type }: { type: WorkspaceItemType }) {
  return <span className={`st-type st-type--${type}`}>{LABELS[type]}</span>;
}
