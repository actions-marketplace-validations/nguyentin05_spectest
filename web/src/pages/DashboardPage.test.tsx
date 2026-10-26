import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import type { RunReport } from "../api/types";
import { server } from "../mocks/server";
import { renderAt } from "../test/render";

const lastReport: RunReport = {
  id: "run-last",
  target_url: "http://localhost:8080",
  suite_path: "tests/spectest.yaml",
  status: "passed",
  total_scenarios: 1,
  passed_scenarios: 1,
  failed_scenarios: 0,
  skipped_scenarios: 0,
  total_cases: 1,
  passed: 1,
  failed: 0,
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
      status: "PASS",
      http_status: 422,
      expected_http_status: 422,
      latency_ms: 12,
    },
  ],
};

it("shows plan stats and the last run summary", async () => {
  renderAt("/");
  expect(await screen.findByTestId("stat-scenarios")).toHaveTextContent("2");
  expect(screen.getByTestId("stat-setups")).toHaveTextContent("3");
  expect(screen.getByText("Đặt hàng vượt tồn kho")).toBeInTheDocument();
});

it("renders the latest run from the real report shape", async () => {
  server.use(http.get("/api/run/report", () => HttpResponse.json(lastReport)));
  renderAt("/");
  expect(await screen.findByText("Lần chạy gần nhất")).toBeInTheDocument();
  expect(await screen.findByText("POST {order}/api/orders")).toBeInTheDocument();
});

it("invites the user to generate a suite when none exists", async () => {
  server.use(
    http.get("/api/specs/plan", () =>
      HttpResponse.json({ success: false, error: "File not found" }, { status: 404 }),
    ),
  );
  renderAt("/");
  expect(await screen.findByText("Chưa có spectest.yaml")).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Sinh kịch bản đầu tiên" })).toHaveAttribute(
    "href",
    "/generate",
  );
});
