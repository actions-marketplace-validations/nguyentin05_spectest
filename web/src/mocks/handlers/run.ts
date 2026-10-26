import { http, HttpResponse } from "msw";
import type {
  RunCaseResult,
  RunReport,
  RunRequest,
  RunStartResponse,
  RunStreamEvent,
} from "../../api/types";
import { appendLog, mockState, sleep, type MockRun } from "../store";

type StepResultEvent = Extract<RunStreamEvent, { type: "step_result" }>;
type ScenarioCompleteEvent = Extract<RunStreamEvent, { type: "scenario_complete" }>;

type ScriptStep = {
  call: string;
  method: string;
  url: string;
  statusCode: number;
  expectedCode: number;
  latencyMs: number;
  savedVars?: Record<string, string>;
};

type ScriptScenario = {
  id: string;
  name: string;
  steps: ScriptStep[];
};

const SCENARIOS: ScriptScenario[] = [
  {
    id: "order-over-stock",
    name: "Đặt hàng vượt tồn kho",
    steps: [
      {
        call: "POST {auth}/api/register",
        method: "POST",
        url: "http://localhost:8001/api/register",
        statusCode: 201,
        expectedCode: 201,
        latencyMs: 32,
        savedVars: { user_id: "usr_8f2c1", email: "user_7k2p@test.com" },
      },
      {
        call: "POST {auth}/api/login",
        method: "POST",
        url: "http://localhost:8001/api/login",
        statusCode: 200,
        expectedCode: 200,
        latencyMs: 28,
        savedVars: { user_token: "eyJhbGciOiJIUzI1NiJ9.mock" },
      },
      {
        call: "POST {order}/api/orders",
        method: "POST",
        url: "http://localhost:8002/api/orders",
        statusCode: 422,
        expectedCode: 422,
        latencyMs: 45,
      },
    ],
  },
  {
    id: "login-wrong-password",
    name: "Login sai password",
    steps: [
      {
        call: "POST {auth}/api/register",
        method: "POST",
        url: "http://localhost:8001/api/register",
        statusCode: 201,
        expectedCode: 201,
        latencyMs: 14,
        savedVars: { user_id: "usr_3a7d2", email: "user_9d1x@test.com" },
      },
      {
        call: "POST {auth}/api/login",
        method: "POST",
        url: "http://localhost:8001/api/login",
        statusCode: 401,
        expectedCode: 401,
        latencyMs: 21,
      },
    ],
  },
];

function timestamp(): string {
  return new Date().toISOString();
}

function isActive(run: MockRun): boolean {
  return mockState.run === run && run.status === "running";
}

function emit(run: MockRun, event: RunStreamEvent): void {
  run.events.push(event);
  for (const subscriber of [...run.subscribers]) {
    subscriber(event);
  }
}

function subscribe(run: MockRun, listener: (event: RunStreamEvent) => void): () => void {
  run.subscribers.add(listener);
  return () => run.subscribers.delete(listener);
}

function buildReport(run: MockRun): RunReport {
  const stepEvents = run.events.filter(
    (event): event is StepResultEvent => event.type === "step_result",
  );
  const scenarioEvents = run.events.filter(
    (event): event is ScenarioCompleteEvent => event.type === "scenario_complete",
  );

  const results: RunCaseResult[] = stepEvents.map((event) => ({
    scenario_id: event.data.scenario_id,
    call: event.data.call,
    method: event.data.method,
    url: event.data.url,
    status: event.data.passed ? "PASS" : "FAIL",
    http_status: event.data.status_code,
    expected_http_status: event.data.expect,
    latency_ms: event.data.latency_ms,
    error: event.data.error,
    saved_vars: event.data.saved_vars,
  }));

  const failed = results.filter((result) => result.status === "FAIL").length;
  const status =
    run.status === "cancelled" ? "cancelled" : failed > 0 ? "failed" : "passed";

  return {
    id: run.id,
    target_url: run.targetUrl,
    suite_path: run.suitePath,
    status,
    total_scenarios: SCENARIOS.length,
    passed_scenarios: scenarioEvents.filter((event) => event.data.passed).length,
    failed_scenarios: scenarioEvents.filter((event) => !event.data.passed).length,
    skipped_scenarios: scenarioEvents.filter((event) => event.data.skipped).length,
    total_cases: results.length,
    passed: results.length - failed,
    failed,
    skipped: 0,
    start_time: new Date(run.startedAt).toISOString(),
    end_time: timestamp(),
    duration_ms: Date.now() - run.startedAt,
    results,
  };
}

async function drive(run: MockRun): Promise<void> {
  for (const scenario of SCENARIOS) {
    await sleep(mockState.runDelayMs);
    if (!isActive(run)) return;

    emit(run, {
      type: "scenario_start",
      timestamp: timestamp(),
      message: `Executing scenario: ${scenario.id}`,
      data: { scenario_id: scenario.id, scenario_name: scenario.name },
    });

    for (const step of scenario.steps) {
      await sleep(mockState.runDelayMs);
      if (!isActive(run)) return;

      const passed = step.statusCode === step.expectedCode;
      emit(run, {
        type: "step_result",
        timestamp: timestamp(),
        data: {
          scenario_id: scenario.id,
          call: step.call,
          method: step.method,
          url: step.url,
          status_code: step.statusCode,
          expect: step.expectedCode,
          latency_ms: step.latencyMs,
          passed,
          error: passed
            ? undefined
            : `Expected ${step.expectedCode}, received ${step.statusCode}`,
          saved_vars: step.savedVars,
        },
      });
    }

    await sleep(mockState.runDelayMs);
    if (!isActive(run)) return;

    emit(run, {
      type: "scenario_complete",
      timestamp: timestamp(),
      data: {
        scenario_id: scenario.id,
        passed: scenario.steps.every((step) => step.statusCode === step.expectedCode),
      },
    });
  }

  await sleep(mockState.runDelayMs);
  if (!isActive(run)) return;

  run.status = "completed";
  const report = buildReport(run);
  mockState.report = report;
  appendLog(
    `Test run completed: ${run.id} (${report.passed_scenarios}/${report.total_scenarios} scenarios passed)`,
  );
  emit(run, {
    type: "complete",
    timestamp: timestamp(),
    message: "All test scenarios completed",
    data: { report },
  });
  run.subscribers.clear();
}

function startRun(body: RunRequest): RunStartResponse {
  const targetUrl = body.target_url ?? Object.values(body.targets ?? {})[0] ?? "";
  const run: MockRun = {
    id: `run-${Math.random().toString(16).slice(2, 10)}`,
    status: "running",
    suitePath: body.suite_path,
    targetUrl,
    startedAt: Date.now(),
    events: [],
    subscribers: new Set(),
  };

  mockState.run = run;
  mockState.report = null;
  appendLog(`Test run started: ${run.id} -> ${targetUrl || "per-server targets"}`);

  emit(run, {
    type: "start",
    timestamp: timestamp(),
    message: "Beginning execution of suite",
    data: {
      total_scenarios: SCENARIOS.length,
      servers: mockState.plan.servers,
    },
  });

  void drive(run);

  return {
    run_id: run.id,
    status: "running",
    message: "Test execution started",
    target_url: targetUrl,
    suite_path: body.suite_path,
  };
}

function badRequest(message: string) {
  return HttpResponse.json({ error: true, message }, { status: 400 });
}

function runStream(run: MockRun, signal: AbortSignal): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  let closed = false;
  let ping: ReturnType<typeof setInterval> | undefined;
  let unsubscribe: (() => void) | undefined;

  return new ReadableStream<Uint8Array>({
    start(controller) {
      const close = () => {
        if (closed) return;
        closed = true;
        if (ping) clearInterval(ping);
        unsubscribe?.();
        try {
          controller.close();
        } catch {
          // The stream was already closed by the client.
        }
      };

      const write = (event: RunStreamEvent) => {
        if (closed) return;
        controller.enqueue(encoder.encode(`data: ${JSON.stringify(event)}\n\n`));
        if (event.type === "complete" || event.type === "cancelled") close();
      };

      for (const event of [...run.events]) write(event);
      if (closed) return;

      if (run.status !== "running") {
        close();
        return;
      }

      unsubscribe = subscribe(run, write);
      ping = setInterval(() => {
        if (!closed) controller.enqueue(encoder.encode(": ping\n\n"));
      }, 15000);
      signal.addEventListener("abort", close);
    },
    cancel() {
      closed = true;
      if (ping) clearInterval(ping);
      unsubscribe?.();
    },
  });
}

function sseResponse(body: BodyInit) {
  return new HttpResponse(body, {
    headers: {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      Connection: "keep-alive",
    },
  });
}

export const runHandlers = [
  http.post("/api/run", async ({ request }) => {
    let body: RunRequest;
    try {
      body = (await request.json()) as RunRequest;
    } catch {
      return badRequest("Invalid JSON body");
    }

    if (!body?.suite_path) {
      return badRequest("suite_path is required");
    }
    if (!body.target_url && !body.targets) {
      return badRequest("target_url or targets is required");
    }

    return HttpResponse.json(startRun(body), { status: 202 });
  }),

  http.get("/api/run/stream", ({ request }) => {
    const run = mockState.run;
    if (!run) {
      const encoder = new TextEncoder();
      return sseResponse(
        new ReadableStream({
          start(controller) {
            controller.enqueue(
              encoder.encode(
                `data: ${JSON.stringify({
                  type: "idle",
                  timestamp: timestamp(),
                  message: "No active test run is in progress",
                })}\n\n`,
              ),
            );
            controller.close();
          },
        }),
      );
    }

    return sseResponse(runStream(run, request.signal));
  }),

  http.post("/api/run/cancel", () => {
    const run = mockState.run;
    if (!run || run.status !== "running") {
      return HttpResponse.json({
        success: false,
        message: "no active test run is in progress",
      });
    }

    run.status = "cancelled";
    mockState.report = buildReport(run);
    appendLog(`Test run cancelled: ${run.id}`);
    emit(run, {
      type: "cancelled",
      timestamp: timestamp(),
      message: "test run cancelled successfully",
    });
    run.subscribers.clear();

    return HttpResponse.json({
      success: true,
      message: "test run cancelled successfully",
    });
  }),

  http.get("/api/run/report", () => {
    if (!mockState.report) {
      return HttpResponse.json(
        { error: true, message: "no test runs recorded yet" },
        { status: 404 },
      );
    }
    return HttpResponse.json(mockState.report);
  }),
];
