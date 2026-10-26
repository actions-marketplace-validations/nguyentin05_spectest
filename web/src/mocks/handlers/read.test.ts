import { http, HttpResponse } from "msw";
import { api } from "../../api/endpoints";
import { server } from "../server";
import { resetMockStore } from "../store";

beforeEach(resetMockStore);

it("serves read endpoints with documented shapes", async () => {
  await expect(api.health()).resolves.toMatchObject({ status: "ok" });
  await expect(api.getConfig()).resolves.toMatchObject({
    has_api_key: true,
    api_key_masked: "AIzaSy...4xK9",
  });
  const plan = await api.plan();
  expect(plan.stats).toMatchObject({
    total_scenarios: 2,
    total_setups: 3,
    locked_scenarios: 1,
  });
  expect(plan.scenarios[1]).toMatchObject({ id: "order-over-stock", locked: true });
  const ws = await api.workspace();
  expect(ws.items.map((i) => i.type)).toEqual(["openapi", "openapi", "prd", "suite", "history"]);
  await expect(api.history()).resolves.toMatchObject({ updates: [{ n: 1 }] });
  await expect(api.markdown("docs/requirements.md")).resolves.toMatchObject({
    file_name: "requirements.md",
  });
  await expect(api.logs()).resolves.toMatchObject({
    logs: expect.arrayContaining([expect.stringContaining("listening on :4173")]),
  });
});

it("returns documented 404s", async () => {
  await expect(api.plan("missing.yaml")).rejects.toMatchObject({ status: 404 });
  await expect(api.history("missing")).rejects.toMatchObject({ status: 404 });
  await expect(api.markdown("docs/missing.md")).rejects.toMatchObject({ status: 400 });
});

it("reads markdown with POST and the path body", async () => {
  let method = "";
  let body: unknown;
  server.use(
    http.post("/api/markdown/read", async ({ request }) => {
      method = request.method;
      body = await request.json();
      return HttpResponse.json({
        path: "docs/requirements.md",
        title: "Requirements",
        sections: [],
        table_count: 1,
        line_count: 2,
        raw_content: "# Requirements",
      });
    }),
  );

  const doc = await api.markdown("docs/requirements.md");

  expect(method).toBe("POST");
  expect(body).toMatchObject({ path: "docs/requirements.md" });
  expect(doc).toMatchObject({
    path: "docs/requirements.md",
    file_name: "requirements.md",
    content: "# Requirements",
  });
});
