const viNumber = new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 1 });

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${viNumber.format(value)} ${units[unit] ?? "B"}`;
}

export function formatDurationMs(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return "—";
  if (ms < 1000) return `${viNumber.format(ms)} ms`;
  return `${viNumber.format(ms / 1000)} giây`;
}

export function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const zone = value.match(/(Z|[+-]\d{2}:\d{2})$/)?.[0];
  return new Intl.DateTimeFormat("vi-VN", {
    dateStyle: "short",
    timeStyle: "medium",
    ...(zone ? { timeZone: zone === "Z" ? "UTC" : zone } : {}),
  }).format(date);
}

export function maskMiddle(value: string, head = 6, tail = 4): string {
  if (value.length <= head + tail) return value;
  return `${value.slice(0, head)}...${value.slice(-tail)}`;
}
