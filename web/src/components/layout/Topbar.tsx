import { useLocation } from "react-router-dom";
import { isMock } from "../../lib/mode";
import { NAV_ITEMS } from "./Sidebar";

function pageTitle(pathname: string): string {
  const navItem = NAV_ITEMS.find((item) => item.to === pathname);
  if (navItem) return navItem.label;
  if (pathname === "/docs") return "Tài liệu";
  return "Không tìm thấy trang";
}

export function Topbar() {
  const { pathname } = useLocation();
  return (
    <header className="st-topbar">
      <p className="st-topbar__title">{pageTitle(pathname)}</p>
      <span className={isMock ? "st-mode-chip is-mock" : "st-mode-chip is-real"}>
        {isMock ? "Chế độ mẫu" : "Backend thật"}
      </span>
    </header>
  );
}
