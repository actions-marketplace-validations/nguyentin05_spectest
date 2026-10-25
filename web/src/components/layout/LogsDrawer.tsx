import { useEffect, useId, useRef } from "react";
import { api } from "../../api/endpoints";
import { useApi } from "../../hooks/useApi";
import { Button } from "../ui/Button";
import { ErrorState } from "../ui/ErrorState";
import { LoadingBlock } from "../ui/LoadingBlock";

type LogsDrawerProps = {
  open: boolean;
  onClose: () => void;
};

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

const LOG_LINE = /^(\[[^\]]+\])\s*(.*)$/;

function LogLine({ line }: { line: string }) {
  const match = LOG_LINE.exec(line);
  if (!match) return <li className="st-logs__line">{line}</li>;

  return (
    <li className="st-logs__line">
      <span className="st-logs__time">{match[1]}</span>{" "}
      <span>{match[2]}</span>
    </li>
  );
}

export function LogsDrawer({ open, onClose }: LogsDrawerProps) {
  const { data, error, loading, reload } = useApi(api.logs);
  const panelRef = useRef<HTMLDivElement>(null);
  const restoreRef = useRef<HTMLElement | null>(null);
  const onCloseRef = useRef(onClose);
  const titleId = useId();
  onCloseRef.current = onClose;

  useEffect(() => {
    if (!open) return;

    restoreRef.current =
      document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const panel = panelRef.current;
    const focusables = () =>
      panel
        ? Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
            (element) => !element.hasAttribute("disabled"),
          )
        : [];

    (focusables()[0] ?? panel)?.focus();

    function handleKeyDown(event: globalThis.KeyboardEvent) {
      if (event.key === "Escape") {
        event.preventDefault();
        onCloseRef.current();
        return;
      }
      if (event.key !== "Tab") return;

      const items = focusables();
      const first = items[0];
      const last = items[items.length - 1];
      if (!first || !last) return;

      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      restoreRef.current?.focus();
    };
  }, [open]);

  if (!open) return null;

  let body;
  if (loading) {
    body = <LoadingBlock label="Đang tải nhật ký…" rows={5} />;
  } else if (error) {
    body = <ErrorState error={error} onRetry={reload} />;
  } else if (!data || data.logs.length === 0) {
    body = <p className="st-logs__empty">Chưa có dòng nhật ký</p>;
  } else {
    body = (
      <ol className="st-logs" aria-label="Nhật ký hệ thống">
        {data.logs.map((line, index) => (
          <LogLine key={`${index}-${line}`} line={line} />
        ))}
      </ol>
    );
  }

  return (
    <div
      className="st-drawer"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        ref={panelRef}
        className="st-drawer__panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
      >
        <header className="st-drawer__head">
          <h2 className="st-drawer__title" id={titleId}>
            Nhật ký hệ thống
          </h2>
          <div className="st-drawer__actions">
            <Button variant="secondary" size="sm" onClick={reload}>
              Làm mới
            </Button>
            <Button variant="ghost" aria-label="Đóng" onClick={onClose}>
              ×
            </Button>
          </div>
        </header>
        <div className="st-drawer__body">{body}</div>
      </div>
    </div>
  );
}
