"use client";

import * as React from "react";
import { ArrowDown, ArrowUp, ArrowUpDown, ChevronLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { EmptyState } from "@/components/ui/empty-state";

export type DataTableColumn<T> = {
  id: string;
  header: string;
  accessor?: (row: T) => string | number | null | undefined;
  cell?: (row: T) => React.ReactNode;
  sortable?: boolean;
  align?: "left" | "right";
  className?: string;
};

export type DataTableProps<T> = {
  columns: DataTableColumn<T>[];
  data: T[];
  rowKey: (row: T) => string;
  filterPlaceholder?: string;
  pageSizeOptions?: number[];
  defaultPageSize?: number;
  stickyHeader?: boolean;
  emptyTitle?: string;
  emptyDescription?: string;
  className?: string;
  toolbar?: React.ReactNode;
};

type SortState = { id: string; dir: "asc" | "desc" } | null;

function compareValues(a: string | number | null | undefined, b: string | number | null | undefined) {
  if (a == null && b == null) return 0;
  if (a == null) return -1;
  if (b == null) return 1;
  if (typeof a === "number" && typeof b === "number") return a - b;
  return String(a).localeCompare(String(b), "id", { sensitivity: "base", numeric: true });
}

export function DataTable<T>({
  columns,
  data,
  rowKey,
  filterPlaceholder = "Filter…",
  pageSizeOptions = [10, 25, 50],
  defaultPageSize = 10,
  stickyHeader = true,
  emptyTitle = "No results",
  emptyDescription = "Try adjusting filters or check back later.",
  className,
  toolbar,
}: DataTableProps<T>) {
  const [query, setQuery] = React.useState("");
  const [sort, setSort] = React.useState<SortState>(null);
  const [page, setPage] = React.useState(0);
  const [pageSize, setPageSize] = React.useState(defaultPageSize);

  const filtered = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return data;
    return data.filter((row) =>
      columns.some((col) => {
        const value = col.accessor?.(row);
        return value != null && String(value).toLowerCase().includes(q);
      }),
    );
  }, [columns, data, query]);

  const sorted = React.useMemo(() => {
    if (!sort) return filtered;
    const col = columns.find((c) => c.id === sort.id);
    if (!col?.accessor) return filtered;
    const copy = [...filtered];
    copy.sort((a, b) => {
      const result = compareValues(col.accessor?.(a), col.accessor?.(b));
      return sort.dir === "asc" ? result : -result;
    });
    return copy;
  }, [columns, filtered, sort]);

  const pageCount = Math.max(1, Math.ceil(sorted.length / pageSize));
  const safePage = Math.min(page, pageCount - 1);
  const pageRows = sorted.slice(safePage * pageSize, safePage * pageSize + pageSize);

  React.useEffect(() => {
    setPage(0);
  }, [query, pageSize, data]);

  function toggleSort(id: string) {
    setSort((prev) => {
      if (!prev || prev.id !== id) return { id, dir: "asc" };
      if (prev.dir === "asc") return { id, dir: "desc" };
      return null;
    });
  }

  return (
    <div className={cn("space-y-3", className)}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={filterPlaceholder}
          className="h-10 max-w-sm"
          aria-label={filterPlaceholder}
        />
        {toolbar}
      </div>

      {sorted.length === 0 ? (
        <EmptyState title={emptyTitle} description={emptyDescription} />
      ) : (
        <>
          <Table stickyHeader={stickyHeader}>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                {columns.map((col) => {
                  const active = sort?.id === col.id;
                  const SortIcon =
                    !active ? ArrowUpDown : sort?.dir === "asc" ? ArrowUp : ArrowDown;
                  return (
                    <TableHead
                      key={col.id}
                      sticky={stickyHeader}
                      className={cn(col.align === "right" && "text-right", col.className)}
                    >
                      {col.sortable ? (
                        <button
                          type="button"
                          onClick={() => toggleSort(col.id)}
                          className="inline-flex items-center gap-1.5 text-label uppercase tracking-wide text-foreground-secondary hover:text-foreground"
                        >
                          {col.header}
                          <SortIcon className="h-3.5 w-3.5" strokeWidth={1.75} />
                        </button>
                      ) : (
                        col.header
                      )}
                    </TableHead>
                  );
                })}
              </TableRow>
            </TableHeader>
            <TableBody>
              {pageRows.map((row) => (
                <TableRow key={rowKey(row)}>
                  {columns.map((col) => (
                    <TableCell
                      key={col.id}
                      className={cn(col.align === "right" && "text-right tabular-nums", col.className)}
                    >
                      {col.cell
                        ? col.cell(row)
                        : col.accessor
                          ? String(col.accessor(row) ?? "")
                          : null}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>

          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-caption text-foreground-muted">
              {sorted.length} row{sorted.length === 1 ? "" : "s"}
              {query.trim() ? ` matching “${query.trim()}”` : ""}
            </p>
            <div className="flex flex-wrap items-center gap-2">
              <label className="flex items-center gap-2 text-label text-foreground-secondary">
                Rows
                <select
                  className="h-9 rounded-md border border-border bg-background px-2 text-sm"
                  value={pageSize}
                  onChange={(e) => setPageSize(Number(e.target.value))}
                >
                  {pageSizeOptions.map((n) => (
                    <option key={n} value={n}>
                      {n}
                    </option>
                  ))}
                </select>
              </label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={safePage <= 0}
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                aria-label="Previous page"
              >
                <ChevronLeft className="h-4 w-4" strokeWidth={1.75} />
              </Button>
              <span className="min-w-[5rem] text-center text-label text-foreground-secondary">
                {safePage + 1} / {pageCount}
              </span>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={safePage >= pageCount - 1}
                onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
                aria-label="Next page"
              >
                <ChevronRight className="h-4 w-4" strokeWidth={1.75} />
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
