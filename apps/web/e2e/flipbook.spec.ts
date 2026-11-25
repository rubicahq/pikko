import { test, expect } from "@playwright/test";

test.describe("Catalog flipbook editor", () => {
  test("merchant can open flipbook editor from catalogs", async ({ page }) => {
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
    const slug = `flipbook-${Date.now()}`;
    await page.getByLabel("Store URL").click();
    await page.getByLabel("Store URL").fill(slug);
    await expect(page.getByLabel("Store URL")).toHaveValue(slug);
    await page.locator("#category").selectOption("other");
    await page.getByRole("button", { name: "Continue." }).click();
    await page.getByRole("button", { name: /Ready Stock/ }).click();
    await page.getByRole("button", { name: "Create my store." }).click();
    await expect(page.getByRole("heading", { name: "Flipbook Store" })).toBeVisible({
      timeout: 15000,
    });

    await page.getByRole("link", { name: "Catalogs" }).click();
    await expect(page.getByRole("heading", { name: "Catalogs" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Add Catalog" })).toBeVisible();

    const storeId = page.url().match(/\/stores\/([^/]+)/)?.[1];
    expect(storeId).toBeTruthy();
    await page.goto(`/stores/${storeId}/catalogs/flipbook`);
    await expect(page).toHaveURL(/\/catalogs\/flipbook$/);
    await expect(page.getByRole("heading", { name: "Catalog flipbook" })).toBeVisible();
    await expect(page.getByText("No catalog pages yet", { exact: false })).toBeVisible();
  });
});
