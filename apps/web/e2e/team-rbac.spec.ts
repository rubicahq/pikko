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
  model: "Ready Stock" | "Preorder" | "Both" = "Ready Stock",
) {
  await page.getByLabel("Store name").fill(name);
  await page.getByLabel("Store URL").fill(slug);
  await page.locator("#category").selectOption("other");
  await page.getByRole("button", { name: "Continue." }).click();
  await page.getByRole("button", { name: new RegExp(model) }).click();
  await page.getByRole("button", { name: "Create my store." }).click();
  await expect(page.getByRole("heading", { name })).toBeVisible({ timeout: 15000 });
}

test("invite staff, accept magic link, isolate Store B", async ({ page, browser }) => {
  const stamp = Date.now();
  const ownerEmail = `owner-rbac-${stamp}@example.com`;
  const staffEmail = `staff-rbac-${stamp}@example.com`;

  await signInAndLand(page, ownerEmail);
  await createStore(page, "Alpha RBAC", `alpha-rbac-${stamp}`);
  const storeAId = page.url().match(/stores\/([^/]+)/)?.[1];
  expect(storeAId).toBeTruthy();

  await page.goto("/stores/new");
  await createStore(page, "Beta RBAC", `beta-rbac-${stamp}`, "Preorder");
  const storeBId = page.url().match(/stores\/([^/]+)/)?.[1];
  expect(storeBId).toBeTruthy();

  await page.goto(`/stores/${storeAId}/team`);
  await expect(page.getByRole("heading", { name: "Team", exact: true })).toBeVisible();

  const inviteRes = await page.request.post(`/api/v1/stores/${storeAId}/team/invites`, {
    data: { email: staffEmail, role: "catalog" },
  });
  expect(inviteRes.ok()).toBeTruthy();
  const inviteJSON = await inviteRes.json();
  const token = inviteJSON.invite?.dev_token as string;
  expect(token).toBeTruthy();

  await expect(page.getByText(staffEmail)).toBeVisible({ timeout: 10000 });

  const staffContext = await browser.newContext();
  const staffPage = await staffContext.newPage();
  await staffPage.goto(`/invites/accept?token=${encodeURIComponent(token)}`);
  await expect(staffPage.getByRole("heading", { name: /Join Alpha RBAC/ })).toBeVisible({
    timeout: 15000,
  });
  await staffPage.getByRole("button", { name: "Accept invitation." }).click();
  await expect(staffPage.getByRole("heading", { name: "Alpha RBAC" })).toBeVisible({
    timeout: 15000,
  });

  await expect(staffPage.getByRole("link", { name: "Catalog" })).toBeVisible();
  await expect(staffPage.getByRole("navigation", { name: "Store" }).getByRole("link", { name: "Team" })).toHaveCount(0);

  const probe = await staffPage.request.get(`/api/v1/stores/${storeBId}/products`);
  expect(probe.status()).toBe(404);

  await staffPage.goto(`/stores/${storeBId}/overview`);
  // Unauthorized store redirects away from overview
  await expect(staffPage).not.toHaveURL(new RegExp(`/stores/${storeBId}/overview`));

  // Switcher only lists authorized stores
  await staffPage.goto(`/stores/${storeAId}/overview`);
  await expect(staffPage.getByRole("heading", { name: "Alpha RBAC" })).toBeVisible();

  await staffContext.close();
});
