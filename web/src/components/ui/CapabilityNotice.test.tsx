import { render, screen } from "@testing-library/react";
import { ApiError } from "../../api/errors";
import { CapabilityNotice } from "./CapabilityNotice";

it("explains a missing backend feature instead of showing a raw error", () => {
  render(<CapabilityNotice feature="chạy kiểm thử" error={new ApiError(404, "not implemented")} />);
  expect(screen.getByRole("status").textContent).toBe("Backend chưa hỗ trợ chạy kiểm thử.");
  expect(screen.queryByText("not implemented")).not.toBeInTheDocument();
});

it("does not mention mock data when the feature is unavailable", () => {
  render(<CapabilityNotice feature="chạy kiểm thử" />);
  const notice = screen.getByRole("status");
  expect(notice.textContent).toBe("Backend chưa hỗ trợ chạy kiểm thử.");
  expect(notice.textContent).not.toMatch(/chế độ mẫu|dữ liệu mẫu/i);
});
