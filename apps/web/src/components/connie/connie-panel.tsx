"use client";

import { useEffect, useRef, useState } from "react";
import { Loader2, Sparkles, Wrench } from "lucide-react";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";

type ConnieTool = {
  name: string;
  description: string;
  inputSchema?: Record<string, unknown>;
};

type ChatMessage = {
  role: "user" | "assistant";
  content: string;
  toolsUsed?: string[];
};

type ChatResponse = {
  message: { role: string; content: string };
  tools_used?: string[];
  ai_configured: boolean;
  mode: string;
};

const QUICK_PROMPTS = [
  "List pending orders",
  "Status storefront",
  "List products",
  "Subscription usage",
  "Packing queue",
];

export function ConniePanel({
  storeId,
  triggerClassName,
}: {
  storeId: string;
  triggerClassName?: string;
}) {
  const [open, setOpen] = useState(false);
  const [input, setInput] = useState("");
  const [busy, setBusy] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [tools, setTools] = useState<ConnieTool[]>([]);
  const [aiConfigured, setAiConfigured] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    async function loadTools() {
      try {
        const data = await apiFetch<{
          tools: ConnieTool[];
          ai_configured: boolean;
        }>(`/api/v1/stores/${storeId}/connie/tools`);
        if (cancelled) return;
        setTools(data.tools || []);
        setAiConfigured(!!data.ai_configured);
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof ApiError ? e.message : "Unable to load Connie tools.");
        }
      }
    }
    loadTools();
    return () => {
      cancelled = true;
    };
  }, [open, storeId]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, busy]);

  async function send(text: string) {
    const content = text.trim();
    if (!content || busy) return;
    setError(null);
    setInput("");
    const nextMessages: ChatMessage[] = [...messages, { role: "user", content }];
    setMessages(nextMessages);
    setBusy(true);
    try {
      const data = await apiFetch<ChatResponse>(
        `/api/v1/stores/${storeId}/connie/chat`,
        {
          method: "POST",
          body: {
            messages: nextMessages.map((m) => ({
              role: m.role,
              content: m.content,
            })),
          },
        },
      );
      setMessages((prev) => [
        ...prev,
        {
          role: "assistant",
          content: data.message?.content || "…",
          toolsUsed: data.tools_used,
        },
      ]);
      setAiConfigured(!!data.ai_configured);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Connie chat failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <button
          type="button"
          className={cn(
            "inline-flex h-9 max-w-[11rem] items-center gap-1.5 rounded-md bg-warm-feature px-2.5",
            "text-sm font-medium text-foreground hover:brightness-[0.98]",
            triggerClassName,
          )}
          aria-label="Ask Connie"
          title="Ask Connie"
        >
          <Sparkles className="h-4 w-4 shrink-0 text-brand" strokeWidth={1.75} />
          <span className="truncate">Ask Connie</span>
        </button>
      </SheetTrigger>
      <SheetContent
        side="right"
        className="w-[min(100%,28rem)] bg-warm-feature/40"
      >
        <SheetHeader className="bg-background">
          <SheetTitle className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-brand" strokeWidth={1.75} />
            Ask Connie
          </SheetTitle>
          <p className="mt-1 text-sm text-foreground-secondary">
            Store ops teammate — {tools.length} tools
            {aiConfigured ? " · AI on" : " · tool mode"}
          </p>
        </SheetHeader>

        <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden p-4">
          <div className="flex flex-wrap gap-1.5">
            {QUICK_PROMPTS.map((p) => (
              <button
                key={p}
                type="button"
                disabled={busy}
                onClick={() => send(p)}
                className="rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground-secondary hover:bg-surface-hover"
              >
                {p}
              </button>
            ))}
          </div>

          <div className="min-h-0 flex-1 space-y-3 overflow-y-auto rounded-lg border border-border bg-background p-3">
            {messages.length === 0 ? (
              <div className="space-y-2 text-sm text-foreground-secondary">
                <p>
                  Tanya Connie tentang orders, products, catalogs, customers,
                  campaigns, packing, finance, atau subscription.
                </p>
                <p className="flex items-start gap-1.5 text-xs text-foreground-muted">
                  <Wrench className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                  Setiap jawaban bisa memanggil MCP tools dengan isolasi toko +
                  RBAC.
                </p>
              </div>
            ) : null}
            {messages.map((m, i) => (
              <div
                key={`${m.role}-${i}`}
                className={cn(
                  "rounded-lg px-3 py-2 text-sm whitespace-pre-wrap",
                  m.role === "user"
                    ? "ml-6 bg-ink text-foreground-inverse"
                    : "mr-2 bg-warm-feature text-foreground",
                )}
              >
                {m.content}
                {m.toolsUsed && m.toolsUsed.length > 0 ? (
                  <div className="mt-2 flex flex-wrap gap-1">
                    {m.toolsUsed.map((t) => (
                      <span
                        key={t}
                        className="rounded bg-background/70 px-1.5 py-0.5 text-[10px] font-medium text-foreground-muted"
                      >
                        {t}
                      </span>
                    ))}
                  </div>
                ) : null}
              </div>
            ))}
            {busy ? (
              <div className="flex items-center gap-2 text-sm text-foreground-muted">
                <Loader2 className="h-4 w-4 animate-spin" />
                Connie is working…
              </div>
            ) : null}
            <div ref={bottomRef} />
          </div>

          {error ? (
            <p className="text-sm text-error" role="alert">
              {error}
            </p>
          ) : null}

          <form
            className="flex gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              void send(input);
            }}
          >
            <input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="Ask about your store…"
              disabled={busy}
              className="h-10 min-w-0 flex-1 rounded-md border border-border bg-background px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-brand/40"
            />
            <Button type="submit" size="sm" disabled={busy || !input.trim()}>
              Send
            </Button>
          </form>
        </div>
      </SheetContent>
    </Sheet>
  );
}
