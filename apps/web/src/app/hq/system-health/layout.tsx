"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { HQShell } from "@/components/hq/hq-shell";
import { ApiError, hqFetch } from "@/lib/hq-api";

type SessionUser = { email: string };

export default function SystemHealthLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [email, setEmail] = useState<string | undefined>();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await hqFetch<{ user: SessionUser }>("/auth/session");
        if (!cancelled) {
          setEmail(res.user.email);
          setReady(true);
        }
      } catch (err) {
        if (!cancelled) {
          if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
            router.replace("/auth");
            return;
          }
          router.replace("/auth");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [router]);

  if (!ready) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background-subtle text-sm text-foreground-muted">
        Loading HQ…
      </div>
    );
  }

  return <HQShell email={email}>{children}</HQShell>;
}
