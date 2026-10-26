import { http, HttpResponse } from "msw";
import { mockState } from "../store";

export const historyHandlers = [
  http.get("/api/specs/history", ({ request }) => {
    const dir = new URL(request.url).searchParams.get("dir");
    if (dir === "missing") {
      return HttpResponse.json(
        { success: false, error: `History file not found: ${dir}` },
        { status: 404 },
      );
    }
    return HttpResponse.json(mockState.history);
  }),
];
