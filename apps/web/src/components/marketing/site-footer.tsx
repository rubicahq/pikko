import Link from "next/link";
import { Wordmark } from "@/components/brand/wordmark";

const columns = [
  {
    title: "Product",
    links: [
      { href: "/solutions", label: "Solutions" },
      { href: "/pricing", label: "Pricing" },
      { href: "/faq", label: "FAQ" },
    ],
  },
  {
    title: "Legal",
    links: [
      { href: "/privacy", label: "Privacy" },
      { href: "/terms", label: "Terms" },
    ],
  },
  {
    title: "Get started",
    links: [
      { href: "/auth", label: "Sign in" },
      { href: "/auth", label: "Create a store" },
    ],
  },
];

export function SiteFooter() {
  return (
    <footer className="border-t border-border bg-background-subtle">
      <div className="mx-auto grid max-w-marketing gap-10 px-4 py-16 md:grid-cols-[1.4fr_1fr_1fr_1fr] md:px-8">
        <div>
          <Wordmark className="font-display text-xl tracking-[-0.03em]" />
          <p className="mt-4 max-w-xs text-sm leading-relaxed text-foreground-secondary">
            Sell your way. Commerce infrastructure for independent sellers —
            ready stock, preorder, and operations that stay simple.
          </p>
        </div>
        {columns.map((col) => (
          <div key={col.title}>
            <p className="text-xs font-medium uppercase tracking-[0.08em] text-foreground-muted">
              {col.title}
            </p>
            <ul className="mt-4 space-y-3">
              {col.links.map((link) => (
                <li key={`${col.title}-${link.label}`}>
                  <Link
                    href={link.href}
                    className="text-sm text-foreground-secondary transition-colors duration-hover hover:text-foreground"
                  >
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      <div className="border-t border-border">
        <div className="mx-auto flex max-w-marketing flex-col gap-2 px-4 py-6 text-sm text-foreground-muted md:flex-row md:items-center md:justify-between md:px-8">
          <span>Pikko. Sell your way.</span>
          <span>© {new Date().getFullYear()} Rubica HQ · Controlled pilots on pikko.site</span>
        </div>
      </div>
    </footer>
  );
}
