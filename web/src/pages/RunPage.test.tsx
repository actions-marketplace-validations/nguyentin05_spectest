import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { mockState } from "../mocks/store";
import { server } from "../mocks/server";
import { renderAt } from "../test/render";

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
});

it("shows the server message when a run cannot be cancelled", async () => {
  mockState.runDelayMs = 50;
  server.use(
    http.post("/api/run/cancel", () =>
      HttpResponse.json(
        { success: false, error: "No active test run to cancel" },
        { status: 409 },
      ),
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
    "No active test run to cancel",
  );
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
