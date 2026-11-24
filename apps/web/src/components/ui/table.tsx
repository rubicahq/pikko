import * as React from "react";
import { cn } from "@/lib/utils";

export function Table({
  className,
  stickyHeader,
  ...props
}: React.HTMLAttributes<HTMLTableElement> & { stickyHeader?: boolean }) {
  return (
    <div
      className={cn(
        "relative w-full overflow-auto rounded-lg border border-border bg-background",
        stickyHeader && "max-h-[min(70vh,720px)]",
      )}
    >
      <table
        className={cn("w-full caption-bottom text-left text-table text-foreground", className)}
        {...props}
      />
    </div>
  );
}

export function TableHeader({ className, ...props }: React.HTMLAttributes<HTMLTableSectionElement>) {
  return <thead className={cn("bg-surface [&_tr]:border-b [&_tr]:border-border", className)} {...props} />;
}

export function TableBody({ className, ...props }: React.HTMLAttributes<HTMLTableSectionElement>) {
  return <tbody className={cn("[&_tr:last-child]:border-0", className)} {...props} />;
}

export function TableRow({ className, ...props }: React.HTMLAttributes<HTMLTableRowElement>) {
  return (
    <tr
      className={cn(
        "border-b border-border transition-colors hover:bg-background-subtle data-[state=selected]:bg-brand-soft",
        className,
      )}
      {...props}
    />
  );
}

export function TableHead({
  className,
  sticky,
  ...props
}: React.ThHTMLAttributes<HTMLTableCellElement> & { sticky?: boolean }) {
  return (
    <th
      className={cn(
        "h-10 px-4 text-left align-middle text-label uppercase tracking-wide text-foreground-secondary",
        sticky && "sticky top-0 z-10 bg-surface",
        className,
      )}
      {...props}
    />
  );
}

export function TableCell({ className, ...props }: React.TdHTMLAttributes<HTMLTableCellElement>) {
  return <td className={cn("px-4 py-3 align-middle", className)} {...props} />;
}
