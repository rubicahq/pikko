import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Reveal } from "@/components/marketing/reveal";
import { marketingMetadata } from "@/lib/marketing-seo";

export const metadata = marketingMetadata({
  title: "Social seller solution",
  description:
    "Pikko for social sellers — a branded storefront, structured checkout, and order tracking that reduce repetitive DMs.",
  path: "/solutions/social-sellers",
});

export default function SocialSellersSolutionPage() {
  return (
    <div className="mx-auto max-w-marketing px-4 py-16 md:px-8 md:py-24">
      <Reveal>
        <p className="text-sm font-medium text-brand-dark">Solutions · Social sellers</p>
        <h1 className="mt-3 max-w-3xl font-display text-4xl font-medium tracking-[-0.03em] md:text-5xl">
          Let customers shop when you are offline.
        </h1>
        <p className="mt-5 max-w-reading text-lg leading-relaxed text-foreground-secondary">
          Publish a storefront that matches your brand, keep prices and stock in one
          place, and replace “berapa ya?” threads with checkout, payment evidence,
          and tracking links.
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

      <div className="mt-16 grid gap-12 border-t border-border pt-14 md:grid-cols-2">
        <Reveal>
          <h2 className="font-display text-2xl font-medium tracking-[-0.02em]">
            What changes for your customers
          </h2>
          <p className="mt-4 text-base leading-relaxed text-foreground-secondary">
            They browse products and collections on a shareable link, understand
            availability before asking, checkout as a guest, and track payment and
            fulfillment without waiting for a reply.
          </p>
        </Reveal>
        <Reveal delayMs={80}>
          <h2 className="font-display text-2xl font-medium tracking-[-0.02em]">
            What changes for you
          </h2>
          <p className="mt-4 text-base leading-relaxed text-foreground-secondary">
            Orders arrive structured. Payment evidence lands in one queue. Customer
            records stay with the store. Ready stock and optional preorder can live
            side by side when you need both.
          </p>
        </Reveal>
      </div>
    </div>
  );
}
