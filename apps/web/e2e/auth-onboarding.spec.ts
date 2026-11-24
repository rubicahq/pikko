import { test, expect } from "@playwright/test";

test.describe("Phase 1 auth and onboarding", () => {
  test("magic link flow creates first store", async ({ page }) => {
    const email = `merchant-${Date.now()}@example.com`;

    await page.goto("/auth");
    await expect(page.getByRole("heading", { name: "Let's get started." })).toBeVisible();
    await page.getByLabel("Email address").fill(email);
    await page.getByRole("button", { name: "Continue with email." }).click();

    await expect(page.getByRole("heading", { name: "Check your inbox." })).toBeVisible();
    await expect(page.getByText(email)).toBeVisible();

    await page.getByRole("button", { name: "Open sign-in link" }).click();
    await expect(page.getByRole("heading", { name: "Continue to Pikko." })).toBeVisible();
    await page.getByRole("button", { name: "Sign in securely." }).click();

    await expect(page.getByRole("heading", { name: /Create your first store|Create a new store/ })).toBeVisible({
      timeout: 15000,
    });

    await page.getByLabel("Store name").fill("Japan Picks");
    await expect(page.getByLabel("Store URL")).toHaveValue(/japan/);
    await page.locator("#category").selectOption("personal_shopper_jastip");
    await page.getByRole("button", { name: "Continue." }).click();

    await expect(page.getByRole("heading", { name: "How do you sell?" })).toBeVisible();
    await page.getByRole("button", { name: /Both/ }).click();
    await page.getByRole("button", { name: "Create my store." }).click();

    await expect(page.getByRole("heading", { name: "Japan Picks" })).toBeVisible({ timeout: 15000 });
    await expect(page.getByText("Setup checklist")).toBeVisible();
  });

  test("GET confirm peek does not consume; POST confirm works", async ({ page, request }) => {
    const email = `peek-${Date.now()}@example.com`;
    const res = await request.post("/api/v1/auth/magic-link/request", {
      data: { email },
    });
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.dev_token).toBeTruthy();
    const token = body.dev_token as string;

    const peek1 = await request.get(`/api/v1/auth/magic-link/peek?token=${encodeURIComponent(token)}`);
    expect(peek1.ok()).toBeTruthy();
    const peek2 = await request.get(`/api/v1/auth/magic-link/peek?token=${encodeURIComponent(token)}`);
    expect(peek2.ok()).toBeTruthy();

    await page.goto(`/auth/confirm?token=${encodeURIComponent(token)}`);
    await page.getByRole("button", { name: "Sign in securely." }).click();
    await expect(page.getByRole("heading", { name: /Create your/ })).toBeVisible({ timeout: 15000 });

    const peek3 = await request.get(`/api/v1/auth/magic-link/peek?token=${encodeURIComponent(token)}`);
    expect(peek3.status()).toBe(410);
  });
});
