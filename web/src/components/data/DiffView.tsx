import { JsonTree } from "./JsonTree";

type DiffViewProps = {
  before: unknown;
  after: unknown;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function changedPaths(before: unknown, after: unknown, prefix = ""): string[] {
  if (Object.is(before, after)) return [];

  try {
    if (JSON.stringify(before) === JSON.stringify(after)) return [];
  } catch {
    // Fall through to the structural comparison below.
  }

  if (isRecord(before) && isRecord(after)) {
    const keys = new Set([...Object.keys(before), ...Object.keys(after)]);
    return [...keys].flatMap((key) =>
      changedPaths(before[key], after[key], prefix ? `${prefix}.${key}` : key),
    );
  }

  if (Array.isArray(before) && Array.isArray(after)) {
    const length = Math.max(before.length, after.length);
    return Array.from({ length }, (_, index) =>
      changedPaths(before[index], after[index], `${prefix}[${index}]`),
    ).flat();
  }

  return [prefix || "root"];
}

export function DiffView({ before, after }: DiffViewProps) {
  const paths = changedPaths(before, after);

  return (
    <div className="st-diff">
      <div className="st-diff__keys" aria-label="Khóa thay đổi">
        {paths.length === 0 ? (
          <span className="st-diff__key">Không đổi</span>
        ) : (
          paths.map((path) => (
            <span key={path} className="st-diff__key is-changed">
              {path}
            </span>
          ))
        )}
      </div>
      <div className="st-diff__sides">
        <section className="st-diff__side">
          <h4>Trước</h4>
          <JsonTree value={before} defaultExpandDepth={3} />
        </section>
        <section className="st-diff__side">
          <h4>Sau</h4>
          <JsonTree value={after} defaultExpandDepth={3} />
        </section>
      </div>
    </div>
  );
}
