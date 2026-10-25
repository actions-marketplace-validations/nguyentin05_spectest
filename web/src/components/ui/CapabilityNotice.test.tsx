import { render, screen } from "@testing-library/react";
import { ApiError } from "../../api/errors";
import { CapabilityNotice } from "./CapabilityNotice";

it("explains a missing backend feature instead of showing a raw error", () => {
  render(<CapabilityNotice feature="chạy kiểm thử" error={new ApiError(404, "not implemented")} />);
  expect(screen.getByRole("status")).toHaveTextContent("Backend chưa hỗ trợ chạy kiểm thử");
  expect(screen.queryByText("not implemented")).not.toBeInTheDocument();
});
