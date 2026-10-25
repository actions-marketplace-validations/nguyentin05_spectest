import { http, HttpResponse } from "msw";
import { mockState } from "../store";

export const markdownHandlers = [
  http.get("/api/markdown/read", ({ request }) => {
    const path = new URL(request.url).searchParams.get("path");
    if (!path) {
      return HttpResponse.json(
        { success: false, error: "Missing path parameter" },
        { status: 400 },
      );
    }
    const content = mockState.markdown[path];
    if (content === undefined) {
      return HttpResponse.json(
        { success: false, error: `File not found: ${path}` },
        { status: 404 },
      );
    }
    const fileName = path.split("/").pop() ?? path;
    return HttpResponse.json({ path, file_name: fileName, content });
  }),
];
