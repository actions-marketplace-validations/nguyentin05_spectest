import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App } from "../../App";

it("renders the navigation and marks the current page without a data mode chip", () => {
  render(
    <MemoryRouter initialEntries={["/suite"]}>
      <App />
    </MemoryRouter>,
  );
  expect(screen.getByRole("heading", { name: "Kịch bản" })).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Kịch bản" })).toHaveAttribute("aria-current", "page");
  expect(screen.queryByText("Chế độ mẫu")).not.toBeInTheDocument();
  expect(screen.queryByText("Backend thật")).not.toBeInTheDocument();
});

it.each([
  ["/", "Tổng quan"],
  ["/suite", "Kịch bản"],
  ["/generate", "Sinh kịch bản"],
  ["/run", "Chạy thử"],
  ["/history", "Lịch sử"],
  ["/workspace", "Tệp"],
  ["/settings", "Cài đặt"],
])("renders the %s route with its title", (path, title) => {
  render(
    <MemoryRouter initialEntries={[path]}>
      <App />
    </MemoryRouter>,
  );
  expect(screen.getByRole("heading", { name: title })).toBeInTheDocument();
});
