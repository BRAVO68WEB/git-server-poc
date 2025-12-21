import { createEnv } from "@t3-oss/env-nextjs";
import { z } from "zod";

export const env = createEnv({
  /**
   * Server-side environment variables schema.
   * These are only available on the server and will throw if accessed on the client.
   */
  server: {
    NODE_ENV: z
      .enum(["development", "test", "production"])
      .default("development"),
    STASIS_SERVER_HOSTED_URL: z.url().default("http://localhost:8080"),
    STASIS_SSH_HOST_NAME: z.string().default("localhost:2222"),
  },

  /**
   * Client-side environment variables schema.
   * These are exposed to the client and must be prefixed with `NEXT_PUBLIC_`.
   */
  client: {
    NEXT_PUBLIC_API_URL: z.string().default("http://localhost/api"),
  },

  /**
   * Runtime environment configuration.
   * For Next.js >= 13.4.4, you can use experimental__runtimeEnv for client variables.
   */
  runtimeEnv: {
    NODE_ENV: process.env.NODE_ENV,
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
    STASIS_SERVER_HOSTED_URL: process.env.STASIS_SERVER_HOSTED_URL,
    STASIS_SSH_HOST_NAME: process.env.STASIS_SSH_HOST_NAME,
  },

  /**
   * Skip validation in certain environments (e.g., Docker builds).
   * Set SKIP_ENV_VALIDATION=true to bypass validation.
   */
  skipValidation: !!process.env.SKIP_ENV_VALIDATION,

  /**
   * Treat empty strings as undefined.
   * This allows optional env vars to be truly optional.
   */
  emptyStringAsUndefined: true,
});
