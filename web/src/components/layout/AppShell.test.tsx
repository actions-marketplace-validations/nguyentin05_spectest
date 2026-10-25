import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App } from "../../App";

it("renders the navigation, marks the current page, and shows the data mode", () => {
  render(
    <MemoryRouter initialEntries={["/suite"]}>
      <App />
    </MemoryRouter>,
  );
  expect(screen.getByRole("heading", { name: "Kịch bản" })).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Kịch bản" })).toHaveAttribute("aria-current", "page");
  expect(screen.getByText("Chế độ mẫu")).toBeInTheDocument();
});
