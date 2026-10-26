import { formatDurationMs } from "../../lib/format";

export function Duration({ ms }: { ms: number }) {
  return <span className="st-duration">{formatDurationMs(ms)}</span>;
}
