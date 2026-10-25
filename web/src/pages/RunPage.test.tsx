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
