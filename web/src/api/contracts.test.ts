import { toWorkspaceResponse } from "./contracts";
import type { WorkspaceResponse } from "./types";

it("marks the evolution log as history when the backend calls it other", () => {
  const raw: WorkspaceResponse = {
    current_dir: "/repo",
    items: [
      {
        name: "tests/spectest.log.yaml",
        path: "/repo/tests/spectest.log.yaml",
        type: "other",
        size: 10,
      },
      {
        name: "tests\\spectest.log.yml",
        path: "C:\\repo\\tests\\spectest.log.yml",
        type: "other",
        size: 20,
      },
      {
        name: "specs/auth-api.yaml",
        path: "/repo/specs/auth-api.yaml",
        type: "other",
        size: 30,
      },
    ],
  };

  expect(toWorkspaceResponse(raw).items.map((item) => item.type)).toEqual([
    "history",
    "history",
    "other",
  ]);
});
