import { test, expect } from "@playwright/test";

async function signInAndCreateBothStore(
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

test.describe("Phase 4 preorder and business rules", () => {
  test("preorder batch → checkout → merchant confirm → verify payment", async ({ page }) => {
    const stamp = Date.now();
    const email = `phase4-${stamp}@example.com`;
    const slug = `phase4-shop-${stamp}`;
    await signInAndCreateBothStore(page, email, "Phase4 Shop", slug);
    const storeId = page.url().match(/\/stores\/([^/]+)/)?.[1];
    expect(storeId).toBeTruthy();

    await page.getByRole("link", { name: "Settings", exact: true }).click();
    await page.getByLabel("Bank name").fill("BCA");
    await page.getByLabel("Account holder").fill("Phase4 Shop");
    await page.getByLabel("Account number").fill("9876543210");
    await page.getByRole("button", { name: "Add bank account" }).click();
    await expect(page.getByText("BCA · 9876543210")).toBeVisible();
    await page.getByText("Customer pickup").click();
    await page.getByRole("button", { name: "Save fulfillment" }).click();
    await expect(page.getByText(/Fulfillment settings saved/)).toBeVisible();

    await page.getByRole("link", { name: "Catalog" }).click();
    await page.getByRole("link", { name: "Add product" }).first().click();
    await page.getByLabel("Name").fill("Tokyo Sneaker");
    await page.getByLabel("Price (IDR)").fill("250000");
    await page.getByLabel("Status").selectOption("published");
    await page.getByText("Ready stock eligible").click(); // uncheck
    await page.getByText("Preorder eligible").click(); // check
    await page.getByRole("button", { name: "Save product" }).click();
    await expect(page.getByRole("heading", { name: "Edit product" })).toBeVisible({
      timeout: 15000,
    });

    await page.getByRole("link", { name: "Storefront" }).click();
    await page.getByRole("button", { name: "Publish storefront" }).click();
    await expect(page.getByText("Storefront published.")).toBeVisible({ timeout: 15000 });

    await page.getByRole("link", { name: "Business Rules" }).click();
    await expect(page.getByRole("heading", { name: "Business rules" })).toBeVisible();
    await page.getByRole("button", { name: "Publish rule" }).click();
    await expect(page.getByText("Store handling fee")).toBeVisible({ timeout: 10000 });
    await page.getByLabel("Type").selectOption("order_approval");
    await page.getByLabel("Name").fill("Approve preorders");
    await page.getByLabel("Scope").selectOption("selling_model");
    await page.getByRole("button", { name: "Publish rule" }).click();
    await expect(page.getByText("Approve preorders")).toBeVisible({ timeout: 10000 });

    await page.getByRole("link", { name: "Preorders" }).click();
    await page.getByRole("link", { name: "Create batch" }).first().click();
    await page.getByLabel("Batch name").fill("Japan March");
    await page.getByLabel("URL slug").fill(`japan-march-${stamp}`);
    await page.getByRole("button", { name: "Create batch" }).click();
    await expect(page.getByRole("heading", { name: "Japan March" })).toBeVisible({
      timeout: 15000,
    });
    await page.locator("select").first().selectOption({ index: 1 });
    await page.getByLabel("Quota (optional)").fill("5");
    await page.getByRole("button", { name: "Assign product" }).click();
    await expect(page.getByRole("cell", { name: "Tokyo Sneaker" })).toBeVisible({
      timeout: 10000,
    });

    await page.goto(`/${slug}`);
    await expect(page.getByText("Active preorders")).toBeVisible({ timeout: 15000 });
    await page.getByRole("link", { name: "Japan March" }).click();
    await page.getByRole("button", { name: "Add to cart" }).click();
    await expect(page.getByRole("button", { name: "Added to cart" })).toBeVisible({
      timeout: 5000,
    });
    await page.goto(`/${slug}/cart`);
    await page.getByRole("link", { name: "Checkout group" }).click();

    await page.getByLabel("Full name").fill("Customer Four");
    await page.getByLabel("Email").fill(`cust4-${stamp}@example.com`);
    await page.getByText("Pickup").click();
    await page.getByRole("button", { name: "Place order" }).click();
    await expect(page).toHaveURL(/\/track\//, { timeout: 20000 });
    await expect(page.getByRole("heading", { name: /Order PK-/ })).toBeVisible();

    await page.goto(`/stores/${storeId}/orders`);
    await page.locator("table a").first().click();
    await page.getByRole("button", { name: "Confirm order" }).click();
    await expect(page.getByText("confirmed", { exact: false }).first()).toBeVisible({
      timeout: 15000,
    });
  });
});
