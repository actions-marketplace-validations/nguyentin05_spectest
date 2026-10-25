import { ApiError } from "./errors";

const API_BASE = "/api";

type ErrorEnvelope = { success: false; error: string };

function isErrorEnvelope(value: unknown): value is ErrorEnvelope {
  if (typeof value !== "object" || value === null) return false;
  const candidate = value as { success?: unknown; error?: unknown };
  return candidate.success === false && typeof candidate.error === "string";
}

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body !== undefined && !(init.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, { ...init, headers });
  } catch {
    throw new ApiError(0, "Không kết nối được máy chủ");
  }

  let text = "";
  try {
    text = await response.text();
  } catch {
    text = "";
  }

  let payload: unknown;
  if (text) {
    try {
      payload = JSON.parse(text) as unknown;
    } catch {
      payload = undefined;
    }
  }

  if (!response.ok) {
    const message = isErrorEnvelope(payload)
      ? payload.error
      : response.status === 502 || response.status === 503 || response.status === 504
        ? "Không kết nối được máy chủ"
        : `HTTP ${response.status}`;
    throw new ApiError(response.status, message, payload);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return payload as T;
}

export function apiRequestForm<T>(path: string, form: FormData): Promise<T> {
  return apiRequest<T>(path, { method: "POST", body: form });
}
