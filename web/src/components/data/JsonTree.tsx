import { useState } from "react";
import { CopyButton } from "../ui/CopyButton";

type JsonTreeProps = {
  value: unknown;
  name?: string;
  defaultExpandDepth?: number;
  truncateAt?: number;
};

type JsonNodeProps = {
  label: string;
  value: unknown;
  depth: number;
  defaultExpandDepth: number;
  truncateAt: number;
};

const MAX_ARRAY_ITEMS = 20;
// shortcut: lists render fully, virtualize when a real workspace exceeds ~500 items

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function containerSize(value: unknown[] | Record<string, unknown>): number {
  return Array.isArray(value) ? value.length : Object.keys(value).length;
}

function entriesOf(value: unknown[] | Record<string, unknown>): [string, unknown][] {
  return Array.isArray(value)
    ? value.map((item, index) => [String(index), item])
    : Object.entries(value);
}

function PrimitiveRow({
  label,
  value,
  truncateAt,
}: {
  label: string;
  value: unknown;
  truncateAt: number;
}) {
  const [expanded, setExpanded] = useState(false);
  const kind = value === null ? "null" : typeof value;
  const text = value === null ? "null" : String(value);
  const isLong = text.length > truncateAt;
  const shown = isLong && !expanded ? `${text.slice(0, truncateAt)}…` : text;

  return (
    <div className="st-json__row st-json__row--primitive">
      <span className="st-json__key">{label}</span>
      <span className={`st-json__value st-json__value--${kind}`}>{shown}</span>
      {isLong && (
        <button
          type="button"
          className="st-json__more"
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? "thu gọn" : "xem thêm"}
        </button>
      )}
      <span className={`st-json__type st-json__type--${kind}`}>{kind}</span>
      <CopyButton value={text} label={label} />
    </div>
  );
}

function JsonNode({ label, value, depth, defaultExpandDepth, truncateAt }: JsonNodeProps) {
  // Depth 1 is the root's first child; the default depth of 2 keeps it collapsed.
  const [expanded, setExpanded] = useState(depth < defaultExpandDepth - 1);
  const [showAll, setShowAll] = useState(false);

  const isArray = Array.isArray(value);
  const isObject = isRecord(value);

  if (!isArray && !isObject) {
    return <PrimitiveRow label={label} value={value} truncateAt={truncateAt} />;
  }

  const entries = entriesOf(value);
  const shown = showAll ? entries : entries.slice(0, MAX_ARRAY_ITEMS);
  const hidden = entries.length - shown.length;
  const copyValue = (() => {
    try {
      return JSON.stringify(value, null, 2) ?? String(value);
    } catch {
      return String(value);
    }
  })();

  return (
    <div className="st-json__node">
      <div className="st-json__row">
        <button
          type="button"
          className="st-json__toggle"
          aria-expanded={expanded}
          aria-label={`${expanded ? "Đóng" : "Mở"} ${label}`}
          onClick={() => setExpanded((current) => !current)}
        >
          <span className="st-json__chevron" aria-hidden="true">
            {expanded ? "▾" : "▸"}
          </span>
          <span className="st-json__key">{label}</span>
          <span className="st-json__meta">
            {containerSize(value)} {isArray ? "mục" : "khóa"}
          </span>
        </button>
        <CopyButton value={copyValue} label={label} />
      </div>

      {expanded && (
        <div className="st-json__children">
          {shown.map(([key, child]) => (
            <JsonNode
              key={key}
              label={key}
              value={child}
              depth={depth + 1}
              defaultExpandDepth={defaultExpandDepth}
              truncateAt={truncateAt}
            />
          ))}
          {hidden > 0 && (
            <button type="button" className="st-json__more" onClick={() => setShowAll(true)}>
              Xem thêm {hidden} mục
            </button>
          )}
        </div>
      )}
    </div>
  );
}

export function JsonTree({
  value,
  name,
  defaultExpandDepth = 2,
  truncateAt = 120,
}: JsonTreeProps) {
  const isContainer = Array.isArray(value) || isRecord(value);

  return (
    <div className="st-json" aria-label={name ? `${name} JSON` : "JSON"}>
      {name && <p className="st-json__root-name">{name}</p>}
      {isContainer ? (
        <div className="st-json__children st-json__children--root">
          {entriesOf(value).map(([key, child]) => (
            <JsonNode
              key={key}
              label={key}
              value={child}
              depth={1}
              defaultExpandDepth={defaultExpandDepth}
              truncateAt={truncateAt}
            />
          ))}
        </div>
      ) : (
        <PrimitiveRow label={name ?? "giá trị"} value={value} truncateAt={truncateAt} />
      )}
    </div>
  );
}
