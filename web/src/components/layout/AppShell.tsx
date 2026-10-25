import type { ReactNode } from "react";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";
import { ToastProvider } from "../ui/Toast";

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <ToastProvider>
      <a className="st-skip-link" href="#main">
        Bỏ qua điều hướng
      </a>
      <div className="st-shell">
        <Sidebar />
        <div className="st-shell__body">
          <Topbar />
          <main id="main" className="st-main">
            {children}
          </main>
        </div>
      </div>
    </ToastProvider>
  );
}
