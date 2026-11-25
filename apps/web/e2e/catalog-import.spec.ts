import { test, expect } from "@playwright/test";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";

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

function minimalPdfPath() {
  const pdf = `%PDF-1.4
1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj
2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj
3 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources<< /Font<< /F1 5 0 R >> >> >>endobj
4 0 obj<< /Length 44 >>stream
BT /F1 24 Tf 100 700 Td (Hello) Tj ET
endstream
endobj
5 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj
xref
0 6
0000000000 65535 f 
0000000009 00000 n 
0000000058 00000 n 
0000000115 00000 n 
0000000266 00000 n 
0000000361 00000 n 
trailer<< /Size 6 /Root 1 0 R >>
startxref
429
%%EOF
`;
  const p = path.join(os.tmpdir(), `pikko-e2e-${Date.now()}.pdf`);
  fs.writeFileSync(p, pdf);
  return p;
}

test.describe("PDF catalog flipbook (pages only)", () => {
  test("upload PDF, wait for pages, land on catalog detail", async ({ page }) => {
    const email = `import-${Date.now()}@example.com`;
    await signInAndCreateStore(page, email, "Import Shop", `import-shop-${Date.now()}`);

    await page.getByRole("link", { name: "Catalogs" }).click();
    await expect(page.getByRole("heading", { name: "Catalogs" })).toBeVisible();
    await page.getByRole("link", { name: "Add Catalog" }).click();
    await expect(page.getByRole("heading", { name: "Add Catalog" })).toBeVisible();

    await expect(page.locator("#pdf")).toBeDisabled();
    await page.getByLabel("Catalog name").fill("Summer Lookbook");
    await expect(page.locator("#pdf")).toBeEnabled();

    const pdfPath = minimalPdfPath();
    await page.locator("#pdf").setInputFiles(pdfPath);
    await page.getByRole("button", { name: "Create flipbook" }).click();

    await expect(page.getByRole("heading", { name: "Preparing flipbook" })).toBeVisible({
      timeout: 20000,
    });
    await expect(page.getByRole("heading", { name: "Summer Lookbook" })).toBeVisible({
      timeout: 60000,
    });
    await expect(page.getByRole("link", { name: "Open flipbook" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Review import" })).toHaveCount(0);
  });
});
