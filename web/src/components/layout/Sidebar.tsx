import { NavLink } from "react-router-dom";

export type NavItem = { to: string; label: string };

export const NAV_ITEMS: NavItem[] = [
  { to: "/", label: "Tổng quan" },
  { to: "/suite", label: "Kịch bản" },
  { to: "/generate", label: "Sinh kịch bản" },
  { to: "/run", label: "Chạy thử" },
  { to: "/history", label: "Lịch sử" },
  { to: "/workspace", label: "Tệp" },
  { to: "/settings", label: "Cài đặt" },
];

export function Sidebar() {
  return (
    <aside className="st-rail">
      <p className="st-brand">
        <span className="st-brand__mark" aria-hidden="true" />
        SpecTest
      </p>
      <nav className="st-rail__nav" aria-label="Điều hướng chính">
        {NAV_ITEMS.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === "/"}
            className={({ isActive }) => (isActive ? "st-nav__link is-active" : "st-nav__link")}
          >
            {item.label}
          </NavLink>
        ))}
      </nav>
      <p className="st-rail__footer">Kiểm thử theo đặc tả</p>
    </aside>
  );
}
