import { render, screen } from "@testing-library/react";
import { Field } from "./Field";

it("connects the label, hint, and error to the control", () => {
  render(
    <Field id="url" label="Địa chỉ" hint="Bắt đầu bằng http" error="URL không hợp lệ">
      <input />
    </Field>,
  );

  const input = screen.getByLabelText("Địa chỉ");
  expect(input).toHaveAttribute("aria-describedby", "url-hint url-error");
  expect(input).toHaveAttribute("aria-invalid", "true");
  expect(screen.getByText("URL không hợp lệ")).toBeInTheDocument();
});
