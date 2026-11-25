"use client";

import { useState } from "react";
import { Copy, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";

type Props = {
  url: string;
};

export function CustomerPortalLink({ url }: Props) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className="space-y-2">
      <div>
        <p className="text-xs font-medium uppercase tracking-wide text-foreground-muted">
          Customer portal link
        </p>
        <p className="mt-0.5 text-sm text-foreground-secondary">
          Share this personal link — opens their portal with secret + phone in the URL (no email login).
        </p>
      </div>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-stretch">
        <input
          readOnly
          value={url}
          aria-label="Customer portal URL"
          className="min-w-0 flex-1 rounded-md border border-border bg-background px-3 py-2 font-mono text-sm text-foreground outline-none focus-visible:ring-2 focus-visible:ring-brand/40"
          onFocus={(e) => e.currentTarget.select()}
        />
        <div className="flex shrink-0 gap-2">
          <Button type="button" variant="secondary" size="sm" onClick={() => void copy()}>
            <Copy className="size-4" aria-hidden />
            {copied ? "Copied" : "Copy"}
          </Button>
          <Button type="button" variant="outline" size="sm" asChild>
            <a href={url} target="_blank" rel="noreferrer">
              <ExternalLink className="size-4" aria-hidden />
              Open
            </a>
          </Button>
        </div>
      </div>
    </div>
  );
}
