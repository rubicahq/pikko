import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Reveal } from "@/components/marketing/reveal";
import { marketingMetadata } from "@/lib/marketing-seo";

export const metadata = marketingMetadata({
  title: "Solutions",
  description:
    "Pikko solutions for jastipers and social sellers — storefronts, preorders, and operations without marketplace lock-in.",
  path: "/solutions",
});

export default function SolutionsPage() {
  return (
    <div className="mx-auto max-w-marketing px-4 py-16 md:px-8 md:py-24">
      <Reveal>
        <p className="text-sm font-medium text-brand-dark">Solutions</p>
        <h1 className="mt-3 max-w-3xl font-display text-4xl font-medium tracking-[-0.03em] md:text-5xl">
          One product. Two beachheads. Your store either way.
        </h1>
        <p className="mt-5 max-w-reading text-lg leading-relaxed text-foreground-secondary">
          Pikko is not a marketplace and not an ERP. It is merchant-owned commerce
          shaped around how independent sellers already work.
        </p>
      </Reveal>

      <div className="mt-14 grid gap-8 md:grid-cols-2">
        <Reveal>
          <div className="flex h-full flex-col border-t-2 border-brand pt-8">
            <h2 className="font-display text-2xl font-medium tracking-[-0.02em]">
              Jastip / personal shoppers
            </h2>
            <p className="mt-4 flex-1 text-base leading-relaxed text-foreground-secondary">
              Launch preorder collections, take structured orders before you buy,
              consolidate purchasing, allocate what arrived, and keep every
              customer on one tracking thread.
            </p>
            <div className="mt-8">
              <Button asChild variant="outline">
                <Link href="/solutions/jastip">Explore jastip</Link>
              </Button>
            </div>
          </div>
        </Reveal>
        <Reveal delayMs={80}>
          <div className="flex h-full flex-col border-t-2 border-dark pt-8">
            <h2 className="font-display text-2xl font-medium tracking-[-0.02em]">
              Social sellers
            </h2>
            <p className="mt-4 flex-1 text-base leading-relaxed text-foreground-secondary">
              Give followers a real storefront to browse, checkout without a
              question marathon, and follow payment and fulfillment status on their
              own.
            </p>
            <div className="mt-8">
              <Button asChild variant="outline">
                <Link href="/solutions/social-sellers">Explore social selling</Link>
              </Button>
            </div>
          </div>
        </Reveal>
      </div>
    </div>
  );
}
