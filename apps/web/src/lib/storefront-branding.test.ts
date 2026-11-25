import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { storefrontIconsMetadata, storefrontLogoSrc } from "./storefront-branding";
import { urls } from "./urls";

describe("storefront branding URLs", () => {
  it("publicFile stays same-origin relative (never api host)", () => {
    const u = urls.publicFile("store-id", "file-id");
    assert.equal(u, "/api/v1/public/files/store-id/file-id");
    assert.ok(!u.includes("api:8080"));
    assert.ok(!u.startsWith("http"));
  });

  it("file stays same-origin relative", () => {
    const u = urls.file("store-id", "file-id");
    assert.equal(u, "/api/v1/stores/store-id/files/file-id");
    assert.ok(!u.startsWith("http"));
  });

  it("storefrontLogoSrc prefers logo_file_id public path", () => {
    assert.equal(
      storefrontLogoSrc("s1", { logo_file_id: "f1", logo_url: "/old" }),
      "/api/v1/public/files/s1/f1",
    );
  });

  it("storefrontIconsMetadata wires favicon + store icon", () => {
    const icons = storefrontIconsMetadata("s1", {
      favicon_file_id: "fav",
      store_icon_file_id: "icon",
    });
    assert.deepEqual(icons, {
      icon: [{ url: "/api/v1/public/files/s1/fav" }],
      apple: [{ url: "/api/v1/public/files/s1/icon" }],
    });
  });
});
