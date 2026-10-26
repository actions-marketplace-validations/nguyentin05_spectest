import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import type { RunReport } from "../api/types";
import { mockState } from "../mocks/store";
import { server } from "../mocks/server";
import { renderAt } from "../test/render";

function streamEvents(events: unknown[]) {
  const encoder = new TextEncoder();
  return new HttpResponse(
    new ReadableStream({
      start(controller) {
        for (const event of events) {
          controller.enqueue(encoder.encode(`data: ${JSON.stringify(event)}\n\n`));
        }
        controller.close();
      },
    }),
    { headers: { "Content-Type": "text/event-stream" } },
  );
}

const failedReport: RunReport = {
  id: "run-test",
  target_url: "http://localhost:8080",
  suite_path: "tests/spectest.yaml",
  status: "failed",
  total_scenarios: 1,
  passed_scenarios: 0,
  failed_scenarios: 1,
  skipped_scenarios: 0,
  total_cases: 1,
  passed: 0,
  failed: 1,
  skipped: 0,
  start_time: "2026-10-26T10:00:00Z",
  end_time: "2026-10-26T10:00:00.120Z",
  duration_ms: 120,
  results: [
    {
      scenario_id: "order-over-stock",
      call: "POST {order}/api/orders",
      method: "POST",
      url: "http://localhost:8080/api/orders",
      status: "FAIL",
      http_status: 500,
      expected_http_status: 422,
      latency_ms: 12,
      error: "Expected 422, received 500",
      saved_vars: { order_id: "ord-1" },
    },
  ],
};

it("streams a run and shows every step on the signal spine", async () => {
  mockState.runDelayMs = 0;
  renderAt("/run");
  await userEvent.type(
    await screen.findByLabelText("Địa chỉ máy chủ đích"),
    "http://localhost:8080",
  );
  await userEvent.click(screen.getByRole("button", { name: "Bắt đầu chạy" }));
  expect(await screen.findByText("Hoàn tất · 2/2 kịch bản đạt")).toBeInTheDocument();
  expect(screen.getAllByText("POST {order}/api/orders").length).toBeGreaterThan(0);
  expect(screen.getAllByText("422").length).toBeGreaterThan(0);
  expect(
    screen.queryByText("Payload đầy đủ nằm trong báo cáo chi tiết sau khi chạy xong."),
  ).not.toBeInTheDocument();
});

it("shows step errors and AI diagnosis from the real stream", async () => {
  server.use(
    http.get("/api/run/stream", () =>
      streamEvents([
        {
          type: "connected",
          timestamp: "2026-10-26T10:00:00Z",
          message: "Connected to test run",
          data: { run_id: "run-test", suite_path: "tests/spectest.yaml" },
        },
        {
          type: "start",
          timestamp: "2026-10-26T10:00:00Z",
          message: "Beginning execution",
          data: { total_scenarios: 1, servers: ["order"], targets: {} },
        },
        {
          type: "scenario_start",
          timestamp: "2026-10-26T10:00:00.010Z",
          message: "Executing scenario",
          data: { scenario_id: "order-over-stock", scenario_name: "Đặt hàng vượt tồn kho" },
        },
        {
          type: "step_result",
          timestamp: "2026-10-26T10:00:00.030Z",
          data: {
            scenario_id: "order-over-stock",
            call: "POST {order}/api/orders",
            method: "POST",
            url: "http://localhost:8080/api/orders",
            status_code: 500,
            expect: 422,
            latency_ms: 12,
            passed: false,
            error: "Expected 422, received 500",
            ai_diagnosis: "Thiếu quyền admin trong token.",
            saved_vars: { order_id: "ord-1" },
          },
        },
        {
          type: "scenario_complete",
          timestamp: "2026-10-26T10:00:00.100Z",
          data: { scenario_id: "order-over-stock", passed: false },
        },
        {
          type: "complete",
          timestamp: "2026-10-26T10:00:00.120Z",
          message: "Run finished",
          data: { report: failedReport },
        },
      ]),
    ),
    http.get("/api/run/report", () => HttpResponse.json(failedReport)),
  );

  renderAt("/run?run=run-test");

  expect(await screen.findByText(/Expected 422, received 500/)).toBeInTheDocument();
  expect(screen.getByText(/Thiếu quyền admin trong token\./)).toBeInTheDocument();
  expect(await screen.findByText("Hoàn tất · 0/1 kịch bản đạt")).toBeInTheDocument();
});

it("asks for a suite before running when none exists", async () => {
  server.use(
    http.get("/api/specs/plan", () =>
      HttpResponse.json(
        { error: true, message: "no spectest.yaml file found in workspace" },
        { status: 404 },
      ),
    ),
  );

  renderAt("/run");

  const title = await screen.findByRole("heading", { name: "Chưa có spectest.yaml" });
  expect(
    within(title.parentElement!).getByRole("link", { name: "Sinh kịch bản" }),
  ).toHaveAttribute("href", "/generate");
  expect(screen.queryByText("Backend chưa hỗ trợ cấu hình chạy.")).not.toBeInTheDocument();
});

it("explains when the backend cannot load the run configuration", async () => {
  server.use(
    http.get("/api/specs/plan", () =>
      HttpResponse.json({ error: true, message: "not implemented" }, { status: 501 }),
    ),
  );

  renderAt("/run");

  expect(
    await screen.findByText("Backend chưa hỗ trợ chạy kiểm thử."),
  ).toBeInTheDocument();
});

it("cancels a run and reports it", async () => {
  mockState.runDelayMs = 50;
  renderAt("/run");
  await userEvent.type(
    await screen.findByLabelText("Địa chỉ máy chủ đích"),
    "http://localhost:8080",
  );
  await userEvent.click(screen.getByRole("button", { name: "Bắt đầu chạy" }));
  await userEvent.click(await screen.findByRole("button", { name: "Hủy phiên chạy" }));
  await userEvent.click(await screen.findByRole("button", { name: "Xác nhận hủy" }));
  expect(await screen.findByText("Đã hủy phiên chạy")).toBeInTheDocument();
});

it("opens the detailed report after completion", async () => {
  mockState.runDelayMs = 0;
  renderAt("/run");
  await userEvent.type(
    await screen.findByLabelText("Địa chỉ máy chủ đích"),
    "http://localhost:8080",
  );
  await userEvent.click(screen.getByRole("button", { name: "Bắt đầu chạy" }));
  await screen.findByText("Hoàn tất · 2/2 kịch bản đạt");
  await userEvent.click(screen.getByRole("tab", { name: "Báo cáo chi tiết" }));
  expect(await screen.findByRole("table")).toBeInTheDocument();
  expect(screen.getAllByRole("row")).toHaveLength(6);
  expect(screen.queryByText("request_payload")).not.toBeInTheDocument();
  expect(screen.queryByText("response_body")).not.toBeInTheDocument();
});

it("shows the server message when a run cannot be cancelled", async () => {
  mockState.runDelayMs = 50;
  server.use(
    http.post("/api/run/cancel", () =>
      HttpResponse.json({
        success: false,
        message: "no active test run is in progress",
      }),
    ),
  );
  renderAt("/run");
  await userEvent.type(
    await screen.findByLabelText("Địa chỉ máy chủ đích"),
    "http://localhost:8080",
  );
  await userEvent.click(screen.getByRole("button", { name: "Bắt đầu chạy" }));
  await userEvent.click(await screen.findByRole("button", { name: "Hủy phiên chạy" }));
  await userEvent.click(await screen.findByRole("button", { name: "Xác nhận hủy" }));
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "no active test run is in progress",
  );
  expect(
    screen.queryByText("Phiên chạy đã dừng theo yêu cầu."),
  ).not.toBeInTheDocument();
});

it("keeps received events when the stream drops and offers retry", async () => {
  const enc = new TextEncoder();
  server.use(
    http.get(
      "/api/run/stream",
      () =>
        new HttpResponse(
          new ReadableStream({
            start(controller) {
              controller.enqueue(
                enc.encode(
                  'data: {"type":"start","timestamp":"t","message":"go","data":{"suite_path":"tests/spectest.yaml","total_scenarios":2,"servers":["auth","order"]}}\n\n',
                ),
              );
              // Synchronous error() discards queued chunks, so let the reader consume first.
              setTimeout(() => controller.error(new Error("socket closed")), 10);
            },
          }),
          { headers: { "Content-Type": "text/event-stream" } },
        ),
    ),
  );
  renderAt("/run?run=run-a1b2c3d4");
  expect(await screen.findByText("Mất kết nối với phiên chạy")).toBeInTheDocument();
  expect(screen.getByText("Bắt đầu chạy suite")).toBeInTheDocument();
  const retry = screen.getByRole("button", { name: "Kết nối lại" });

  await userEvent.click(retry);
  expect(await screen.findByText("Mất kết nối với phiên chạy")).toBeInTheDocument();
  expect(screen.getAllByText("Bắt đầu chạy suite")).toHaveLength(1);
});
