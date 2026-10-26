import { http, HttpResponse } from "msw";
import { mockState } from "../store";

export const historyHandlers = [
  http.get("/api/specs/history", ({ request }) => {
    const path = new URL(request.url).searchParams.get("path");
    if (path === "missing") {
      return HttpResponse.json(
        { error: true, message: "no spectest.log.yaml file found in workspace" },
        { status: 404 },
      );
    }
    return HttpResponse.json(mockState.history);
  }),
];
