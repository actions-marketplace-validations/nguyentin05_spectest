import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderAt } from "../test/render";

it("loads masked config and saves only edited fields", async () => {
  renderAt("/settings");
  const key = await screen.findByLabelText("API key");
  expect(key).toHaveAttribute("placeholder", "AIzaSy...4xK9");
  await userEvent.clear(screen.getByLabelText("Model"));
  await userEvent.type(screen.getByLabelText("Model"), "gemini-2.0-flash");
  await userEvent.click(screen.getByRole("button", { name: "Lưu cấu hình" }));
  expect(await screen.findByText("Đã lưu cấu hình")).toBeInTheDocument();
});

it("blocks cloud providers without a key and explains why", async () => {
  renderAt("/settings");
  await userEvent.selectOptions(await screen.findByLabelText("Nhà cung cấp"), "openai");
  await userEvent.click(screen.getByRole("button", { name: "Kiểm tra kết nối" }));
  expect(await screen.findByText("Cần API key cho nhà cung cấp này")).toBeInTheDocument();
});

it("shows latency after a successful ping", async () => {
  renderAt("/settings");
  await userEvent.click(await screen.findByRole("button", { name: "Kiểm tra kết nối" }));
  expect(await screen.findByText("Kết nối thành công · 482 ms")).toBeInTheDocument();
});

it("probes the target server and reports readiness", async () => {
  renderAt("/settings");
  await screen.findByLabelText("Nhà cung cấp");
  await userEvent.click(screen.getByRole("button", { name: "Kiểm tra máy chủ" }));
  expect(await screen.findByText("Sẵn sàng")).toBeInTheDocument();
  expect(screen.getByText("15 ms")).toBeInTheDocument();
});
