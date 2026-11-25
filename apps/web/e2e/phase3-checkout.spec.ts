import { test, expect } from "@playwright/test";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";

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
  await expect(page.getByLabel("Store URL")).toHaveValue(slug);
  await page.locator("#category").selectOption("other");
  await page.getByRole("button", { name: "Continue." }).click();
  await page.getByRole("button", { name: /Ready Stock/ }).click();
  await page.getByRole("button", { name: "Create my store." }).click();
  await expect(page.getByRole("heading", { name: storeName })).toBeVisible({ timeout: 15000 });
}

function tinyPngPath() {
  const png = Buffer.from(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
    "base64",
  );
  const p = path.join(os.tmpdir(), `pikko-proof-${Date.now()}.png`);
  fs.writeFileSync(p, png);
  return p;
}

test.describe("Phase 3 checkout and payment verification", () => {
  test("buy flow: cart → checkout → evidence → merchant verify", async ({ page }) => {
    const stamp = Date.now();
    const email = `phase3-${stamp}@example.com`;
    const slug = `phase3-shop-${stamp}`;
    await signInAndCreateStore(page, email, "Phase3 Shop", slug);
    const storeId = page.url().match(/\/stores\/([^/]+)/)?.[1];
    expect(storeId).toBeTruthy();

    // Commerce settings
    await page.getByRole("link", { name: "Settings", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Payments & fulfillment" })).toBeVisible();
    await page.getByLabel("Bank name").fill("BCA");
    await page.getByLabel("Account holder").fill("Phase3 Shop");
    await page.getByLabel("Account number").fill("1234567890");
    await page.getByRole("button", { name: "Add bank account" }).click();
    await expect(page.getByText("BCA · 1234567890")).toBeVisible();

    await page.getByText("Flat-rate shipping").click();
    await page.getByLabel("Shipping fee (IDR)").fill("10000");
    await page.getByRole("button", { name: "Save fulfillment" }).click();
    await expect(page.getByText(/Fulfillment settings saved/)).toBeVisible();

    // Create + publish product
    await page.getByRole("link", { name: "Products" }).click();
    await page.getByRole("link", { name: "Add product" }).first().click();
    await page.getByLabel("Name").fill("Phase Tee");
    await page.getByLabel("Price (IDR)").fill("50000");
    await page.getByLabel("Status").selectOption("published");
    await page.getByRole("button", { name: "Save product" }).click();
    await expect(page.getByRole("heading", { name: "Edit product" })).toBeVisible({
      timeout: 15000,
    });

    // Publish storefront
    await page.getByRole("link", { name: "Storefront" }).click();
    await page.getByRole("button", { name: "Publish storefront" }).click();
    await expect(page.getByText("Storefront published.")).toBeVisible({ timeout: 15000 });

    // Customer: add to cart + checkout
    await page.goto(`/${slug}`);
    await expect(page.getByText("Phase Tee")).toBeVisible({ timeout: 15000 });
    await page.getByRole("link", { name: /Phase Tee/ }).click();
    await page.getByRole("button", { name: "Add to cart" }).click();
    await page.getByRole("link", { name: "View cart" }).click();
    await expect(page.getByRole("heading", { name: "Cart" })).toBeVisible();
    await page.getByRole("link", { name: "Checkout" }).click();

    await page.getByLabel("Full name").fill("Customer One");
    await page.getByLabel("Email").fill(`customer-${stamp}@example.com`);
    await page.getByLabel("Phone / WhatsApp").fill("08123456789");
    await page.getByText(/Shipping \(/).click();
    await page.getByLabel("Recipient").fill("Customer One");
    await page.getByLabel("Address").fill("Jl. Contoh 1");
    await page.getByLabel("City").fill("Jakarta");
    await page.getByRole("button", { name: "Place order" }).click();

    await expect(page).toHaveURL(/\/track\//, { timeout: 20000 });
    await expect(page.getByRole("heading", { name: /Order PK-/ })).toBeVisible();
    await expect(page.getByText("Bank transfer")).toBeVisible();

    const proof = tinyPngPath();
    await page.locator("#evidence").setInputFiles(proof);
    await expect(page.getByText(/Payment proof uploaded/)).toBeVisible({ timeout: 15000 });

    // Merchant verifies
    await page.goto(`/stores/${storeId}/orders`);
    await expect(page.getByRole("heading", { name: "Orders" })).toBeVisible({ timeout: 15000 });
    await page.locator("table a").first().click();
    await page.getByRole("button", { name: "Verify payment" }).click();
    await expect(page.getByText("paid", { exact: false }).first()).toBeVisible({ timeout: 15000 });
  });
});
