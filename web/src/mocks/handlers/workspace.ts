import { http, HttpResponse } from "msw";
import type { UploadTextRequest, WorkspaceItem, WorkspaceItemType } from "../../api/types";
import { workspaceDir } from "../fixtures";
import { appendLog, mockState } from "../store";

const UPLOAD_DIR = ".spectest_ui_uploads";

function inferType(fileName: string): WorkspaceItemType {
  if (fileName.endsWith("spectest.log.yaml")) return "history";
  if (fileName.endsWith("spectest.yaml")) return "suite";
  if (/\.(ya?ml|json)$/i.test(fileName)) return "openapi";
  if (/\.md$/i.test(fileName)) return "prd";
  return "other";
}

function addUpload(fileName: string, size: number): WorkspaceItem {
  const safeName = fileName.split(/[\\/]/).pop() ?? fileName;
  const item: WorkspaceItem = {
    name: safeName,
    path: `${workspaceDir}/${UPLOAD_DIR}/${safeName}`,
    type: inferType(safeName),
    size,
  };
  mockState.workspace.push(item);
  appendLog(`Uploaded file: ${safeName}`);
  return item;
}

function uploadResponse(item: WorkspaceItem) {
  return HttpResponse.json({
    success: true,
    file_name: item.name,
    path: item.path,
    type: item.type,
    size: item.size,
  });
}

function badRequest(message: string) {
  return HttpResponse.json({ success: false, error: message }, { status: 400 });
}

export const workspaceHandlers = [
  http.get("/api/specs/workspace", () =>
    HttpResponse.json({ current_dir: workspaceDir, items: mockState.workspace }),
  ),

  http.post("/api/specs/upload", async ({ request }) => {
    const contentType = request.headers.get("content-type") ?? "";

    if (contentType.includes("multipart/form-data")) {
      let form: FormData;
      try {
        form = await request.formData();
      } catch {
        return badRequest("Invalid multipart body");
      }

      const file = form.get("file");
      if (!file || typeof file === "string") {
        return badRequest("Missing file field");
      }

      return uploadResponse(addUpload(file.name, file.size));
    }

    let body: UploadTextRequest;
    try {
      body = (await request.json()) as UploadTextRequest;
    } catch {
      return badRequest("Invalid JSON body");
    }

    if (!body?.file_name || typeof body.content !== "string") {
      return badRequest("file_name and content are required");
    }

    const size = new TextEncoder().encode(body.content).length;
    return uploadResponse(addUpload(body.file_name, size));
  }),
];
