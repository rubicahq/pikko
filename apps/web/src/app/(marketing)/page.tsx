import Image from "next/image";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Reveal } from "@/components/marketing/reveal";
import { redirectIfAuthenticated } from "@/lib/auth-redirect";
import { marketingMetadata } from "@/lib/marketing-seo";

export const metadata = marketingMetadata({
  title: "Pikko",
  path: "/",
});

export default async function MarketingHomePage() {
  await redirectIfAuthenticated();

  return (
    <>
      {/* Hero — brand-first, one composition, no cards/stats/icon grids */}
      <section className="relative overflow-hidden border-b border-border">
        <div
          aria-hidden
          className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_80%_60%_at_70%_-10%,var(--brand-soft),transparent_55%),linear-gradient(180deg,var(--background)_0%,var(--background-subtle)_100%)]"
        />
        <div
          aria-hidden
          className="pointer-events-none absolute inset-y-0 right-0 hidden w-1/2 bg-[linear-gradient(135deg,transparent_40%,var(--surface)_40%,var(--surface)_41%,transparent_41%),linear-gradient(225deg,transparent_55%,var(--brand-border)_55%,var(--brand-border)_56%,transparent_56%)] opacity-40 md:block"
        />
        <div className="relative mx-auto grid max-w-marketing gap-12 px-4 pb-20 pt-16 md:grid-cols-[1.15fr_0.85fr] md:items-end md:px-8 md:pb-28 md:pt-24">
          <div>
            <Image
              src="/brand/pikko-logo.png"
              alt="Pikko"
              width={220}
              height={89}
              className="h-10 w-auto md:h-12"
              priority
            />
            <h1 className="mt-6 max-w-3xl font-display text-[42px] font-medium leading-[1.02] tracking-[-0.03em] text-foreground md:text-[64px]">
              Sell your way.
            </h1>
            <p className="mt-5 max-w-xl text-lg leading-relaxed text-foreground-secondary">
              Create a beautiful online store, give customers a clearer shopping
              experience, and manage orders the way your business already works.
            </p>
            <div className="mt-9 flex flex-wrap gap-3">
              <Button asChild size="large">
                <Link href="/auth">Create your store.</Link>
              </Button>
              <Button asChild variant="outline" size="large">
                <Link href="/solutions">See how it fits.</Link>
              </Button>
            </div>
          </div>
          <Reveal className="relative min-h-[280px] md:min-h-[360px]" delayMs={80}>
            <div className="absolute inset-0 rounded-2xl border border-border bg-dark p-6 text-foreground-inverse shadow-md md:p-8">
              <p className="text-xs font-medium uppercase tracking-[0.12em] text-foreground-muted">
                Storefront
              </p>
              <p className="mt-6 font-display text-3xl font-medium tracking-[-0.02em] md:text-4xl">
                Your brand.
                <br />
                Your link.
                <br />
                Your rules.
              </p>
              <p className="mt-6 max-w-sm text-sm leading-relaxed text-foreground-subtle">
                Ready stock and preorder batches on one shareable storefront —
                without rebuilding every order from chat.
              </p>
              <Image
                src="/brand/pikko-icon.png"
                alt=""
                width={64}
                height={64}
                aria-hidden
                className="absolute bottom-6 right-6 h-16 w-16 md:bottom-8 md:right-8"
              />
            </div>
          </Reveal>
        </div>
      </section>

      <section className="mx-auto max-w-marketing px-4 py-20 md:px-8 md:py-28">
        <Reveal>
          <h2 className="max-w-2xl font-display text-3xl font-medium tracking-[-0.02em] text-foreground md:text-4xl">
            Built for how independent sellers actually sell.
          </h2>
          <p className="mt-4 max-w-reading text-base leading-relaxed text-foreground-secondary">
            Product info lives in posts, PDFs, and chats. Pikko turns that into a
            storefront customers can browse, with structured checkout and
            merchant tools that stay out of the way.
          </p>
        </Reveal>
        <div className="mt-14 grid gap-10 border-t border-border pt-14 md:grid-cols-3 md:gap-12">
          {[
            {
              title: "Beautiful storefronts",
              body: "Themes, branding, and product browsing that feel like your brand — not a spreadsheet pasted into a template.",
            },
            {
              title: "Flexible ordering",
              body: "Ready stock and preorder batches with deadlines, ETA, handling fees, and approval when your flow needs it.",
            },
            {
              title: "Effortless operations",
              body: "Structured orders, payment verification, purchasing lists, and customer records without re-entering chats.",
            },
          ].map((item, i) => (
            <Reveal key={item.title} delayMs={i * 60}>
              <h3 className="font-display text-xl font-medium tracking-[-0.02em] text-foreground">
                {item.title}
              </h3>
              <p className="mt-3 text-sm leading-relaxed text-foreground-secondary">{item.body}</p>
            </Reveal>
          ))}
        </div>
      </section>

      <section className="border-y border-border bg-background-subtle">
        <div className="mx-auto grid max-w-marketing gap-12 px-4 py-20 md:grid-cols-2 md:px-8 md:py-28">
          <Reveal>
            <p className="text-sm font-medium text-brand-dark">For jastipers</p>
            <h2 className="mt-3 font-display text-3xl font-medium tracking-[-0.02em] md:text-[36px]">
              Collect preorders. Buy once. Fulfill with clarity.
            </h2>
            <p className="mt-4 text-base leading-relaxed text-foreground-secondary">
              Open a batch, gather paid orders, consolidate purchasing, allocate
              what arrived, and keep every customer on the same tracking link.
            </p>
            <div className="mt-8">
              <Button asChild variant="outline">
                <Link href="/solutions/jastip">Jastip solution</Link>
              </Button>
            </div>
          </Reveal>
          <Reveal delayMs={80}>
            <p className="text-sm font-medium text-brand-dark">For social sellers</p>
            <h2 className="mt-3 font-display text-3xl font-medium tracking-[-0.02em] md:text-[36px]">
              A real store for customers who already trust you.
            </h2>
            <p className="mt-4 text-base leading-relaxed text-foreground-secondary">
              Publish catalog pages customers can browse on their own, checkout
              without a thread of questions, and track payment and fulfillment
              without chasing you.
            </p>
            <div className="mt-8">
              <Button asChild variant="outline">
                <Link href="/solutions/social-sellers">Social seller solution</Link>
              </Button>
            </div>
          </Reveal>
        </div>
      </section>

      <section className="bg-dark text-foreground-inverse">
        <div className="mx-auto max-w-marketing px-4 py-20 md:px-8 md:py-28">
          <Reveal>
            <h2 className="max-w-2xl font-display text-3xl font-medium tracking-[-0.02em] md:text-5xl md:leading-[1.05]">
              A better way to sell starts here.
            </h2>
            <p className="mt-5 max-w-lg text-base leading-relaxed text-foreground-subtle">
              Passwordless sign-in. First store in two short steps. Built for
              independent sellers who want a store that stays simple.
            </p>
            <div className="mt-9">
              <Button asChild size="large">
                <Link href="/auth">Create your store.</Link>
              </Button>
            </div>
          </Reveal>
        </div>
      </section>
    </>
  );
}
