import { fileURLToPath } from "node:url";
import { resolve } from "node:path";
import { config as dotenvConfig } from "dotenv";
import createJiti from "jiti";
import type { NextConfig } from "next";

if (process.env.NODE_ENV == "development") {
  const envPath = resolve(fileURLToPath(import.meta.url), "../../configs/.env");
  dotenvConfig({ path: envPath });
  console.log(envPath);
}

const jiti = createJiti(fileURLToPath(import.meta.url));

// Import env here to validate during build. Using jiti we can import .ts files
jiti("./lib/env");

const nextConfig: NextConfig = {
  /* config options here */
};

export default nextConfig;
