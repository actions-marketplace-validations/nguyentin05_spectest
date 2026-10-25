import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Tabs } from "./Tabs";

it("moves between tabs with the arrow keys", async () => {
  render(
    <Tabs
      items={[
        { id: "one", label: "Một", content: <p>nội dung một</p> },
        { id: "two", label: "Hai", content: <p>nội dung hai</p> },
      ]}
    />,
  );

  const first = screen.getByRole("tab", { name: "Một" });
  first.focus();
  await userEvent.keyboard("{ArrowRight}");

  expect(screen.getByRole("tab", { name: "Hai" })).toHaveFocus();
  expect(screen.getByRole("tab", { name: "Hai" })).toHaveAttribute("aria-selected", "true");
  expect(screen.getByText("nội dung hai")).toBeInTheDocument();
});
