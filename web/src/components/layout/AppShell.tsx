import { useEffect, useState, type ReactNode } from "react";
import { LogsDrawer } from "./LogsDrawer";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";
import { ToastProvider } from "../ui/Toast";

export function AppShell({ children }: { children: ReactNode }) {
  const [logsOpen, setLogsOpen] = useState(false);
  const [navOpen, setNavOpen] = useState(false);

  useEffect(() => {
    if (!navOpen) return;

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key !== "Escape") return;
      event.preventDefault();
      setNavOpen(false);
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [navOpen]);

  return (
    <ToastProvider>
      <a className="st-skip-link" href="#main">
        Bỏ qua điều hướng
      </a>
      <div className="st-shell">
        <Sidebar open={navOpen} onNavigate={() => setNavOpen(false)} />
        <div className="st-shell__body">
          <Topbar
            navOpen={navOpen}
            onToggleNav={() => setNavOpen((current) => !current)}
            onOpenLogs={() => setLogsOpen(true)}
          />
          <main id="main" className="st-main">
            {children}
          </main>
        </div>
      </div>
      {navOpen && (
        <button
          type="button"
          className="st-nav-overlay"
          aria-label="Đóng menu"
          onClick={() => setNavOpen(false)}
        />
      )}
      {logsOpen && <LogsDrawer open onClose={() => setLogsOpen(false)} />}
    </ToastProvider>
  );
}
