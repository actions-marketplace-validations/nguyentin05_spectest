import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderAt } from "../test/render";

it("lists workspace files, filters by type, and links viewers", async () => {
  renderAt("/workspace");
  expect(await screen.findByText("specs/auth-api.yaml")).toBeInTheDocument();
  expect(screen.getByText("docs/requirements.md")).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Đặc tả API" }));
  expect(screen.queryByText("docs/requirements.md")).not.toBeInTheDocument();
  expect(screen.getAllByText("specs/auth-api.yaml").length).toBeGreaterThan(0);
});

it("links a markdown file to the document reader", async () => {
  renderAt("/workspace");
  const link = await screen.findByRole("link", { name: /Mở docs\/requirements.md/ });
  expect(link).toHaveAttribute("href", "/docs?path=docs%2Frequirements.md");
});
