import { Reveal } from "@/components/marketing/reveal";
import { marketingMetadata } from "@/lib/marketing-seo";
import { appBaseDomain, supportEmail } from "@/lib/site";

export const metadata = marketingMetadata({
  title: "Privacy",
  description:
    "Pikko privacy notice for merchants and customers. How we collect, use, and protect data.",
  path: "/privacy",
});

export default function PrivacyPage() {
  const domain = appBaseDomain() === "localhost" ? "pikko.id" : appBaseDomain();
  const support = supportEmail();
  return (
    <article className="mx-auto max-w-reading px-4 py-16 md:px-8 md:py-24">
      <Reveal>
        <p className="text-sm font-medium text-brand-dark">Legal</p>
        <h1 className="mt-3 font-display text-4xl font-medium tracking-[-0.03em]">Privacy</h1>
        <p className="mt-4 text-sm text-foreground-muted">
          Last updated: 24 November 2026 (Asia/Jakarta). This notice describes
          technical practices for Pikko on {domain}. It is not a substitute for
          formal legal counsel or regulatory approval.
        </p>
      </Reveal>

      <div className="prose-pikko mt-10 space-y-8 text-base leading-relaxed text-foreground-secondary">
        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            Who we are
          </h2>
          <p className="mt-3">
            Pikko is operated by Rubica HQ for merchant commerce at{" "}
            <strong className="font-medium text-foreground">{domain}</strong>{" "}
            and related subdomains (app, shop, api). Contact:{" "}
            <a className="underline underline-offset-2" href={`mailto:${support}`}>
              {support}
            </a>
            .
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            Data we process
          </h2>
          <ul className="mt-3 list-disc space-y-2 pl-5">
            <li>
              <strong className="font-medium text-foreground">Merchants:</strong>{" "}
              email address, display name, session identifiers, store configuration,
              catalog, orders, payment method labels you configure, and operational
              notes.
            </li>
            <li>
              <strong className="font-medium text-foreground">Customers:</strong>{" "}
              name, contact details, shipping/pickup info, order contents, payment
              evidence files, and tracking tokens you share with them.
            </li>
            <li>
              <strong className="font-medium text-foreground">Technical:</strong>{" "}
              IP-derived rate-limit buckets, request IDs, and application logs needed
              for security and reliability. We do not put raw magic-link tokens or
              secrets in analytics.
            </li>
          </ul>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            Why we process it
          </h2>
          <p className="mt-3">
            To authenticate merchants, operate storefronts and checkouts, verify
            payments, fulfill orders, send transactional email, prevent abuse, and
            improve reliability. Marketing automation and promotional campaigns are
            out of MVP scope.
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            Storage and processors
          </h2>
          <p className="mt-3">
            Application data lives in PostgreSQL and object storage on our VPS (or
            configured S3-compatible storage). Transactional email may be delivered
            via Brevo when enabled. Access is limited to operators who need it for
            support and incident response.
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            Retention and rights
          </h2>
          <p className="mt-3">
            We retain transaction and audit records as needed for dispute handling
            and operational integrity. Merchants may request access, correction, or
            deletion of account data via {support}, subject to legal and
            fraud-prevention holds on order history. Store archival disables public
            browsing while preserving historical orders.
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            Security
          </h2>
          <p className="mt-3">
            Sessions use HttpOnly cookies; magic-link tokens are hashed and
            single-use; store-scoped authorization isolates tenant data; uploads are
            validated and stored privately unless published as storefront assets.
            Report suspected vulnerabilities to {support}.
          </p>
        </section>

        <section>
          <h2 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
            Changes
          </h2>
          <p className="mt-3">
            We may update this notice as the product evolves. Material changes will
            be reflected on this page with an updated date. Counsel-approved launch
            contracts may supersede this working notice when published.
          </p>
        </section>
      </div>
    </article>
  );
}
