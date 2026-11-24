import { test, expect } from "@playwright/test";

test.describe("Phase 7 domain-agnostic launch prep", () => {
  test("sitemap and canonicals use marketing origin from env", async ({ request, page }) => {
    const sitemap = await request.get("/sitemap.xml");
    expect(sitemap.ok()).toBeTruthy();
    const xml = await sitemap.text();
    expect(xml).toContain("<urlset");
    expect(xml).toContain("/pricing");
    // Absolute URLs must not hardcode a foreign production host when local env is localhost.
    const marketing =
      process.env.NEXT_PUBLIC_MARKETING_URL ||
      process.env.PLAYWRIGHT_BASE_URL ||
      "http://127.0.0.1:3000";
    const origin = marketing.replace(/\/$/, "");
    expect(xml).toContain(origin);

    await page.goto("/pricing");
    const canonical = page.locator('link[rel="canonical"]');
    await expect(canonical).toHaveAttribute("href", new RegExp(`^${origin.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}`));
  });

  test("marketing surfaces expose env-driven support contact", async ({ page }) => {
    await page.goto("/faq");
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    const mail = page.locator('a[href^="mailto:"]').first();
    await expect(mail).toBeVisible();
    const href = await mail.getAttribute("href");
    expect(href).toMatch(/^mailto:.+@.+/);
  });
});
