import { formatBytes } from "../../lib/format";

export function FileSize({ bytes }: { bytes: number }) {
  return <span className="st-file-size">{formatBytes(bytes)}</span>;
}
