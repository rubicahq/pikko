import { AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

export function ErrorState({
  title = "Something went wrong",
  description = "We couldn’t load this view. Try again in a moment.",
  onRetry,
  className,
}: {
  title?: string;
  description?: string;
  onRetry?: () => void;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-start rounded-lg border border-error-border bg-error-bg px-6 py-8",
        className,
      )}
      role="alert"
    >
      <div className="flex items-center gap-2 text-error">
        <AlertCircle className="h-5 w-5" strokeWidth={1.75} />
        <h3 className="text-card">{title}</h3>
      </div>
      <p className="mt-2 max-w-md text-body text-foreground-secondary">{description}</p>
      {onRetry ? (
        <Button type="button" variant="outline" className="mt-5" onClick={onRetry}>
          Try again
        </Button>
      ) : null}
    </div>
  );
}
