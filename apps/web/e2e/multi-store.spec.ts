import { test, expect } from "@playwright/test";

async function signInAndLand(page: import("@playwright/test").Page, email: string) {
  await page.goto("/auth");
  await page.getByLabel("Email address").fill(email);
  await page.getByRole("button", { name: "Continue with email." }).click();
  await page.getByRole("button", { name: "Open sign-in link" }).click();
  await page.getByRole("button", { name: "Sign in securely." }).click();
  await expect(page.getByRole("heading", { name: /Create your/ })).toBeVisible({ timeout: 15000 });
}

async function createStore(
  page: import("@playwright/test").Page,
  name: string,
  slug: string,
  model: "Ready Stock" | "Preorder" | "Both",
) {
  await page.getByLabel("Store name").fill(name);
  await page.getByLabel("Store URL").fill(slug);
  await page.locator("#category").selectOption("other");
  await page.getByRole("button", { name: "Continue." }).click();
  await page.getByRole("button", { name: new RegExp(model) }).click();
  await page.getByRole("button", { name: "Create my store." }).click();
  await expect(page.getByRole("heading", { name })).toBeVisible({ timeout: 15000 });
}

test("additional store, switcher, manage stores", async ({ page }) => {
  const email = `multi-${Date.now()}@example.com`;
  await signInAndLand(page, email);
  await createStore(page, "Alpha Shop", `alpha-${Date.now()}`, "Ready Stock");

  await page.goto("/stores/new");
  await expect(page.getByRole("heading", { name: /Create a new store/ })).toBeVisible();
  await createStore(page, "Beta Shop", `beta-${Date.now()}`, "Preorder");

  await page.goto("/stores");
  await expect(page.getByRole("heading", { name: "Manage stores" })).toBeVisible();
  await expect(page.getByText("Alpha Shop")).toBeVisible();
  await expect(page.getByText("Beta Shop")).toBeVisible();

  await page.getByRole("link", { name: "Open dashboard" }).first().click();
  await expect(page.getByText("Setup checklist")).toBeVisible();
});
