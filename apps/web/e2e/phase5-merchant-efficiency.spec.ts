import { test, expect } from "@playwright/test";

async function signInAndCreateStore(
  page: import("@playwright/test").Page,
  email: string,
  storeName: string,
  slug: string,
) {
  await page.goto("/auth");
  await page.getByLabel("Email address").fill(email);
  await page.getByRole("button", { name: "Continue with email." }).click();
  await page.getByRole("button", { name: "Open sign-in link" }).click();
  await page.getByRole("button", { name: "Sign in securely." }).click();
  await expect(page.getByRole("heading", { name: /Create your/ })).toBeVisible({
    timeout: 15000,
  });
  await page.getByLabel("Store name").fill(storeName);
  await page.getByLabel("Store URL").click();
  await page.getByLabel("Store URL").fill(slug);
  await page.locator("#category").selectOption("personal_shopper_jastip");
  await page.getByRole("button", { name: "Continue." }).click();
  await page.getByRole("button", { name: /Both/ }).click();
  await page.getByRole("button", { name: "Create my store." }).click();
  await expect(page.getByRole("heading", { name: storeName })).toBeVisible({ timeout: 15000 });
}

test.describe("Phase 5 merchant efficiency", () => {
  test("customers, analytics, notifications, CSV import surfaces", async ({ page }) => {
    const stamp = Date.now();
    const email = `phase5-${stamp}@example.com`;
    const slug = `phase5-shop-${stamp}`;
    await signInAndCreateStore(page, email, "Phase5 Shop", slug);
    const storeId = page.url().match(/\/stores\/([^/]+)/)?.[1];
    expect(storeId).toBeTruthy();

    await page.getByRole("link", { name: "Customers", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Customers", exact: true })).toBeVisible();
    await expect(page.getByText(/No customers yet/)).toBeVisible();

    await page.getByRole("link", { name: "Analytics", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Analytics" })).toBeVisible();
    await expect(page.getByText(/Verified payments/)).toBeVisible();

    await page.locator("aside").getByRole("link", { name: "Notifications" }).click();
    await expect(page.getByRole("heading", { name: "Notifications" })).toBeVisible();

    await page.locator("aside").getByRole("link", { name: "Products" }).click();
    await page.getByRole("link", { name: "Import CSV", exact: true }).click();
    await expect(page.getByRole("heading", { name: /import products/i })).toBeVisible();
    await expect(page.getByText(/Download CSV template/)).toBeVisible();

    await page.locator("aside").getByRole("link", { name: "Preorders" }).click();
    await page.getByRole("link", { name: "Create batch" }).first().click();
    await page.getByLabel("Batch name").fill("Phase5 Batch");
    await page.getByLabel("URL slug").fill(`phase5-batch-${stamp}`);
    await page.getByRole("button", { name: "Create batch" }).click();
    await expect(page.getByRole("heading", { name: "Phase5 Batch" })).toBeVisible({
      timeout: 15000,
    });
    await expect(page.getByRole("heading", { name: "Batch cover" })).toBeVisible();
  });
});
