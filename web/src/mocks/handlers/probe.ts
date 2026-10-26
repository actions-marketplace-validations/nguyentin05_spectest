import { http, HttpResponse } from "msw";
import type { HealthcheckRequest } from "../../api/types";

function badRequest(message: string) {
  return HttpResponse.json({ success: false, error: message }, { status: 400 });
}

export const probeHandlers = [
  http.post("/api/server/healthcheck", async ({ request }) => {
    let body: HealthcheckRequest;
    try {
      body = (await request.json()) as HealthcheckRequest;
    } catch {
      return badRequest("Invalid JSON body");
    }

    if (!body?.target_url) {
      return badRequest("target_url is required");
    }

    let target: URL;
    try {
      target = new URL(body.target_url);
    } catch {
      return badRequest("target_url must be a valid URL");
    }

    if (target.protocol !== "http:" && target.protocol !== "https:") {
      return badRequest("target_url must use http or https");
    }

    if (body.target_url.includes("unreachable")) {
      return HttpResponse.json({
        ready: false,
        target_url: body.target_url,
        latency_ms: 0,
        message: "Target server is unreachable",
      });
    }

    return HttpResponse.json({
      ready: true,
      target_url: body.target_url,
      latency_ms: 15,
      message: "Target server is healthy and responding",
    });
  }),
];
