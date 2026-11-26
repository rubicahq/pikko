"use client";

import { useEffect, useRef, useState } from "react";
import { Loader2, Sparkles } from "lucide-react";
import { apiFetch, ApiError } from "@/lib/api";
import { Description as DialogDescription } from "@radix-ui/react-dialog";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import Link from "next/link";

type PaymentCheck = {
  order_id: string;
  order_number: string;
  order_url: string;
  pikko_status: string;
  doku_status?: string;
  checked_at: string;
  verification: string;
  explanation: string;
};

type ChatMessage = {
  role: "user" | "assistant";
  content: string;
  paymentCheck?: PaymentCheck;
};

type ChatResponse = {
  message: { role: string; content: string };
  tool_results?: Array<{ name: string; ok: boolean; result?: PaymentCheck }>;
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
  const [error, setError] = useState<string | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

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
          paymentCheck: data.tool_results?.find((r) => r.name === "check_doku_payment" && r.ok)?.result,
        },
      ]);
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
          aria-label="Ask Connie AI"
          title="Ask Connie AI"
        >
          <Sparkles className="h-4 w-4 shrink-0 text-brand" strokeWidth={1.75} />
          <span className="truncate">Ask Connie AI</span>
        </button>
      </SheetTrigger>
      <SheetContent
        side="right"
        className="h-dvh w-[min(100%,28rem)]"
      >
        <SheetHeader className="shrink-0">
          <SheetTitle className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-brand" strokeWidth={1.75} />
            Ask Connie
          </SheetTitle>
          <DialogDescription className="mt-1 text-sm text-foreground-secondary">
            Tanya Connie tentang pesanan, produk, atau operasional toko.
          </DialogDescription>
        </SheetHeader>

        <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
          <div className="flex shrink-0 flex-wrap gap-1.5 px-4 pt-4">
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

          <div className="min-h-0 flex-1 space-y-3 overflow-y-auto px-4 pb-3">
            {messages.map((m, i) => (
              <div
                key={`${m.role}-${i}`}
                className={cn(
                  "rounded-lg px-3 py-2 text-sm break-words whitespace-pre-wrap",
                  m.role === "user"
                    ? "ml-6 bg-ink text-foreground-inverse"
                    : "mr-2 bg-warm-feature text-foreground",
                )}
              >
                {m.content}
                {m.paymentCheck ? (
                  <div className="mt-3 rounded-md border border-border bg-background p-3 whitespace-normal">
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-medium">{m.paymentCheck.order_number}</span>
                      <span className={cn("rounded-full px-2 py-0.5 text-xs", m.paymentCheck.verification === "confirmed" ? "bg-success/10 text-success" : "bg-warning/10 text-warning")}>{m.paymentCheck.verification}</span>
                    </div>
                    <dl className="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 text-xs">
                      <dt className="text-foreground-muted">Pikko</dt><dd>{m.paymentCheck.pikko_status}</dd>
                      <dt className="text-foreground-muted">DOKU</dt><dd>{m.paymentCheck.doku_status || "Belum terverifikasi"}</dd>
                      <dt className="text-foreground-muted">Dicek</dt><dd>{new Date(m.paymentCheck.checked_at).toLocaleString("id-ID")}</dd>
                    </dl>
                    <p className="mt-2 text-xs text-foreground-secondary">{m.paymentCheck.explanation}</p>
                    <Link href={m.paymentCheck.order_url} className="mt-2 inline-block text-xs font-medium text-brand-accessible-dark">Buka detail order</Link>
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
            <p className="px-4 text-sm text-error" role="alert">
              {error}
            </p>
          ) : null}

          <form
            className="flex shrink-0 items-center gap-2 border-t border-border px-4 pt-3 pb-[max(1rem,env(safe-area-inset-bottom))]"
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
