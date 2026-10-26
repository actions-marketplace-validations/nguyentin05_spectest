import { http, HttpResponse } from "msw";
import { mockState } from "../store";

export const suiteHandlers = [
  http.get("/api/specs/plan", ({ request }) => {
    const path = new URL(request.url).searchParams.get("path");
    if (path === "missing.yaml") {
      return HttpResponse.json(
        { success: false, error: `File not found: ${path}` },
        { status: 404 },
      );
    }
    return HttpResponse.json(mockState.plan);
  }),
];
