import Link from "next/link";
import { Reveal } from "@/components/marketing/reveal";
import { marketingMetadata } from "@/lib/marketing-seo";

export const metadata = marketingMetadata({
  title: "FAQ",
  description: "Common questions about Pikko — accounts, stores, preorders, payments, and pilots.",
  path: "/faq",
});

const faqs = [
  {
    q: "What is Pikko?",
    a: "Pikko is simple commerce infrastructure for independent sellers. Create a storefront, take ready-stock and preorder orders, verify payments, and fulfill without rebuilding every order from chat.",
  },
  {
    q: "Do I need a password?",
    a: "No. Sign in with a magic link sent to your email. The same flow works for new and returning merchants.",
  },
  {
    q: "Can I run more than one store?",
    a: "Yes. One verified email can own multiple stores. Each store has its own catalog, orders, customers, and settings.",
  },
  {
    q: "Does Pikko support jastip / preorder?",
    a: "Yes. Open preorder batches with deadlines and ETA, collect structured orders, consolidate purchasing, allocate arrivals, and keep customers on tracking links.",
  },
  {
    q: "How do customers pay?",
    a: "MVP checkout uses merchant bank transfer with payment evidence. Merchants verify or reject evidence from the dashboard. Payment gateways are post-MVP.",
  },
  {
    q: "Is this a marketplace?",
    a: "No. Pikko is merchant-owned commerce. Customers shop on your storefront link — not a shared marketplace feed.",
  },
  {
    q: "Where does Pikko run today?",
    a: "Controlled pilots run on pikko.site (app., shop., api.). Migration to pikko.id is a separate, explicitly approved launch step.",
  },
  {
    q: "How much does it cost?",
    a: "Commercial pricing is not finalized. See the pricing page for pilot framing — we do not invent plan prices ahead of launch decisions.",
  },
];

export default function FAQPage() {
  return (
    <div className="mx-auto max-w-marketing px-4 py-16 md:px-8 md:py-24">
      <Reveal>
        <p className="text-sm font-medium text-brand-dark">FAQ</p>
        <h1 className="mt-3 max-w-2xl font-display text-4xl font-medium tracking-[-0.03em] md:text-5xl">
          Answers before you open a store.
        </h1>
        <p className="mt-5 max-w-reading text-lg leading-relaxed text-foreground-secondary">
          Still stuck? Email{" "}
          <a className="text-foreground underline underline-offset-2" href="mailto:support@pikko.site">
            support@pikko.site
          </a>{" "}
          or{" "}
          <Link href="/auth" className="text-foreground underline underline-offset-2">
            create your store
          </Link>
          .
        </p>
      </Reveal>

      <div className="mt-14 max-w-reading space-y-0 border-t border-border">
        {faqs.map((item, i) => (
          <Reveal key={item.q} delayMs={Math.min(i * 40, 200)}>
            <details className="group border-b border-border py-6">
              <summary className="cursor-pointer list-none font-display text-lg font-medium tracking-[-0.02em] text-foreground marker:content-none [&::-webkit-details-marker]:hidden">
                <span className="flex items-start justify-between gap-4">
                  {item.q}
                  <span
                    aria-hidden
                    className="mt-1 text-foreground-muted transition-transform duration-hover group-open:rotate-45"
                  >
                    +
                  </span>
                </span>
              </summary>
              <p className="mt-3 text-base leading-relaxed text-foreground-secondary">{item.a}</p>
            </details>
          </Reveal>
        ))}
      </div>
    </div>
  );
}
