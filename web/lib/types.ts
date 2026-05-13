import { z } from "zod";

// Zod mirrors of the API contract. Drift between the OpenAPI spec and
// these schemas surfaces as a runtime validation error at the seam,
// which is far easier to debug than a downstream type confusion.

export const PackCountSchema = z.object({
  size: z.number().int().positive(),
  count: z.number().int().positive(),
});
export type PackCount = z.infer<typeof PackCountSchema>;

export const PackSetSchema = z.object({
  sizes: z.array(z.number().int().positive()),
  version: z.string(),
});
export type PackSet = z.infer<typeof PackSetSchema>;

export const CalculateResponseSchema = z.object({
  packs: z.array(PackCountSchema),
  total_items: z.number().int().nonnegative(),
  total_packs: z.number().int().nonnegative(),
  overshoot: z.number().int().nonnegative(),
});
export type CalculateResponse = z.infer<typeof CalculateResponseSchema>;

export const ErrorResponseSchema = z.object({
  code: z.string(),
  message: z.string(),
  request_id: z.string().optional().default(""),
});
export type ErrorResponse = z.infer<typeof ErrorResponseSchema>;

// ApiError carries the full envelope so UI surfaces can show the
// machine-readable code, the human message, and the request id for
// support correlation.
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId: string;

  constructor(status: number, code: string, message: string, requestId: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}
