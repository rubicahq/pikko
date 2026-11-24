import { Reveal } from "@/components/marketing/reveal";
import { marketingMetadata } from "@/lib/marketing-seo";
import { appBaseDomain, supportEmail } from "@/lib/site";

export const metadata = marketingMetadata({
  title: "Terms",
  description:
    "Pikko terms of use. Merchant and customer responsibilities for the commerce platform.",
  path: "/terms",
});

export default function TermsPage() {
  const domain = appBaseDomain() === "localhost" ? "pikko.id" : appBaseDomain();
  const support = supportEmail();
  return (
    <article className="mx-auto max-w-reading px-4 py-16 md:px-8 md:py-24">
      <Reveal>
        <p className="text-sm font-medium text-brand-dark">Legal</p>
        <h1 className="mt-3 font-display text-4xl font-medium tracking-[-0.03em]">Terms</h1>
        <p className="mt-4 text-sm text-foreground-muted">
          Last updated: 24 November 2026 (Asia/Jakarta). These terms govern use of
          Pikko on {domain}. They are a working notice and may be replaced by a
          counsel-approved commercial contract when published.
        </p>
      </Reveal>

      <div className="mt-10 space-y-8 text-base leading-relaxed text-foreground-secondary">
        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            1. Service
          </h2>
          <p className="mt-3">
            Pikko provides merchant tools to publish a storefront, accept orders
            (ready stock and preorder), verify bank-transfer evidence, and manage
            fulfillment. The service runs on {domain} and related subdomains and
            may change as we learn from real use.
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            2. Eligibility and accounts
          </h2>
          <p className="mt-3">
            You must use a valid email you control. Authentication is passwordless
            via magic link. You are responsible for activity under your sessions and
            for keeping access to that email secure. Do not share magic links.
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            3. Merchant responsibilities
          </h2>
          <ul className="mt-3 list-disc space-y-2 pl-5">
            <li>Provide accurate store, product, pricing, and fulfillment information.</li>
            <li>Handle customer payments, refunds, and disputes for your own sales.</li>
            <li>Comply with applicable Indonesian consumer and privacy requirements.</li>
            <li>Do not upload unlawful, infringing, or harmful content or executables.</li>
          </ul>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            4. Customers
          </h2>
          <p className="mt-3">
            Customer checkout on a merchant storefront is a transaction between the
            customer and that merchant. Pikko is infrastructure, not the seller of
            catalog goods. Tracking links authorize access to that order only.
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            5. Acceptable use
          </h2>
          <p className="mt-3">
            No abuse of rate limits, scanning of auth tokens, cross-store probing,
            malware distribution, or attempts to circumvent tenant isolation. We may
            suspend accounts that threaten platform integrity.
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            6. Availability
          </h2>
          <p className="mt-3">
            The service is provided as-is during early access. Features may change;
            downtime may occur for maintenance. Commercial SLAs and paid plans are
            not part of this notice until published separately.
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            7. Limitation
          </h2>
          <p className="mt-3">
            To the extent permitted by law, Rubica HQ is not liable for indirect
            losses, lost profits, or merchant–customer disputes arising from sales on
            the platform. Aggregate liability for early-access use is limited to
            amounts paid to us for the service in the prior three months (currently
            zero for unpaid early access).
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            8. Contact
          </h2>
          <p className="mt-3">
            Questions:{" "}
            <a className="underline underline-offset-2" href={`mailto:${support}`}>
              {support}
            </a>
            . Privacy practices are described at /privacy.
          </p>
        </section>
      </div>
    </article>
  );
}
