import { api } from "../../api/endpoints";

it("persists config and rejects an invalid key", async () => {
  await api.saveConfig({
    provider: "openai",
    endpoint: "https://api.openai.com/v1",
    model: "gpt-4o-mini",
    api_key: "sk-x",
  });
  await expect(api.getConfig()).resolves.toMatchObject({
    provider: "openai",
    model: "gpt-4o-mini",
    has_api_key: true,
  });
  await expect(
    api.testConfig({
      provider: "gemini",
      endpoint: "https://generativelanguage.googleapis.com",
      model: "gemini-1.5-flash",
      api_key: "invalid",
    }),
  ).rejects.toMatchObject({ status: 400, message: expect.stringContaining("invalid API key") });
});

it("healthchecks a target", async () => {
  await expect(
    api.healthcheck({ target_url: "http://localhost:8080", timeout_seconds: 5 }),
  ).resolves.toMatchObject({ ready: true, latency_ms: 15 });
  await expect(api.healthcheck({ target_url: "http://unreachable" })).resolves.toMatchObject({
    ready: false,
  });
  await expect(api.healthcheck({ target_url: "not-a-url" })).rejects.toMatchObject({ status: 400 });
});

it("uploads raw text and multipart files, appending them to the workspace", async () => {
  await expect(
    api.uploadText({ file_name: "petstore.yaml", content: "openapi: 3.0.0" }),
  ).resolves.toMatchObject({ file_name: "petstore.yaml", type: "openapi" });

  const file = new File(["# PRD"], "requirements.md", { type: "text/markdown" });
  await expect(api.uploadFile(file)).resolves.toMatchObject({
    file_name: "requirements.md",
    type: "prd",
  });

  expect((await api.workspace()).items).toHaveLength(7);
});

it("generates a suite and records it in the workspace", async () => {
  const res = await api.generate({
    apis: [{ server: "auth", path: "specs/auth-api.yaml" }],
    req_path: "docs/requirements.md",
    output_dir: "./tests",
    dry_run: false,
  });

  expect(res).toMatchObject({ success: true, scenarios_generated: 6 });
  expect((await api.workspace()).items.some((item) => item.type === "suite")).toBe(true);
});
