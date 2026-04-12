import { defineConfig } from "cypress";

export default defineConfig({
  e2e: {
    baseUrl: process.env.CYPRESS_BASE_URL ?? "http://localhost:8080",
    supportFile: "cypress/support/e2e.ts",
    specPattern: "cypress/e2e/**/*.cy.ts",
    video: false
  },
  env: {
    adminEmail: process.env.CYPRESS_ADMIN_EMAIL ?? "admin@railzway.com",
    adminPassword: process.env.CYPRESS_ADMIN_PASSWORD ?? "password",
    slowCommandMs: Number(process.env.CYPRESS_SLOW_COMMAND_MS ?? 0)
  }
});
