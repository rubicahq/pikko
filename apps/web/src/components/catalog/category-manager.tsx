"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch, type Category } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export function CategoryManager({
  storeId,
  categories,
}: {
  storeId: string;
  categories: Category[];
}) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);

  async function addCategory(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/categories`, {
        method: "POST",
        body: { name },
      });
      setName("");
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to create category.");
    }
  }

  return (
    <div className="rounded-lg border border-border p-5">
      <h2 className="text-base font-medium text-foreground">Categories</h2>
      <ul className="mt-3 space-y-1 text-sm text-foreground-secondary">
        {categories.length === 0 && <li>No categories yet.</li>}
        {categories.map((c) => (
          <li key={c.id}>{c.name}</li>
        ))}
      </ul>
      <form onSubmit={addCategory} className="mt-4 flex flex-wrap items-end gap-3">
        <div className="min-w-[180px] flex-1 space-y-2">
          <Label htmlFor="cat-name">New category</Label>
          <Input id="cat-name" value={name} onChange={(e) => setName(e.target.value)} required />
        </div>
        <Button type="submit" variant="outline">
          Add
        </Button>
      </form>
      {error && <p className="mt-2 text-sm text-error">{error}</p>}
    </div>
  );
}
