import { defineConfig } from "cypress";

const billingSpecPattern = [
  "cypress/e2e/catalog/**/*.cy.ts",
  "cypress/e2e/customers/**/*.cy.ts",
  "cypress/e2e/subscriptions/**/*.cy.ts",
  "cypress/e2e/invoices/**/*.cy.ts",
  "cypress/e2e/usage/**/*.cy.ts",
  "cypress/e2e/ledger/**/*.cy.ts",
  "cypress/e2e/taxes/**/*.cy.ts",
  "cypress/e2e/rating/**/*.cy.ts",
  "cypress/e2e/test-clocks/**/*.cy.ts",
  "cypress/e2e/workflows/**/*.cy.ts",
];

export default defineConfig({
  e2e: {
    baseUrl: process.env.CYPRESS_BASE_URL ?? "http://localhost:8080",
    supportFile: "cypress/support/e2e.ts",
    specPattern: billingSpecPattern,
    video: false,
    screenshotOnRunFailure: false,
  },
  env: {
    adminEmail: process.env.CYPRESS_ADMIN_EMAIL ?? "admin@railzway.com",
    adminPassword: process.env.CYPRESS_ADMIN_PASSWORD ?? "password",
    slowCommandMs: Number(process.env.CYPRESS_SLOW_COMMAND_MS ?? 0)
  }
});
