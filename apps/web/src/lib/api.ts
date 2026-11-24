import { urls } from "./urls";

export class ApiError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

type Options = {
  method?: string;
  body?: unknown;
  token?: string;
};

export async function apiFetch<T>(path: string, opts: Options = {}): Promise<T> {
  const headers: Record<string, string> = {
    Accept: "application/json",
  };
  if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (opts.token) {
    headers.Authorization = `Bearer ${opts.token}`;
  }

  const res = await fetch(urls.api(path), {
    method: opts.method || (opts.body !== undefined ? "POST" : "GET"),
    headers,
    credentials: "include",
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    cache: "no-store",
  });

  const text = await res.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = { raw: text };
    }
  }

  if (!res.ok) {
    const err = data as { error?: { code?: string; message?: string } } | null;
    throw new ApiError(
      res.status,
      err?.error?.code || "error",
      err?.error?.message || res.statusText,
    );
  }
  return data as T;
}

export type User = {
  id: string;
  email: string;
  display_name: string | null;
  locale: string;
  last_store_id: string | null;
  created_at: string;
};

export type Store = {
  id: string;
  name: string;
  slug: string;
  category: string;
  selling_model: "ready_stock" | "preorder" | "both";
  currency: string;
  timezone: string;
  locale: string;
  theme: string;
  publication_status: string;
  checkout_enabled: boolean;
  archived_at?: string | null;
  created_at: string;
  updated_at: string;
  role?: string;
  storefront_url?: string;
};

export type OnboardingDraft = {
  id: string;
  step: number;
  store_name: string | null;
  store_slug: string | null;
  category: string | null;
  selling_model: string | null;
  is_first_store: boolean;
  updated_at: string;
};

export type Account = {
  id: string;
  email: string;
  display_name: string | null;
  locale: string;
  language: string;
  timezone: string;
  last_store_id: string | null;
  created_at: string;
};

export type Session = {
  id: string;
  user_id: string;
  expires_at: string;
  created_at: string;
  last_seen_at: string;
  user_agent?: string | null;
  current?: boolean;
};
