import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderAt } from "../test/render";

it("loads masked config and saves only edited fields", async () => {
  renderAt("/settings");
  const key = await screen.findByLabelText("API key");
  expect(key).toHaveAttribute("placeholder", "AIzaSy...4xK9");
  expect(screen.getByLabelText("Nhà cung cấp")).toHaveValue("gemini");
  expect(screen.getByLabelText("Model")).toHaveValue("gemini-1.5-flash");
  expect(screen.getByLabelText("Endpoint")).toHaveValue(
    "https://generativelanguage.googleapis.com",
  );
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

it("shows success after a successful ping without inventing latency", async () => {
  renderAt("/settings");
  await userEvent.click(await screen.findByRole("button", { name: "Kiểm tra kết nối" }));
  expect(await screen.findByText("Kết nối thành công")).toBeInTheDocument();
  expect(screen.queryByText(/undefined/)).not.toBeInTheDocument();
});

it("shows the backend message when the connection test fails", async () => {
  renderAt("/settings");
  await userEvent.type(await screen.findByLabelText("API key"), "invalid");
  await userEvent.click(screen.getByRole("button", { name: "Kiểm tra kết nối" }));
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "LLM connection check failed: invalid API key",
  );
  expect(screen.queryByText(/Kết nối thành công/)).not.toBeInTheDocument();
});

it("probes the target server and reports readiness", async () => {
  renderAt("/settings");
  await screen.findByLabelText("Nhà cung cấp");
  await userEvent.click(screen.getByRole("button", { name: "Kiểm tra máy chủ" }));
  expect(await screen.findByText("Sẵn sàng")).toBeInTheDocument();
  expect(screen.getByText("15 ms")).toBeInTheDocument();
});
