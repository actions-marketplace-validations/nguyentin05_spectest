import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { JsonTree } from "./JsonTree";

it("collapses deep values and reveals them on expand", async () => {
  render(<JsonTree value={{ data: { items: [{ product_id: "prod_1" }] } }} />);
  expect(screen.getByText("data")).toBeInTheDocument();
  expect(screen.queryByText("prod_1")).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: /mở data/i }));
  await userEvent.click(screen.getByRole("button", { name: /mở items/i }));
  await userEvent.click(screen.getByRole("button", { name: /mở 0/i }));
  expect(screen.getByText("prod_1")).toBeInTheDocument();
});

it("truncates long strings and expands them", async () => {
  const long = "x".repeat(30);
  render(<JsonTree value={{ note: long }} truncateAt={10} />);
  expect(screen.getByText("xxxxxxxxxx…")).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "xem thêm" }));
  expect(screen.getByText(long)).toBeInTheDocument();
});
