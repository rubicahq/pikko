import { test, expect } from "@playwright/test";

/**
 * Stage 3 consolidated critical smoke (Agent 14).
 * Prefer this over exhaustive UI regression: auth surface, security headers,
 * multi-store isolation entry. Money/tenant depth lives in Go API tests.
 */
test.describe("Stage 3 critical smoke", () => {
  test("security headers include CSP and frame deny", async ({ request }) => {
    const res = await request.get("/");
    expect(res.ok()).toBeTruthy();
    const headers = res.headers();
    expect(headers["x-content-type-options"]).toBe("nosniff");
    expect(headers["x-frame-options"]?.toLowerCase()).toBe("deny");
    expect(headers["content-security-policy"] || "").toContain("frame-ancestors 'none'");
    expect(headers["content-security-policy"] || "").toContain("default-src 'self'");
  });

  test("auth page is reachable without session", async ({ page }) => {
    await page.goto("/auth");
    await expect(page.getByLabel("Email address")).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole("button", { name: /Continue with email/i })).toBeVisible();
  });

  test("multi-store create and switch (E2E-A)", async ({ page }) => {
    const email = `stage3-smoke-${Date.now()}@example.com`;
    await page.goto("/auth");
    await page.getByLabel("Email address").fill(email);
    await page.getByRole("button", { name: "Continue with email." }).click();
    await page.getByRole("button", { name: "Open sign-in link" }).click();
    await page.getByRole("button", { name: "Sign in securely." }).click();
    await expect(page.getByRole("heading", { name: /Create your/ })).toBeVisible({
      timeout: 15000,
    });

    const slugA = `smoke-a-${Date.now()}`;
    await page.getByLabel("Store name").fill("Smoke Alpha");
    await page.getByLabel("Store URL").fill(slugA);
    await page.locator("#category").selectOption("other");
    await page.getByRole("button", { name: "Continue." }).click();
    await page.getByRole("button", { name: /Ready Stock/ }).click();
    await page.getByRole("button", { name: "Create my store." }).click();
    await expect(page.getByRole("heading", { name: "Smoke Alpha" })).toBeVisible({
      timeout: 15000,
    });

    await page.goto("/stores/new");
    const slugB = `smoke-b-${Date.now()}`;
    await page.getByLabel("Store name").fill("Smoke Beta");
    await page.getByLabel("Store URL").fill(slugB);
    await page.locator("#category").selectOption("other");
    await page.getByRole("button", { name: "Continue." }).click();
    await page.getByRole("button", { name: /Preorder/ }).click();
    await page.getByRole("button", { name: "Create my store." }).click();
    await expect(page.getByRole("heading", { name: "Smoke Beta" })).toBeVisible({
      timeout: 15000,
    });

    await page.goto("/stores");
    await expect(page.getByText("Smoke Alpha")).toBeVisible();
    await expect(page.getByText("Smoke Beta")).toBeVisible();
  });
});
