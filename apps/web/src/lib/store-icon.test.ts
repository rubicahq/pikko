import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { storeIconSrc, storeNameInitial } from "./store-icon";

describe("storeIconSrc", () => {
  it("prefers same-origin file path from store_icon_file_id", () => {
    const src = storeIconSrc({
      id: "store-1",
      store_icon_file_id: "file-1",
      store_icon_url: "http://api:8080/api/v1/stores/store-1/files/file-1",
    });
    assert.equal(src, "/api/v1/stores/store-1/files/file-1");
  });

  it("accepts relative store_icon_url when file id missing", () => {
    const src = storeIconSrc({
      id: "store-1",
      store_icon_url: "/api/v1/stores/store-1/files/file-2",
    });
    assert.equal(src, "/api/v1/stores/store-1/files/file-2");
  });

  it("ignores absolute cross-host URLs (Docker api host)", () => {
    const src = storeIconSrc({
      id: "store-1",
      store_icon_url: "http://api:8080/api/v1/stores/store-1/files/file-1",
    });
    assert.equal(src, undefined);
  });

  it("returns undefined when no icon saved", () => {
    assert.equal(storeIconSrc({ id: "store-1" }), undefined);
  });
});

describe("storeNameInitial", () => {
  it("uses first letter uppercased", () => {
    assert.equal(storeNameInitial("Rick Store"), "R");
  });

  it("falls back when name empty", () => {
    assert.equal(storeNameInitial("  "), "S");
  });
});
