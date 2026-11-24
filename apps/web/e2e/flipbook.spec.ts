import { test, expect } from "@playwright/test";

test.describe("Catalog flipbook editor", () => {
  test("merchant can open flipbook editor from catalog", async ({ page }) => {
    const email = `flip-${Date.now()}@example.com`;
    await page.goto("/auth");
    await page.getByLabel("Email address").fill(email);
    await page.getByRole("button", { name: "Continue with email." }).click();
    await page.getByRole("button", { name: "Open sign-in link" }).click();
    await page.getByRole("button", { name: "Sign in securely." }).click();
    await expect(page.getByRole("heading", { name: /Create your/ })).toBeVisible({
      timeout: 15000,
    });
    await page.getByLabel("Store name").fill("Flipbook Store");
    await page.getByLabel("Store URL").fill(`flipbook-${Date.now()}`);
    await page.locator("#category").selectOption("other");
    await page.getByRole("button", { name: "Continue." }).click();
    await page.getByRole("button", { name: /Ready Stock/ }).click();
    await page.getByRole("button", { name: "Create my store." }).click();
    await expect(page.getByRole("heading", { name: "Flipbook Store" })).toBeVisible({
      timeout: 15000,
    });

    await page.getByRole("link", { name: "Catalog" }).click();
    // Prefer href: store slug often starts with "flipbook-", which substring-matches
    // getByRole(..., { name: "Flipbook" }) against the storefront URL link.
    await page.locator('a[href$="/catalog/flipbook"]').click();
    await expect(page).toHaveURL(/\/catalog\/flipbook$/);
    await expect(page.getByRole("heading", { name: "Catalog flipbook" })).toBeVisible();
    await expect(page.getByText("No catalog pages yet", { exact: false })).toBeVisible();
  });
});
