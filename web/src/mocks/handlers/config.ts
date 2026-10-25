import { http, HttpResponse } from "msw";
import { mockState } from "../store";

export const configHandlers = [
  http.get("/api/config", () => HttpResponse.json(mockState.config)),
];
