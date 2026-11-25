"use client";

import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { urls } from "@/lib/urls";

type Page = {
  id: string;
  image_url: string;
  display_order: number;
  status: string;
  page_number?: number | null;
};

export function FlipbookEditor({ storeId, storeSlug }: { storeId: string; storeSlug: string }) {
  const [pages, setPages] = useState<Page[]>([]);
  const [selectedPage, setSelectedPage] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    const pRes = await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-pages`), {
      credentials: "include",
    });
    if (pRes.ok) {
      const data = await pRes.json();
      setPages(data.pages || []);
      setSelectedPage((prev) => prev || data.pages?.[0]?.id || null);
    }
  }, [storeId]);

  useEffect(() => {
    load();
  }, [load]);

  const page = pages.find((p) => p.id === selectedPage) || null;

  async function movePage(id: string, dir: -1 | 1) {
    const idx = pages.findIndex((p) => p.id === id);
    const j = idx + dir;
    if (idx < 0 || j < 0 || j >= pages.length) return;
    const next = [...pages];
    const [item] = next.splice(idx, 1);
    next.splice(j, 0, item);
    setPages(next);
    await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-pages/order`), {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ page_ids: next.map((p) => p.id) }),
    });
  }

  async function setStatus(status: "draft" | "published") {
    setError(null);
    const res = await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-pages/status`), {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status }),
    });
    if (!res.ok) {
      setError("Unable to update publication status.");
      return;
    }
    setMessage(status === "published" ? "Catalog pages published." : "Catalog pages set to draft.");
    await load();
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <Button type="button" onClick={() => setStatus("published")}>
          Publish catalog
        </Button>
        <Button type="button" variant="outline" onClick={() => setStatus("draft")}>
          Unpublish
        </Button>
        <a
          className="text-sm text-brand-dark underline-offset-4 hover:underline"
          href={urls.storefrontCatalog(storeSlug)}
          target="_blank"
          rel="noreferrer"
        >
          Open public flipbook
        </a>
      </div>
      {message && <p className="text-sm text-success">{message}</p>}
      {error && <p className="text-sm text-error">{error}</p>}

      {pages.length === 0 ? (
        <p className="text-sm text-foreground-secondary">
          No catalog pages yet. Upload a PDF under Add Catalog to generate page images, then return
          here to reorder and publish.
        </p>
      ) : (
        <div className="grid gap-6 lg:grid-cols-[220px_1fr]">
          <ul className="space-y-2">
            {pages.map((p, i) => (
              <li key={p.id}>
                <button
                  type="button"
                  onClick={() => setSelectedPage(p.id)}
                  className={`flex w-full items-center gap-2 rounded-md border px-2 py-2 text-left text-sm ${
                    selectedPage === p.id ? "border-brand bg-brand-soft" : "border-border"
                  }`}
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src={p.image_url} alt="" className="h-12 w-9 object-cover" />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate">Page {i + 1}</span>
                    <Badge tone={p.status === "published" ? "success" : "neutral"}>{p.status}</Badge>
                  </span>
                </button>
                <div className="mt-1 flex gap-1">
                  <Button type="button" size="sm" variant="ghost" onClick={() => movePage(p.id, -1)}>
                    Up
                  </Button>
                  <Button type="button" size="sm" variant="ghost" onClick={() => movePage(p.id, 1)}>
                    Down
                  </Button>
                </div>
              </li>
            ))}
          </ul>

          {page && (
            <div className="space-y-4">
              <div className="relative max-w-2xl border border-border bg-background">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={page.image_url} alt="" className="w-full select-none" draggable={false} />
              </div>
              <p className="text-xs text-foreground-secondary">
                Display flipbook only — product hotspots and AI import are retired.
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
