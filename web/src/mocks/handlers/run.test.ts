import type { RunStreamEvent } from "../../api/types";
import { api } from "../../api/endpoints";
import { subscribeRun } from "../../api/stream";
import { mockState } from "../store";

it("runs the full SSE lifecycle and produces a report", async () => {
  mockState.runDelayMs = 0;
  await api.run({ suite_path: "tests/spectest.yaml", target_url: "http://localhost:8080" });
  const events: RunStreamEvent[] = [];
  await new Promise<void>((resolve, reject) => {
    const stop = subscribeRun({
      onEvent: (event) => {
        events.push(event);
        if (event.type === "complete") {
          stop();
          resolve();
        }
      },
      onError: reject,
    });
  });
  expect(events.map((event) => event.type)).toEqual([
    "start",
    "scenario_start",
    "step_result",
    "step_result",
    "step_result",
    "scenario_complete",
    "scenario_start",
    "step_result",
    "step_result",
    "scenario_complete",
    "complete",
  ]);
  const report = await api.report();
  expect(report).toMatchObject({ status: "passed", passed_scenarios: 2, failed_scenarios: 0 });
  expect(report.results).toHaveLength(5);
  expect(report.results[0]).toMatchObject({
    scenario_id: "order-over-stock",
    status: "PASS",
    http_status: 201,
    expected_http_status: 201,
  });
});

it("replays events for a late subscriber", async () => {
  mockState.runDelayMs = 0;
  await api.run({ suite_path: "tests/spectest.yaml", target_url: "http://localhost:8080" });
  await vi.waitFor(() => expect(mockState.report?.status).toBe("passed"));

  const types: string[] = [];
  await new Promise<void>((resolve) => {
    const stop = subscribeRun({
      onEvent: (event) => {
        types.push(event.type);
        if (event.type === "complete") {
          stop();
          resolve();
        }
      },
    });
  });
  expect(types[0]).toBe("start");
  expect(types.at(-1)).toBe("complete");
  expect(types).toHaveLength(11);
});

it("cancels an active run and reports cancelled", async () => {
  mockState.runDelayMs = 50;
  await api.run({ suite_path: "tests/spectest.yaml", target_url: "http://localhost:8080" });
  await api.cancelRun();
  await vi.waitFor(async () => expect((await api.report()).status).toBe("cancelled"));
});
