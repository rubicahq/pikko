import { test, expect } from "@playwright/test";
import path from "path";
import fs from "fs";

async function signInAndCreateStore(page: import("@playwright/test").Page, name: string, slug: string) {
  const email = `phase2-${Date.now()}-${Math.random().toString(16).slice(2)}@example.com`;
  await page.goto("/auth");
  await page.getByLabel("Email address").fill(email);
  await page.getByRole("button", { name: "Continue with email." }).click();
  await page.getByRole("button", { name: "Open sign-in link" }).click();
  await page.getByRole("button", { name: "Sign in securely." }).click();
  await expect(page.getByRole("heading", { name: /Create your/ })).toBeVisible({ timeout: 15000 });
  await page.getByLabel("Store name").fill(name);
  // Mark URL touched before/while setting it so async suggest-slug cannot race-append.
  await page.getByLabel("Store URL").click();
  await page.getByLabel("Store URL").fill(slug);
  await expect(page.getByLabel("Store URL")).toHaveValue(slug);
  await page.locator("#category").selectOption("other");
  await page.getByRole("button", { name: "Continue." }).click();
  await page.getByRole("button", { name: /Ready Stock/ }).click();
  await page.getByRole("button", { name: "Create my store." }).click();
  await expect(page.getByRole("heading", { name })).toBeVisible({ timeout: 15000 });
  return email;
}

function tinyPngPath() {
  const p = path.join("/tmp", `pikko-product-${Date.now()}.png`);
  // 1x1 PNG
  const b64 =
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==";
  fs.writeFileSync(p, Buffer.from(b64, "base64"));
  return p;
}

test.describe("Phase 2 catalog and storefront", () => {
  test("create product, publish storefront, view public page", async ({ page }) => {
    const slug = `phase2-${Date.now().toString(36)}`;
    await signInAndCreateStore(page, "Phase Two Shop", slug);

    // Prefer the overview storefront link (host may be localhost or 127.0.0.1).
    const storefrontHref = await page
      .locator('main a[href*="://"]')
      .first()
      .getAttribute("href", { timeout: 5000 });
    const liveSlug = storefrontHref?.split("/").filter(Boolean).pop() || slug;
    expect(liveSlug).toBe(slug);

    await page.getByRole("link", { name: "Catalog" }).click();
    await expect(page.getByRole("heading", { name: "Catalog" })).toBeVisible();
    await page.getByRole("link", { name: "Add product" }).first().click();
    await page.getByLabel("Name").fill("Canvas Tote");
    await page.getByLabel("SKU (optional)").fill("TOTE-22");
    await page.getByLabel("Price (IDR)").fill("175000");
    await page.getByLabel("Status").selectOption("published");
    await page.getByLabel("Images").setInputFiles(tinyPngPath());
    await page.getByRole("button", { name: "Save product" }).click();
    await expect(page.getByRole("heading", { name: "Edit product" })).toBeVisible({ timeout: 15000 });

    await page.getByRole("link", { name: "Storefront" }).click();
    await expect(page.getByRole("heading", { name: "Storefront" })).toBeVisible();
    await page.getByLabel("Store description").fill("Handpicked goods for everyday carry.");
    await page.getByLabel("Store description").blur();
    await expect(page.getByText("Draft saved.")).toBeVisible({ timeout: 10000 });
    await page.getByLabel("Theme").selectOption({ label: "Sakura" });
    await expect(page.getByText("Draft saved.")).toBeVisible({ timeout: 10000 });
    await page.getByRole("button", { name: "Publish storefront" }).click();
    await expect(page.getByText("Storefront published.")).toBeVisible({ timeout: 15000 });

    await page.goto(`/${liveSlug}`);
    await expect(page.getByRole("heading", { name: "Phase Two Shop" })).toBeVisible({ timeout: 15000 });
    await expect(page.getByText("Handpicked goods for everyday carry.")).toBeVisible();
    await expect(page.getByText("Canvas Tote")).toBeVisible();
    await page.getByRole("link", { name: /Canvas Tote/ }).click();
    await expect(page.getByRole("heading", { name: "Canvas Tote" })).toBeVisible();
    await expect(page.getByText("SKU TOTE-22")).toBeVisible();
    await expect(page.getByText("Rp175.000")).toBeVisible();
  });

  test("cross-store catalog isolation via API", async ({ playwright }) => {
    async function createUserStore(prefix: string) {
      const ctx = await playwright.request.newContext({
        baseURL: process.env.PLAYWRIGHT_BASE_URL || "http://127.0.0.1:3000",
      });
      const email = `${prefix}-${Date.now()}@example.com`;
      const req = await ctx.post("/api/v1/auth/magic-link/request", { data: { email } });
      expect(req.ok()).toBeTruthy();
      const { dev_token } = await req.json();
      const verify = await ctx.post("/api/v1/auth/magic-link/verify", {
        data: { token: dev_token },
      });
      expect(verify.ok()).toBeTruthy();
      const identity = await ctx.post("/api/v1/onboarding/identity", {
        data: {
          name: `${prefix} Store`,
          slug: `${prefix}-${Date.now().toString(36)}`,
          category: "other",
        },
      });
      expect(identity.ok()).toBeTruthy();
      const selling = await ctx.post("/api/v1/onboarding/selling-model", {
        data: { selling_model: "ready_stock" },
      });
      expect(selling.ok()).toBeTruthy();
      const complete = await ctx.post("/api/v1/onboarding/complete");
      expect(complete.ok()).toBeTruthy();
      const body = await complete.json();
      return { ctx, storeId: body.store.id as string };
    }

    const a = await createUserStore("iso-a");
    const b = await createUserStore("iso-b");

    const create = await a.ctx.post(`/api/v1/stores/${a.storeId}/products`, {
      data: { product: { name: "Secret", base_price_idr: 1000, status: "published" } },
    });
    expect(create.ok()).toBeTruthy();
    const { product } = await create.json();

    const cross = await b.ctx.get(`/api/v1/stores/${a.storeId}/products/${product.id}`);
    expect(cross.status()).toBe(404);

    const crossList = await b.ctx.get(`/api/v1/stores/${a.storeId}/products`);
    expect(crossList.status()).toBe(404);

    await a.ctx.dispose();
    await b.ctx.dispose();
  });
});
