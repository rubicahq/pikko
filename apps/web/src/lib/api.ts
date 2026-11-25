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

  return parseApiResponse<T>(res);
}

export async function apiUpload<T>(
  path: string,
  form: FormData,
  opts: { token?: string; method?: string } = {},
): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (opts.token) headers.Authorization = `Bearer ${opts.token}`;
  const res = await fetch(urls.api(path), {
    method: opts.method || "POST",
    headers,
    credentials: "include",
    body: form,
    cache: "no-store",
  });
  return parseApiResponse<T>(res);
}

async function parseApiResponse<T>(res: Response): Promise<T> {
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
  store_icon_file_id?: string | null;
  store_icon_url?: string;
  permissions?: string[];
};

export type OnboardingDraft = {
  id: string;
  step: number;
  store_name: string | null;
  store_slug: string | null;
  category: string | null;
  selling_model: string | null;
  plan_code?: string | null;
  billing_cycle?: string | null;
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

export type Theme = {
  id: string;
  name: string;
  background_color: string;
  text_color: string;
  primary_color: string;
  button_foreground: string;
  surface_color: string;
  border_color: string;
};

export type StorefrontConfig = {
  id: string;
  store_id: string;
  channel: "draft" | "published";
  theme_id: string;
  theme_name?: string;
  background_color: string;
  text_color: string;
  primary_color: string;
  button_foreground: string;
  surface_color: string;
  border_color: string;
  logo_file_id?: string | null;
  favicon_file_id?: string | null;
  store_icon_file_id?: string | null;
  logo_url?: string;
  favicon_url?: string;
  store_icon_url?: string;
  description?: string | null;
  contact_email?: string | null;
  whatsapp?: string | null;
  social_links?: Record<string, string>;
  show_catalogs_on_home: boolean;
  updated_at: string;
  published_at?: string | null;
};

export type Category = {
  id: string;
  store_id: string;
  name: string;
  slug: string;
  description?: string | null;
  sort_order: number;
};

/** Fixed 10-color palette for attribute options (API-enforced). */
export const ATTRIBUTE_OPTION_COLORS = [
  "coral",
  "peach",
  "amber",
  "lime",
  "teal",
  "sky",
  "indigo",
  "violet",
  "rose",
  "slate",
] as const;

export type AttributeOptionColor = (typeof ATTRIBUTE_OPTION_COLORS)[number];

export const ATTRIBUTE_OPTION_COLOR_HEX: Record<AttributeOptionColor, string> = {
  coral: "#FF3D00",
  peach: "#FF8050",
  amber: "#F59E0B",
  lime: "#84CC16",
  teal: "#14B8A6",
  sky: "#0EA5E9",
  indigo: "#6366F1",
  violet: "#8B5CF6",
  rose: "#F43F5E",
  slate: "#64748B",
};

export type AttributeType =
  | "single_option"
  | "multiple_option"
  | "short_text"
  | "number";

export type AttributeOption = {
  id: string;
  attribute_id: string;
  label: string;
  color: AttributeOptionColor | string;
  sort_order: number;
};

export type AttributeDefinition = {
  id: string;
  store_id: string;
  key: string;
  name: string;
  label: string;
  type: AttributeType | string;
  data_type: AttributeType | string;
  options: AttributeOption[];
  is_required: boolean;
  display_order: number;
};

export type ProductAttributeValue = {
  id: string;
  product_id: string;
  attribute_id: string;
  attribute_key?: string;
  attribute_name?: string;
  attribute_label?: string;
  type?: string;
  data_type?: string;
  value_text?: string | null;
  value_number?: number | null;
  option_ids?: string[];
};

export type ProductImage = {
  id: string;
  file_id: string;
  url?: string;
  alt_text?: string | null;
  display_order: number;
  is_primary: boolean;
};

export type Product = {
  id: string;
  store_id: string;
  name: string;
  slug: string;
  description?: string | null;
  sku?: string | null;
  brand?: string | null;
  category_id?: string | null;
  category_name?: string | null;
  base_price_idr: number;
  compare_at_price_idr?: number | null;
  contact_for_price?: boolean;
  ready_stock_eligible: boolean;
  preorder_eligible: boolean;
  availability: string;
  inventory_mode: string;
  stock_quantity?: number | null;
  status: "draft" | "published" | "unlisted" | "archived";
  images?: ProductImage[];
  primary_image_url?: string;
  created_at: string;
  updated_at: string;
};

export type StoredFile = {
  id: string;
  store_id: string;
  purpose: string;
  content_type: string;
  size_bytes: number;
  url?: string;
};

export function formatIDR(amount: number): string {
  return `Rp${formatIDRDigits(amount)}`;
}

/** Thousand-separated digits only (id-ID), e.g. 175000 → "175.000". */
export function formatIDRDigits(amount: number): string {
  return Math.trunc(amount).toLocaleString("id-ID");
}

/** Parse a typed IDR field (strips dots/spaces/Rp); empty → null. */
export function parseIDRInput(raw: string): number | null {
  const digits = raw.replace(/\D/g, "");
  if (!digits) return null;
  const n = Number.parseInt(digits, 10);
  return Number.isNaN(n) ? null : n;
}

/** Mask raw keystrokes into thousand-separated digits while typing. */
export function maskIDRInput(raw: string): string {
  const n = parseIDRInput(raw);
  return n == null ? "" : formatIDRDigits(n);
}
