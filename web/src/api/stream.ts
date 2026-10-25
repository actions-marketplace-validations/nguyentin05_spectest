import { ApiError } from "./errors";
import type { RunStreamEvent } from "./types";

export type RunStreamHandlers = {
  onEvent(event: RunStreamEvent): void;
  onError?(error: Error): void;
  onClose?(): void;
};

function emitFrame(frame: string, onEvent: (event: RunStreamEvent) => void): void {
  const dataLines: string[] = [];
  for (const line of frame.split("\n")) {
    if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).replace(/^ /, ""));
    }
  }
  if (dataLines.length === 0) return;

  const raw = dataLines.join("\n");
  if (!raw.trim()) return;

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw) as unknown;
  } catch {
    return;
  }

  if (typeof parsed === "object" && parsed !== null && "type" in parsed) {
    onEvent(parsed as RunStreamEvent);
  }
}

export async function parseSSEStream(
  body: ReadableStream<Uint8Array>,
  onEvent: (event: RunStreamEvent) => void,
): Promise<void> {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer = (buffer + decoder.decode(value, { stream: true })).replace(/\r\n/g, "\n");
      let boundary = buffer.indexOf("\n\n");
      while (boundary !== -1) {
        const frame = buffer.slice(0, boundary);
        buffer = buffer.slice(boundary + 2);
        emitFrame(frame, onEvent);
        boundary = buffer.indexOf("\n\n");
      }
    }

    buffer = (buffer + decoder.decode()).replace(/\r\n/g, "\n");
    if (buffer.trim()) {
      emitFrame(buffer, onEvent);
    }
  } finally {
    reader.releaseLock();
  }
}

export function subscribeRun(handlers: RunStreamHandlers): () => void {
  const controller = new AbortController();

  const run = async () => {
    try {
      const response = await fetch("/api/run/stream", {
        signal: controller.signal,
        headers: { Accept: "text/event-stream" },
      });
      if (!response.ok) {
        throw new ApiError(response.status, `HTTP ${response.status}`);
      }
      if (!response.body) {
        throw new ApiError(0, "Không nhận được luồng sự kiện");
      }
      await parseSSEStream(response.body, handlers.onEvent);
    } catch (error) {
      if (controller.signal.aborted) return;
      handlers.onError?.(error instanceof Error ? error : new Error(String(error)));
    } finally {
      handlers.onClose?.();
    }
  };

  void run();
  return () => controller.abort();
}
