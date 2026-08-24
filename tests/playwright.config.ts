import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  testMatch: /e2e_flow\.spec\.ts/,
  timeout: 30_000,
  use: { viewport: { width: 1440, height: 900 } },
  reporter: [["list"]],
});
