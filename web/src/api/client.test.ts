import { vi } from "vitest";
import { apiRequest } from "./client";
import { ApiError } from "./errors";

it("surfaces the error envelope message and status", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ success: false, error: "Boom" }), { status: 500 }),
    ),
  );
  await expect(apiRequest("/health")).rejects.toMatchObject({ status: 500, message: "Boom" });
  await expect(apiRequest("/health")).rejects.toBeInstanceOf(ApiError);
});

it("surfaces the real backend error envelope", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: true, message: "no spectest.yaml found" }), {
        status: 404,
      }),
    ),
  );
  await expect(apiRequest("/specs/plan")).rejects.toMatchObject({
    status: 404,
    message: "no spectest.yaml found",
  });
});

it("falls back to a plain-text error body", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response("method not allowed", {
        status: 405,
        headers: { "Content-Type": "text/plain; charset=utf-8" },
      }),
    ),
  );
  await expect(apiRequest("/run")).rejects.toMatchObject({
    status: 405,
    message: "method not allowed",
  });
});

it("throws a network error when fetch rejects", async () => {
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
  await expect(apiRequest("/health")).rejects.toMatchObject({ status: 0 });
});

it("turns a proxy gateway failure into a connection message", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("", { status: 502 })));
  await expect(apiRequest("/health")).rejects.toMatchObject({
    status: 502,
    message: "Không kết nối được máy chủ",
  });
});
