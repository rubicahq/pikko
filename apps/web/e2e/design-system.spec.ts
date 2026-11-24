import { test, expect } from "@playwright/test";

test.describe("Design system (Agent 02)", () => {
  // /dev/components is blocked when NODE_ENV or NEXT_PUBLIC_APP_ENV is production.
  // Run this suite against `next dev` (or a non-production start).
  test("component preview documents operational primitives", async ({ page }) => {
    await page.goto("/dev/components");
    await expect(page.getByRole("heading", { name: "Component preview" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Data table" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Nav IA (live groups)" })).toBeVisible();
    await expect(page.getByText("Workspace")).toBeVisible();
    await expect(page.getByText("Operations")).toBeVisible();
    await expect(page.getByText("Configuration")).toBeVisible();
    // Unimplemented modules stay hidden from live IA preview
    await expect(page.getByText("Packing")).toHaveCount(0);
    await expect(page.getByText("Vendors")).toHaveCount(0);

    await page.getByPlaceholder("Filter orders…").fill("Ayu");
    await expect(page.getByText("PK-1000")).toBeVisible();
  });

  test("marketing home still passes brand hero budget", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: /Sell your way/ })).toBeVisible();
    const overflow = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth + 1;
    });
    expect(overflow).toBeFalsy();
  });
});
