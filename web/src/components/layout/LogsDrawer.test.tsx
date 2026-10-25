import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderAt } from "../../test/render";

it("opens the log drawer, loads logs, and closes on Escape", async () => {
  renderAt("/");
  await userEvent.click(await screen.findByRole("button", { name: "Nhật ký" }));
  expect(
    await screen.findByText("SpecTest Web UI Server listening on :4173"),
  ).toBeInTheDocument();
  await userEvent.keyboard("{Escape}");
  expect(
    screen.queryByRole("dialog", { name: "Nhật ký hệ thống" }),
  ).not.toBeInTheDocument();
});
