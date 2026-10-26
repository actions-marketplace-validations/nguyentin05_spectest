import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderAt } from "../test/render";

it("renders scenarios visually and exposes steps on demand", async () => {
  renderAt("/suite");
  expect(
    await screen.findByRole("heading", { name: "Đặt hàng vượt tồn kho" }),
  ).toBeInTheDocument();
  expect(screen.getByText("Đã khóa")).toBeInTheDocument();
  await userEvent.click(
    screen.getByRole("button", { name: /Đặt hàng vượt tồn kho/ }),
  );
  expect(screen.getByText("POST {order}/api/orders")).toBeInTheDocument();
  expect(screen.getByText("422")).toBeInTheDocument();
  expect(screen.getByText("Bearer {{user_token}}")).toBeInTheDocument();
});

it("filters to locked scenarios", async () => {
  renderAt("/suite");
  await userEvent.click(
    await screen.findByRole("checkbox", { name: "Chỉ kịch bản bị khóa" }),
  );
  expect(screen.queryByText("Login sai password")).not.toBeInTheDocument();
});
