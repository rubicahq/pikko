"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { apiFetch, ApiError } from "@/lib/api";
import { ROLE_LABELS } from "@/lib/rbac";
import { Button } from "@/components/ui/button";

type Peek = {
  valid: boolean;
  store_name?: string;
  role?: string;
  email?: string;
};

function AcceptInner() {
  const params = useSearchParams();
  const router = useRouter();
  const token = params.get("token") || "";
  const [peek, setPeek] = useState<Peek | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!token) {
        setPeek({ valid: false });
        return;
      }
      try {
        const data = await apiFetch<Peek>(
          `/api/v1/team/invites/peek?token=${encodeURIComponent(token)}`,
        );
        if (!cancelled) setPeek(data);
      } catch {
        if (!cancelled) setPeek({ valid: false });
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token]);

  async function accept() {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch<{ redirect_path: string }>("/api/v1/team/invites/accept", {
        body: { token },
      });
      router.replace(res.redirect_path || "/stores");
    } catch (err) {
      if (err instanceof ApiError && err.code === "invalid_invite") {
        router.replace("/auth/invalid");
        return;
      }
      setError(err instanceof ApiError ? err.message : "Unable to accept invite.");
      setLoading(false);
    }
  }

  if (peek === null) {
    return <p className="text-sm text-foreground-secondary">Checking your invite…</p>;
  }

  if (!peek.valid) {
    router.replace("/auth/invalid");
    return null;
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-4 py-12">
      <h1 className="text-[28px] font-medium leading-[1.12] tracking-[-0.02em] text-foreground md:text-4xl">
        Join {peek.store_name}.
      </h1>
      <p className="mt-3 text-base text-foreground-secondary">
        You&apos;re invited as <strong>{ROLE_LABELS[peek.role || ""] || peek.role}</strong>
        {peek.email ? (
          <>
            {" "}
            for <strong>{peek.email}</strong>
          </>
        ) : null}
        . Accept to sign in and open this store — without logging out of other stores later.
      </p>
      <Button className="mt-8 w-full" size="large" loading={loading} onClick={accept}>
        Accept invitation.
      </Button>
      {error ? <p className="mt-3 text-sm text-error">{error}</p> : null}
    </div>
  );
}

export default function InviteAcceptPage() {
  return (
    <Suspense fallback={<p className="p-8 text-sm text-foreground-secondary">Loading…</p>}>
      <AcceptInner />
    </Suspense>
  );
}
