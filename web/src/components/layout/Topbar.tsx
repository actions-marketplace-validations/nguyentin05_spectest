import { useLocation } from "react-router-dom";
import { isMock } from "../../lib/mode";
import { Button } from "../ui/Button";
import { NAV_ITEMS } from "./Sidebar";

function pageTitle(pathname: string): string {
  const navItem = NAV_ITEMS.find((item) => item.to === pathname);
  if (navItem) return navItem.label;
  if (pathname === "/docs") return "Tài liệu";
  return "Không tìm thấy trang";
}

export function Topbar({ onOpenLogs }: { onOpenLogs: () => void }) {
  const { pathname } = useLocation();
  return (
    <header className="st-topbar">
      <p className="st-topbar__title">{pageTitle(pathname)}</p>
      <div className="st-topbar__actions">
        <Button
          variant="ghost"
          size="sm"
          className="st-topbar__logs"
          aria-label="Nhật ký"
          onClick={onOpenLogs}
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 16 16"
            fill="none"
            aria-hidden="true"
          >
            <path
              d="M2 4h12M2 8h12M2 12h8"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
            />
          </svg>
          <span className="st-topbar__logs-label">Nhật ký</span>
        </Button>
        <span className={isMock ? "st-mode-chip is-mock" : "st-mode-chip is-real"}>
          {isMock ? "Chế độ mẫu" : "Backend thật"}
        </span>
      </div>
    </header>
  );
}
