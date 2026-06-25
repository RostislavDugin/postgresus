// Thrown by apiHelper for non-2xx responses. Extends Error so existing callers that only read
// `.message` keep working, while callers that care can branch on the machine-readable `code` the
// backend returns (e.g. a classified connection-test failure).
export class ApiError extends Error {
  readonly code?: string;

  constructor(message: string, code?: string) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
  }
}
