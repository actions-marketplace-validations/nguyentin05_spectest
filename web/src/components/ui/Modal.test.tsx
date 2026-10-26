import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { Modal } from "./Modal";

function Harness() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        Mở hộp
      </button>
      <Modal open={open} title="Xác nhận" onClose={() => setOpen(false)}>
        <button type="button">Bên trong</button>
      </Modal>
    </>
  );
}

it("traps focus, closes on Escape, and restores the trigger", async () => {
  render(<Harness />);
  const trigger = screen.getByRole("button", { name: "Mở hộp" });
  trigger.focus();
  await userEvent.click(trigger);

  expect(screen.getByRole("dialog", { name: "Xác nhận" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Đóng" })).toHaveFocus();

  await userEvent.keyboard("{Escape}");
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  await waitFor(() => expect(trigger).toHaveFocus());
});
