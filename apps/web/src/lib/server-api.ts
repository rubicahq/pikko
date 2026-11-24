import { cookies } from "next/headers";
import type { Account, OnboardingDraft, Session, Store, User } from "./api";

const API = process.env.API_BASE_URL || process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";
const COOKIE = process.env.AUTH_COOKIE_NAME || "pikko_session";

async function serverFetch<T>(path: string): Promise<T | null> {
  const jar = await cookies();
  const token = jar.get(COOKIE)?.value;
  if (!token) return null;
  const res = await fetch(`${API}${path}`, {
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${token}`,
      Cookie: `${COOKIE}=${token}`,
    },
    cache: "no-store",
  });
  if (!res.ok) return null;
  return (await res.json()) as T;
}

export async function getSession(): Promise<{ user: User; session: Session } | null> {
  return serverFetch("/api/v1/auth/session");
}

export async function getStores(includeArchived = false): Promise<Store[]> {
  const data = await serverFetch<{ stores: Store[] }>(
    `/api/v1/stores${includeArchived ? "?include_archived=true" : ""}`,
  );
  return data?.stores || [];
}

export async function getStore(storeId: string): Promise<Store | null> {
  const data = await serverFetch<{ store: Store }>(`/api/v1/stores/${storeId}`);
  return data?.store || null;
}

export async function getAccount(): Promise<Account | null> {
  const data = await serverFetch<{ account: Account }>("/api/v1/account");
  return data?.account || null;
}

export async function getOnboardingDraft(): Promise<OnboardingDraft | null> {
  const data = await serverFetch<{ draft: OnboardingDraft }>("/api/v1/onboarding/draft");
  return data?.draft || null;
}

export async function getCategories(): Promise<{ value: string; label: string }[]> {
  const data = await serverFetch<{ categories: { value: string; label: string }[] }>(
    "/api/v1/meta/categories",
  );
  return data?.categories || [];
}
