export class ApiError extends Error {
  readonly status: number;
  readonly payload?: unknown;

  constructor(status: number, message: string, payload?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

export function isMissingEndpoint(error: unknown): error is ApiError {
  return error instanceof ApiError && [404, 405, 501].includes(error.status);
}
