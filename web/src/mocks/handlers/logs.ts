import { http, HttpResponse } from "msw";
import { mockState } from "../store";

export const logsHandlers = [
  http.get("/api/logs", () => HttpResponse.json({ logs: mockState.logs })),
];
