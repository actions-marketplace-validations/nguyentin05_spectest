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
