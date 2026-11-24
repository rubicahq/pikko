import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Reveal } from "@/components/marketing/reveal";
import { marketingMetadata } from "@/lib/marketing-seo";
import { appBaseDomain, supportEmail } from "@/lib/site";

const domainLabel = () =>
  appBaseDomain() === "localhost" ? "pikko.id" : appBaseDomain();

export const metadata = marketingMetadata({
  title: "Pricing",
  description:
    "Pikko pricing. Commercial plans are not finalized — join early access to start selling.",
  path: "/pricing",
});

export default function PricingPage() {
  const domain = domainLabel();
  const support = supportEmail();
  return (
    <div className="mx-auto max-w-marketing px-4 py-16 md:px-8 md:py-24">
      <Reveal>
        <p className="text-sm font-medium text-brand-dark">Pricing</p>
        <h1 className="mt-3 max-w-3xl font-display text-4xl font-medium tracking-[-0.03em] text-foreground md:text-5xl">
          Start early. Pricing follows when plans are final.
        </h1>
        <p className="mt-5 max-w-reading text-lg leading-relaxed text-foreground-secondary">
          Subscription tiers, store limits, and AI import commercial limits are
          still open business decisions. On{" "}
          <span className="text-foreground">{domain}</span>, invited merchants
          get full MVP access so we can learn from real selling — not invent
          prices ahead of product–market fit.
        </p>
      </Reveal>

      <div className="mt-14 grid gap-8 border-t border-border pt-14 md:grid-cols-[1.2fr_0.8fr]">
        <Reveal>
          <h2 className="font-display text-2xl font-medium tracking-[-0.02em]">
            What early access includes
          </h2>
          <ul className="mt-6 space-y-4 text-base leading-relaxed text-foreground-secondary">
            <li className="border-l-2 border-brand pl-4">
              Passwordless merchant account and multi-store ownership
            </li>
            <li className="border-l-2 border-brand pl-4">
              Published storefront with themes, catalog, and preorder batches
            </li>
            <li className="border-l-2 border-brand pl-4">
              Checkout, bank-transfer evidence, tracking, and fulfillment tools
            </li>
            <li className="border-l-2 border-brand pl-4">
              Notifications, analytics overview, and CSV / PDF catalog import
            </li>
          </ul>
          <p className="mt-8 text-sm leading-relaxed text-foreground-muted">
            Official commercial pricing will be published on {domain} after
            explicit approval. Nothing here is a quote or a paid plan claim.
          </p>
        </Reveal>
        <Reveal delayMs={80}>
          <div className="rounded-xl border border-border bg-background-subtle p-8">
            <p className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
              Early access
            </p>
            <p className="mt-3 text-sm leading-relaxed text-foreground-secondary">
              Access by invitation / early access signup. No payment gateway
              subscription billing in MVP.
            </p>
            <div className="mt-8">
              <Button asChild size="large" className="w-full">
                <Link href="/auth">Request access.</Link>
              </Button>
            </div>
            <p className="mt-4 text-center text-xs text-foreground-muted">
              Questions?{" "}
              <a className="underline underline-offset-2" href={`mailto:${support}`}>
                {support}
              </a>
            </p>
          </div>
        </Reveal>
      </div>
    </div>
  );
}
