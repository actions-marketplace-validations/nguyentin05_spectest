import "@testing-library/jest-dom/vitest";
import { File as NodeFile } from "node:buffer";
import { cleanup } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, vi } from "vitest";
import { server } from "../mocks/server";
import { resetMockStore } from "../mocks/store";

// jsdom's File/FormData cannot be serialized by Node's fetch or parsed by MSW:
// File has no stream() and native FormData rejects jsdom File instances.
const nativeFormData = await new Response(new URLSearchParams()).formData();
globalThis.FormData = nativeFormData.constructor as typeof FormData;
globalThis.File = NodeFile as unknown as typeof File;

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));

afterEach(() => {
  cleanup();
  server.resetHandlers();
  resetMockStore();
  vi.unstubAllGlobals();
});

afterAll(() => server.close());
