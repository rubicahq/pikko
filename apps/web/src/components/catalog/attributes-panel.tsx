"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import {
  apiFetch,
  ATTRIBUTE_OPTION_COLOR_HEX,
  type AttributeDefinition,
  type AttributeOptionColor,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { urls } from "@/lib/urls";

const TYPE_LABELS: Record<string, string> = {
  single_option: "Single option",
  multiple_option: "Multiple options",
  short_text: "Short text",
  number: "Number",
};

export function AttributesPanel({
  storeId,
  attributes,
}: {
  storeId: string;
  attributes: AttributeDefinition[];
}) {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  async function remove(id: string) {
    if (!confirm("Delete this attribute? Product values for it will be removed.")) return;
    setBusyId(id);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/attributes/${id}`, { method: "DELETE" });
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to delete attribute.");
    } finally {
      setBusyId(null);
    }
  }

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Attributes"
        description="Merchant-defined fields for products — options, text, or numbers."
        actions={
          <Button asChild>
            <Link href={urls.storeAttributeNew(storeId)}>Add attribute</Link>
          </Button>
        }
      />

      {error ? (
        <p className="rounded-[14px] border border-error-border bg-error-bg px-4 py-3 text-sm text-error">
          {error}
        </p>
      ) : null}

      {attributes.length === 0 ? (
        <EmptyState
          title="No attributes yet"
          description="Create fields like Brand or Category, then set values on each product."
          action={
            <Button asChild size="sm">
              <Link href={urls.storeAttributeNew(storeId)}>Add attribute</Link>
            </Button>
          }
        />
      ) : (
        <Table stickyHeader>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead sticky>Name</TableHead>
              <TableHead sticky>Type</TableHead>
              <TableHead sticky>Options</TableHead>
              <TableHead sticky className="text-right">
                Actions
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {attributes.map((a) => {
              const t = a.type || a.data_type;
              return (
                <TableRow key={a.id}>
                  <TableCell>
                    <Link
                      href={urls.storeAttribute(storeId, a.id)}
                      className="font-medium text-foreground hover:underline"
                    >
                      {a.name || a.label}
                    </Link>
                  </TableCell>
                  <TableCell className="text-foreground-secondary">
                    {TYPE_LABELS[t] || t}
                  </TableCell>
                  <TableCell>
                    {a.options?.length ? (
                      <div className="flex flex-wrap gap-2">
                        {a.options.map((o) => (
                          <span
                            key={o.id}
                            className="inline-flex items-center gap-1.5 text-sm text-foreground-secondary"
                          >
                            <span
                              className="inline-block h-2.5 w-2.5 rounded-full"
                              style={{
                                backgroundColor:
                                  ATTRIBUTE_OPTION_COLOR_HEX[
                                    (o.color as AttributeOptionColor) in ATTRIBUTE_OPTION_COLOR_HEX
                                      ? (o.color as AttributeOptionColor)
                                      : "slate"
                                  ],
                              }}
                            />
                            {o.label}
                          </span>
                        ))}
                      </div>
                    ) : (
                      <span className="text-foreground-muted">—</span>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button asChild variant="outline" size="sm">
                        <Link href={urls.storeAttribute(storeId, a.id)}>Edit</Link>
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        loading={busyId === a.id}
                        onClick={() => void remove(a.id)}
                      >
                        Delete
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
