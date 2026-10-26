import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { createHistoryFixture } from "../mocks/fixtures";
import { server } from "../mocks/server";
import { renderAt } from "../test/render";

it("renders an update with its trigger and before/after change", async () => {
  renderAt("/history");
  expect(
    await screen.findByText("API added: POST {order}/api/orders/{id}/cancel"),
  ).toBeInTheDocument();
  expect(screen.getByText("POST {auth}/api/register")).toBeInTheDocument();
  expect(screen.getByText("0912345678")).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "Lần cập nhật #1" })).toBeInTheDocument();
});

it("shows an empty state when the log file is missing", async () => {
  server.use(
    http.get("/api/specs/history", () =>
      HttpResponse.json({ success: false, error: "File not found" }, { status: 404 }),
    ),
  );
  renderAt("/history");
  expect(await screen.findByText("Chưa có lịch sử thay đổi")).toBeInTheDocument();
});

it("loads history from the path query parameter", async () => {
  let requested: URL | undefined;
  server.use(
    http.get("/api/specs/history", ({ request }) => {
      requested = new URL(request.url);
      return HttpResponse.json(createHistoryFixture());
    }),
  );
  renderAt("/history?path=tests/spectest.log.yaml");
  expect(
    await screen.findByText("API added: POST {order}/api/orders/{id}/cancel"),
  ).toBeInTheDocument();
  expect(requested?.searchParams.get("path")).toBe("tests/spectest.log.yaml");
  expect(requested?.searchParams.has("dir")).toBe(false);
});

it("lets the backend discover the log when no path is provided", async () => {
  let requested: URL | undefined;
  server.use(
    http.get("/api/specs/history", ({ request }) => {
      requested = new URL(request.url);
      return HttpResponse.json(createHistoryFixture());
    }),
  );
  renderAt("/history");
  expect(
    await screen.findByText("API added: POST {order}/api/orders/{id}/cancel"),
  ).toBeInTheDocument();
  expect(requested?.searchParams.has("path")).toBe(false);
  expect(requested?.searchParams.has("dir")).toBe(false);
});
