import { render, screen } from "@testing-library/react";
import { DiffView } from "./DiffView";

it("highlights changed keys and renders both sides", () => {
  render(
    <DiffView
      before={{ vars: { email: "user@test.com" } }}
      after={{ vars: { email: "user@test.com", phone: "0912345678" } }}
    />,
  );

  expect(screen.getByText("vars.phone")).toHaveClass("is-changed");
  expect(screen.getByText("0912345678")).toBeInTheDocument();
});
