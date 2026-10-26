import { http, HttpResponse } from "msw";
import { mockState } from "../store";

function rawDocument(path: string, content: string) {
  const lines = content.split("\n");
  const title =
    lines
      .map((line) => line.trim())
      .find((line) => line.startsWith("# "))
      ?.slice(2)
      .trim() ?? "SpecTest Markdown Document";
  const sections = lines
    .map((line) => line.trim())
    .filter((line) => line.startsWith("## ") || line.startsWith("### "))
    .map((line) => line.replace(/^#+\s*/, ""));
  const tableCount = lines.filter((line) => {
    const trimmed = line.trim();
    return trimmed.startsWith("|") && trimmed.endsWith("|") && trimmed.includes("---");
  }).length;

  return {
    path,
    title,
    sections,
    table_count: tableCount,
    line_count: lines.length,
    raw_content: content,
  };
}

export const markdownHandlers = [
  http.post("/api/markdown/read", async ({ request }) => {
    let body: { path?: string };
    try {
      body = (await request.json()) as { path?: string };
    } catch {
      return HttpResponse.json(
        { error: true, message: "invalid request body" },
        { status: 400 },
      );
    }

    const path = body.path?.trim();
    if (!path) {
      return HttpResponse.json(
        {
          error: true,
          message: "no markdown path or content provided, and test_matrix.md was not found",
        },
        { status: 400 },
      );
    }

    const content = mockState.markdown[path];
    if (content === undefined) {
      return HttpResponse.json(
        { error: true, message: `cannot read markdown file "${path}"` },
        { status: 400 },
      );
    }

    return HttpResponse.json(rawDocument(path, content));
  }),
];
