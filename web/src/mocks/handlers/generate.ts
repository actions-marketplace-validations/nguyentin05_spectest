import { http, HttpResponse } from "msw";
import type { GenerateRequest } from "../../api/types";
import { workspaceDir } from "../fixtures";
import { appendLog, mockState, sleep } from "../store";

function badRequest(message: string) {
  return HttpResponse.json({ success: false, error: message }, { status: 400 });
}

export const generateHandlers = [
  http.post("/api/generate", async ({ request }) => {
    let body: GenerateRequest;
    try {
      body = (await request.json()) as GenerateRequest;
    } catch {
      return badRequest("Invalid JSON body");
    }

    if (!Array.isArray(body?.apis) || body.apis.length === 0) {
      return badRequest("At least one API spec is required");
    }

    await sleep(600);

    const suiteName = "tests/spectest.yaml";
    if (!mockState.workspace.some((item) => item.name === suiteName)) {
      mockState.workspace.push({
        name: suiteName,
        path: `${workspaceDir}/${suiteName}`,
        type: "suite",
        size: 8940,
      });
    }

    appendLog(`Generated suite: ${body.output_dir}/spectest.yaml`);
    return HttpResponse.json({
      success: true,
      suite_path: "./tests/spectest.yaml",
      log_path: "./tests/spectest.log.yaml",
      servers: body.apis.map((entry) => entry.server),
      apis_count: body.apis.length,
      scenarios_generated: 6,
      message: "Test suite generated and saved successfully",
    });
  }),
];
