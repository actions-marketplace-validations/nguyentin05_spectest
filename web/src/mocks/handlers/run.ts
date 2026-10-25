import { http, HttpResponse } from "msw";
import type {
  RunReport,
  RunReportRecord,
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
  requestPayload: string;
  responseBody: string;
  savedVars?: Record<string, string>;
};

type ScriptScenario = {
  id: string;
  name: string;
  setupsToRun: string[];
  durationMs: number;
  steps: ScriptStep[];
};

const SCENARIOS: ScriptScenario[] = [
  {
    id: "order-over-stock",
    name: "Đặt hàng vượt tồn kho",
    setupsToRun: ["user", "logged_in_user"],
    durationMs: 115,
    steps: [
      {
        call: "POST {auth}/api/register",
        method: "POST",
        url: "http://localhost:8001/api/register",
        statusCode: 201,
        expectedCode: 201,
        latencyMs: 32,
        requestPayload: '{"email":"user_7k2p@test.com","password":"P@ssw0rd123"}',
        responseBody: '{"id":"usr_8f2c1","email":"user_7k2p@test.com"}',
        savedVars: { user_id: "usr_8f2c1", email: "user_7k2p@test.com" },
      },
      {
        call: "POST {auth}/api/login",
        method: "POST",
        url: "http://localhost:8001/api/login",
        statusCode: 200,
        expectedCode: 200,
        latencyMs: 28,
        requestPayload: '{"email":"user_7k2p@test.com","password":"P@ssw0rd123"}',
        responseBody: '{"access_token":"eyJhbGciOiJIUzI1NiJ9.mock"}',
        savedVars: { user_token: "eyJhbGciOiJIUzI1NiJ9.mock" },
      },
      {
        call: "POST {order}/api/orders",
        method: "POST",
        url: "http://localhost:8002/api/orders",
        statusCode: 422,
        expectedCode: 422,
        latencyMs: 45,
        requestPayload: '{"items":[{"product_id":"prod_1","quantity":999}]}',
        responseBody: '{"error":"insufficient stock"}',
      },
    ],
  },
  {
    id: "login-wrong-password",
    name: "Login sai password",
    setupsToRun: ["user"],
    durationMs: 100,
    steps: [
      {
        call: "POST {auth}/api/register",
        method: "POST",
        url: "http://localhost:8001/api/register",
        statusCode: 201,
        expectedCode: 201,
        latencyMs: 14,
        requestPayload: '{"email":"user_9d1x@test.com","password":"P@ssw0rd123"}',
        responseBody: '{"id":"usr_3a7d2","email":"user_9d1x@test.com"}',
        savedVars: { user_id: "usr_3a7d2", email: "user_9d1x@test.com" },
      },
      {
        call: "POST {auth}/api/login",
        method: "POST",
        url: "http://localhost:8001/api/login",
        statusCode: 401,
        expectedCode: 401,
        latencyMs: 21,
        requestPayload: '{"email":"user_9d1x@test.com","password":"sai_mat_khau"}',
        responseBody: '{"error":"invalid credentials"}',
      },
    ],
  },
];

const FLAT_STEPS = SCENARIOS.flatMap((scenario) => scenario.steps);
const SCENARIO_NAMES = new Map(SCENARIOS.map((scenario) => [scenario.id, scenario.name]));

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

  const stepIndexes = new Map<string, number>();
  const records: RunReportRecord[] = [];

  for (const event of stepEvents) {
    const step = FLAT_STEPS[records.length];
    if (!step) break;

    const stepIndex = (stepIndexes.get(event.data.scenario_id) ?? 0) + 1;
    stepIndexes.set(event.data.scenario_id, stepIndex);

    records.push({
      scenario_id: event.data.scenario_id,
      scenario_name: SCENARIO_NAMES.get(event.data.scenario_id) ?? event.data.scenario_id,
      step_index: stepIndex,
      call: step.call,
      method: step.method,
      url: step.url,
      passed: event.data.passed,
      status_code: event.data.status_code,
      expected_code: event.data.expected_code,
      latency_ms: event.data.latency_ms,
      fail_reason: event.data.passed
        ? ""
        : `Expected ${event.data.expected_code}, received ${event.data.status_code}`,
      ai_diagnosis: "",
      request_payload: step.requestPayload,
      response_body: step.responseBody,
    });
  }

  return {
    suite_path: run.suitePath,
    status: run.status === "completed" ? "passed" : "cancelled",
    start_time: new Date(run.startedAt).toISOString(),
    end_time: timestamp(),
    duration_ms: run.status === "completed" ? 254 : Date.now() - run.startedAt,
    total_scenarios: SCENARIOS.length,
    passed_scenarios: scenarioEvents.filter((event) => event.data.passed).length,
    failed_scenarios: scenarioEvents.filter((event) => !event.data.passed).length,
    skipped_scenarios: 0,
    records,
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
      data: { scenario_id: scenario.id, scenario_name: scenario.name, setups_to_run: scenario.setupsToRun },
    });

    for (const step of scenario.steps) {
      await sleep(mockState.runDelayMs);
      if (!isActive(run)) return;

      emit(run, {
        type: "step_result",
        timestamp: timestamp(),
        data: {
          scenario_id: scenario.id,
          call: step.call,
          method: step.method,
          url: step.url,
          status_code: step.statusCode,
          expected_code: step.expectedCode,
          latency_ms: step.latencyMs,
          passed: step.statusCode === step.expectedCode,
          saved_vars: step.savedVars,
        },
      });
    }

    await sleep(mockState.runDelayMs);
    if (!isActive(run)) return;

    emit(run, {
      type: "scenario_complete",
      timestamp: timestamp(),
      data: { scenario_id: scenario.id, passed: true, duration_ms: scenario.durationMs },
    });
  }

  await sleep(mockState.runDelayMs);
  if (!isActive(run)) return;

  const totalChecks = FLAT_STEPS.length;
  run.status = "completed";
  mockState.report = buildReport(run);
  appendLog(`Test run completed: ${run.id} (${SCENARIOS.length}/${SCENARIOS.length} scenarios passed)`);
  emit(run, {
    type: "complete",
    timestamp: timestamp(),
    message: "All test scenarios completed",
    data: {
      total_scenarios: SCENARIOS.length,
      passed_scenarios: SCENARIOS.length,
      failed_scenarios: 0,
      skipped_scenarios: 0,
      total_checks: totalChecks,
      duration_ms: 254,
    },
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
      suite_path: body.suite_path,
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
  return HttpResponse.json({ success: false, error: message }, { status: 400 });
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
      return HttpResponse.json(
        { success: false, error: "No test run available" },
        { status: 404 },
      );
    }

    return new HttpResponse(runStream(run, request.signal), {
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        Connection: "keep-alive",
      },
    });
  }),

  http.post("/api/run/cancel", () => {
    const run = mockState.run;
    if (!run || run.status !== "running") {
      return HttpResponse.json(
        { success: false, error: "No active test run to cancel" },
        { status: 409 },
      );
    }

    run.status = "cancelled";
    mockState.report = buildReport(run);
    appendLog(`Test run cancelled: ${run.id}`);
    emit(run, { type: "cancelled", timestamp: timestamp(), message: "Test run cancelled" });
    run.subscribers.clear();

    return HttpResponse.json({ success: true, message: "Test run cancelled successfully" });
  }),

  http.get("/api/run/report", () => {
    if (!mockState.report) {
      return HttpResponse.json(
        { success: false, error: "No test report available" },
        { status: 404 },
      );
    }
    return HttpResponse.json(mockState.report);
  }),
];
