import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Reveal } from "@/components/marketing/reveal";
import { marketingMetadata } from "@/lib/marketing-seo";

export const metadata = marketingMetadata({
  title: "Jastip solution",
  description:
    "Pikko for jastipers — preorder batches, consolidated purchasing, allocation, and customer tracking without spreadsheet chaos.",
  path: "/solutions/jastip",
});

export default function JastipSolutionPage() {
  return (
    <div className="mx-auto max-w-marketing px-4 py-16 md:px-8 md:py-24">
      <Reveal>
        <p className="text-sm font-medium text-brand-dark">Solutions · Jastip</p>
        <h1 className="mt-3 max-w-3xl font-display text-4xl font-medium tracking-[-0.03em] md:text-5xl">
          Preorder periods that feel as clear as your WhatsApp — without living in it.
        </h1>
        <p className="mt-5 max-w-reading text-lg leading-relaxed text-foreground-secondary">
          Open a batch, share one storefront link, collect paid orders, buy once,
          allocate arrivals, and fulfill per customer. Handling fees, minimums, and
          approval rules stay on the backend — not in your head.
        </p>
        <div className="mt-8 flex flex-wrap gap-3">
          <Button asChild size="large">
            <Link href="/auth">Create your store.</Link>
          </Button>
          <Button asChild variant="outline" size="large">
            <Link href="/solutions">All solutions</Link>
          </Button>
        </div>
      </Reveal>

      <ol className="mt-16 grid gap-10 border-t border-border pt-14 md:grid-cols-3">
        {[
          {
            step: "01",
            title: "Open the batch",
            body: "Set deadlines, ETA, products, and quotas. Publish when you are ready to collect.",
          },
          {
            step: "02",
            title: "Collect structured orders",
            body: "Customers checkout with variants and fees applied. You verify bank-transfer evidence.",
          },
          {
            step: "03",
            title: "Purchase and allocate",
            body: "Consolidate what to buy, mark purchased or unavailable, allocate, then fulfill.",
          },
        ].map((item, i) => (
          <Reveal key={item.step} delayMs={i * 60}>
            <p className="text-xs font-medium tracking-[0.12em] text-foreground-muted">{item.step}</p>
            <h2 className="mt-3 font-display text-xl font-medium tracking-[-0.02em]">{item.title}</h2>
            <p className="mt-3 text-sm leading-relaxed text-foreground-secondary">{item.body}</p>
          </Reveal>
        ))}
      </ol>
    </div>
  );
}
