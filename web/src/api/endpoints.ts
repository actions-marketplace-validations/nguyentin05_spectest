import { withQuery } from "../lib/url";
import { apiRequest, apiRequestForm } from "./client";
import { toLLMConfig, type RawConfig } from "./contracts";
import type {
  ActionResponse,
  ConfigTestRequest,
  ConfigTestResponse,
  ConfigUpdateRequest,
  GeneratePendingResponse,
  GenerateRequest,
  GenerateResponse,
  HealthResponse,
  HealthcheckRequest,
  HealthcheckResponse,
  HistoryResponse,
  LogsResponse,
  MarkdownResponse,
  PlanResponse,
  RunReport,
  RunRequest,
  RunStartResponse,
  UploadResponse,
  UploadTextRequest,
  WorkspaceResponse,
} from "./types";

export const api = {
  health: () => apiRequest<HealthResponse>("/health"),
  logs: () => apiRequest<LogsResponse>("/logs"),
  getConfig: () => apiRequest<RawConfig>("/config").then(toLLMConfig),
  saveConfig: (body: ConfigUpdateRequest) =>
    apiRequest<ActionResponse>("/config", { method: "POST", body: JSON.stringify(body) }),
  testConfig: (body: ConfigTestRequest) =>
    apiRequest<ConfigTestResponse>("/config/test", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  healthcheck: (body: HealthcheckRequest) =>
    apiRequest<HealthcheckResponse>("/server/healthcheck", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  workspace: () => apiRequest<WorkspaceResponse>("/specs/workspace"),
  uploadFile: (file: File) => {
    const form = new FormData();
    form.append("file", file);
    return apiRequestForm<UploadResponse>("/specs/upload", form);
  },
  uploadText: (body: UploadTextRequest) =>
    apiRequest<UploadResponse>("/specs/upload", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  plan: (path?: string) => apiRequest<PlanResponse>(withQuery("/specs/plan", { path })),
  history: (dir?: string) => apiRequest<HistoryResponse>(withQuery("/specs/history", { dir })),
  generate: (body: GenerateRequest) =>
    apiRequest<GenerateResponse | GeneratePendingResponse>("/generate", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  run: (body: RunRequest) =>
    apiRequest<RunStartResponse>("/run", { method: "POST", body: JSON.stringify(body) }),
  cancelRun: () => apiRequest<ActionResponse>("/run/cancel", { method: "POST" }),
  report: () => apiRequest<RunReport>("/run/report"),
  markdown: (path: string) => apiRequest<MarkdownResponse>(withQuery("/markdown/read", { path })),
};
