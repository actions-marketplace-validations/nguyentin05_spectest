import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../mocks/server";
import { renderAt } from "../test/render";

it("shows plan stats and the last run summary", async () => {
  renderAt("/");
  expect(await screen.findByTestId("stat-scenarios")).toHaveTextContent("2");
  expect(screen.getByTestId("stat-setups")).toHaveTextContent("3");
  expect(screen.getByText("Đặt hàng vượt tồn kho")).toBeInTheDocument();
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
