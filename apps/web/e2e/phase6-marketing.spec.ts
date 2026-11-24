import { test, expect } from "@playwright/test";

test.describe("Phase 6 marketing and SEO", () => {
  test("landing shows brand hero and primary CTA", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Sell your way." })).toBeVisible();
    await expect(page.getByRole("link", { name: "Create your store." }).first()).toBeVisible();
    await expect(page.getByRole("navigation", { name: "Primary" })).toBeVisible();
  });

  test("pricing, FAQ, legal, and solutions are reachable", async ({ page }) => {
    for (const path of [
      "/pricing",
      "/faq",
      "/privacy",
      "/terms",
      "/solutions",
      "/solutions/jastip",
      "/solutions/social-sellers",
    ]) {
      await page.goto(path);
      await expect(page.locator("h1")).toBeVisible();
      await expect(page.getByRole("contentinfo")).toBeVisible();
    }
  });

  test("FAQ disclosures are keyboard accessible", async ({ page }) => {
    await page.goto("/faq");
    const first = page.locator("details").first();
    await expect(first).toBeVisible();
    await first.locator("summary").focus();
    await page.keyboard.press("Enter");
    await expect(first).toHaveAttribute("open", "");
  });

  test("robots and sitemap are served", async ({ request }) => {
    const robots = await request.get("/robots.txt");
    expect(robots.ok()).toBeTruthy();
    const robotsBody = await robots.text();
    expect(robotsBody.toLowerCase()).toContain("user-agent");

    const sitemap = await request.get("/sitemap.xml");
    expect(sitemap.ok()).toBeTruthy();
    const xml = await sitemap.text();
    expect(xml).toContain("<urlset");
    expect(xml).toContain("/pricing");
  });

  test("security headers present on marketing response", async ({ request }) => {
    const res = await request.get("/");
    expect(res.ok()).toBeTruthy();
    const headers = res.headers();
    expect(headers["x-content-type-options"]).toBe("nosniff");
    expect(headers["x-frame-options"]?.toLowerCase()).toBe("deny");
  });
});
