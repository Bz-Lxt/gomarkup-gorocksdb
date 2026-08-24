import { expect, test } from "@playwright/test";

const base = process.env.GOROCKSDB_E2E_BASE || "http://127.0.0.1:28741";

test.describe("GoRocksDB 大屏", () => {
  test("首屏标题与控制台", async ({ page }) => {
    const t0 = Date.now();
    await page.goto(base);
    await expect(page.getByRole("heading", { name: /LSM 测深控制室/ })).toBeVisible();
    expect(Date.now() - t0).toBeLessThan(8000);
    await expect(page.getByRole("heading", { name: "手工探测" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "潮池写入面" })).toBeVisible();
  });

  test("真实 Put/Get 闭环", async ({ page }) => {
    await page.goto(base);
    const key = `e2e-${Date.now()}`;
    await page.locator("input").nth(0).fill(key);
    await page.locator("input").nth(1).fill("reef");
    await page.getByRole("button", { name: "写入键值" }).click();
    await expect(page.getByText("已写入")).toBeVisible();
    await page.locator("input").nth(1).fill("");
    await page.getByRole("button", { name: "读取键值" }).click();
    await expect(page.getByText("已读取")).toBeVisible();
    await expect(page.locator("input").nth(1)).toHaveValue("reef");
  });
});
