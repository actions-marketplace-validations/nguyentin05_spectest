import { useCallback, useEffect, useRef, useState } from "react";
import { subscribeRun } from "../api/stream";
import type { RunStreamEvent } from "../api/types";

export type RunStreamState = {
  events: RunStreamEvent[];
  connected: boolean;
  error: Error | null;
  start: () => void;
  stop: () => void;
  clear: () => void;
};

export function useRunStream(): RunStreamState {
  const [events, setEvents] = useState<RunStreamEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const abortRef = useRef<(() => void) | null>(null);
  const seenRef = useRef(new Set<string>());

  const stop = useCallback(() => {
    abortRef.current?.();
    abortRef.current = null;
    setConnected(false);
  }, []);

  const clear = useCallback(() => {
    stop();
    seenRef.current = new Set();
    setEvents([]);
    setError(null);
  }, [stop]);

  const start = useCallback(() => {
    abortRef.current?.();
    setConnected(true);
    setError(null);

    abortRef.current = subscribeRun({
      onEvent(event) {
        const key = JSON.stringify(event);
        if (seenRef.current.has(key)) return;
        seenRef.current.add(key);
        setEvents((current) => [...current, event]);
      },
      onError(cause) {
        setError(cause);
        setConnected(false);
      },
      onClose() {
        setConnected(false);
      },
    });
  }, []);

  useEffect(
    () => () => {
      abortRef.current?.();
    },
    [],
  );

  return { events, connected, error, start, stop, clear };
}
