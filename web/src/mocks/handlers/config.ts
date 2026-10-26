import { http, HttpResponse } from "msw";
import type { ConfigTestRequest, ConfigUpdateRequest } from "../../api/types";
import { maskMiddle } from "../../lib/format";
import { appendLog, mockState, sleep } from "../store";

function badRequest(message: string) {
  return HttpResponse.json({ error: true, message }, { status: 400 });
}

export const configHandlers = [
  http.get("/api/config", () => HttpResponse.json(mockState.config)),

  http.post("/api/config", async ({ request }) => {
    let body: ConfigUpdateRequest;
    try {
      body = (await request.json()) as ConfigUpdateRequest;
    } catch {
      return badRequest("Invalid JSON body");
    }

    if (!body?.provider || !body?.endpoint || !body?.model) {
      return badRequest("provider, endpoint and model are required");
    }

    mockState.config.default_provider = body.provider;
    mockState.config.endpoint_url = body.endpoint;
    mockState.config.active_model = body.model;

    if (body.api_key) {
      mockState.config.has_api_key = true;
      mockState.config.api_key_masked = maskMiddle(body.api_key);
    }

    appendLog(`Configuration updated: ${body.provider}/${body.model}`);
    return HttpResponse.json({
      success: true,
      message: "Configuration updated successfully",
    });
  }),

  http.post("/api/config/test", async ({ request }) => {
    let body: ConfigTestRequest;
    try {
      body = (await request.json()) as ConfigTestRequest;
    } catch {
      return badRequest("Invalid JSON body");
    }

    await sleep(600);

    if (!body?.provider || !body?.model) {
      return badRequest("provider and model are required");
    }

    if (body.api_key === "invalid") {
      return HttpResponse.json({
        success: false,
        message: "LLM connection check failed: invalid API key",
      });
    }

    appendLog(`LLM connection tested: ${body.provider}/${body.model}`);
    return HttpResponse.json({
      success: true,
      provider: body.provider,
      model: body.model,
      message: "Connection to LLM provider succeeded",
    });
  }),
];
