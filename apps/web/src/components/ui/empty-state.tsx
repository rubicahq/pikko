import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function EmptyState({
  title,
  description,
  action,
  className,
  icon,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
  icon?: ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-start rounded-lg border border-border bg-background px-6 py-10",
        className,
      )}
    >
      {icon ? <div className="mb-4 text-foreground-muted">{icon}</div> : null}
      <h3 className="text-card text-foreground">{title}</h3>
      {description ? (
        <p className="mt-2 max-w-md text-body text-foreground-secondary">{description}</p>
      ) : null}
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}
