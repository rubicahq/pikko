import type { Metadata } from "next";
import { Wordmark } from "@/components/brand/wordmark";
import { redirectIfAuthenticated } from "@/lib/auth-redirect";

export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

export default async function AuthLayout({ children }: { children: React.ReactNode }) {
  // Skip magic-link UI when the session cookie is still valid (cross-subdomain).
  await redirectIfAuthenticated();

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="px-4 py-5 md:px-8">
        <Wordmark href="/" />
      </header>
      <main className="flex flex-1 items-start justify-center px-4 pb-16 pt-8 md:items-center md:pt-0">
        <div className="w-full max-w-auth">{children}</div>
      </main>
    </div>
  );
}
