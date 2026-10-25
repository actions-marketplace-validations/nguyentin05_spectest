import { ApiError, isMissingEndpoint } from "./errors";

it("treats missing backend endpoints as capability gaps", () => {
  expect(isMissingEndpoint(new ApiError(404, "not found"))).toBe(true);
  expect(isMissingEndpoint(new ApiError(405, "method not allowed"))).toBe(true);
  expect(isMissingEndpoint(new ApiError(501, "not implemented"))).toBe(true);
});

it("ignores other errors", () => {
  expect(isMissingEndpoint(new ApiError(400, "bad request"))).toBe(false);
  expect(isMissingEndpoint(new ApiError(500, "boom"))).toBe(false);
  expect(isMissingEndpoint(new Error("network"))).toBe(false);
  expect(isMissingEndpoint(null)).toBe(false);
});
