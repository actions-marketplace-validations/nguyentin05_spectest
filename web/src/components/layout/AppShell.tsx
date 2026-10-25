import { useState, type ReactNode } from "react";
import { LogsDrawer } from "./LogsDrawer";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";
import { ToastProvider } from "../ui/Toast";

export function AppShell({ children }: { children: ReactNode }) {
  const [logsOpen, setLogsOpen] = useState(false);

  return (
    <ToastProvider>
      <a className="st-skip-link" href="#main">
        Bỏ qua điều hướng
      </a>
      <div className="st-shell">
        <Sidebar />
        <div className="st-shell__body">
          <Topbar onOpenLogs={() => setLogsOpen(true)} />
          <main id="main" className="st-main">
            {children}
          </main>
        </div>
      </div>
      {logsOpen && <LogsDrawer open onClose={() => setLogsOpen(false)} />}
    </ToastProvider>
  );
}
