import { http, HttpResponse } from "msw";
import { workspaceDir } from "../fixtures";
import { mockState } from "../store";

export const workspaceHandlers = [
  http.get("/api/specs/workspace", () =>
    HttpResponse.json({ current_dir: workspaceDir, items: mockState.workspace }),
  ),
];
