import { defineConfig } from "drizzle-kit";

// drizzle-kit doesn't load .env.local automatically the way Next.js does
process.loadEnvFile(".env.local");

export default defineConfig({
  schema: "./lib/schema.ts",
  out: "./drizzle",
  dialect: "postgresql",
  dbCredentials: {
    url: process.env.DATABASE_URL!,
  },
});
