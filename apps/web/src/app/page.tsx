import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function MarketingHomePage() {
  return (
    <div className="min-h-screen bg-background">
      <header className="mx-auto flex max-w-marketing items-center justify-between px-4 py-5 md:px-8">
        <Link href="/" className="text-lg font-medium tracking-tight text-foreground">
          Pikko
        </Link>
        <div className="flex items-center gap-3">
          <Button asChild variant="ghost">
            <Link href="/auth">Sign in</Link>
          </Button>
          <Button asChild>
            <Link href="/auth">Get started</Link>
          </Button>
        </div>
      </header>

      <main>
        <section className="mx-auto max-w-marketing px-4 pb-24 pt-16 md:px-8 md:pt-24">
          <p className="text-sm font-medium text-foreground-secondary">
            Commerce for independent sellers.
          </p>
          <h1 className="mt-4 max-w-3xl text-[42px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground md:text-[56px]">
            Your business. Your store. Your way.
          </h1>
          <p className="mt-5 max-w-xl text-lg leading-relaxed text-foreground-secondary">
            Create a beautiful online store, make shopping easier for your customers, and manage
            orders the way your business works.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Button asChild size="large">
              <Link href="/auth">Create your store.</Link>
            </Button>
            <Button asChild variant="outline" size="large">
              <Link href="#pillars">Explore Pikko.</Link>
            </Button>
          </div>
        </section>

        <section id="pillars" className="border-t border-border bg-background-subtle">
          <div className="mx-auto grid max-w-marketing gap-12 px-4 py-20 md:grid-cols-3 md:px-8">
            {[
              {
                title: "Beautiful storefronts",
                body: "Themes, branding, and product browsing that feel like your brand — not a spreadsheet.",
              },
              {
                title: "Flexible ordering",
                body: "Ready stock and preorder batches with deadlines, ETA, and handling fees that match how you sell.",
              },
              {
                title: "Effortless operations",
                body: "Structured orders, payment verification, and customer records without re-entering chats.",
              },
            ].map((item) => (
              <div key={item.title}>
                <h2 className="text-xl font-medium text-foreground">{item.title}</h2>
                <p className="mt-3 text-sm leading-relaxed text-foreground-secondary">{item.body}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="bg-dark text-foreground-inverse">
          <div className="mx-auto max-w-marketing px-4 py-20 md:px-8">
            <h2 className="max-w-2xl text-3xl font-medium tracking-[-0.02em] md:text-4xl">
              A better way to sell starts here.
            </h2>
            <div className="mt-8">
              <Button asChild size="large" variant="primary">
                <Link href="/auth">Create your store.</Link>
              </Button>
            </div>
          </div>
        </section>
      </main>

      <footer className="border-t border-border">
        <div className="mx-auto flex max-w-marketing flex-col gap-2 px-4 py-8 text-sm text-foreground-secondary md:flex-row md:justify-between md:px-8">
          <span>Pikko. Sell your way.</span>
          <span>© {new Date().getFullYear()} Rubica HQ</span>
        </div>
      </footer>
    </div>
  );
}
