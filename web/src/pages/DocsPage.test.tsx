import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderAt } from "../test/render";

it("renders a markdown document with a table", async () => {
  renderAt("/docs?path=docs/requirements.md");
  expect(
    await screen.findByRole("heading", { name: "Product Requirements Document" }),
  ).toBeInTheDocument();
  expect(screen.getByRole("table")).toBeInTheDocument();
});

it("shows a friendly 404", async () => {
  renderAt("/docs?path=docs/missing.md");
  expect(await screen.findByText("Không tìm thấy tài liệu")).toBeInTheDocument();
  expect(screen.getByText("docs/missing.md")).toBeInTheDocument();
});

it("picks a document and toggles the raw source", async () => {
  renderAt("/docs");
  await userEvent.click(
    await screen.findByRole("link", { name: "docs/requirements.md" }),
  );

  await userEvent.click(await screen.findByRole("button", { name: "Xem nội dung gốc" }));
  const raw = screen.getByRole("textbox", { name: "Nội dung Markdown gốc" });
  expect((raw as HTMLTextAreaElement).value).toContain("# Product Requirements Document");
});
