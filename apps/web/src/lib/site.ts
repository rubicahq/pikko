/**
 * Domain-agnostic site identity (PRD §7).
 * Absolute hosts come only from env — switch pikko.site → pikko.id by rebuild/redeploy config.
 */

function trimSlash(s: string): string {
  return s.replace(/\/$/, "");
}

function hostFromURL(raw: string | undefined, fallbackHost: string): string {
  if (!raw) return fallbackHost;
  try {
    const u = new URL(raw.includes("://") ? raw : `https://${raw}`);
    return u.host;
  } catch {
    return raw.replace(/^https?:\/\//, "").replace(/\/$/, "") || fallbackHost;
  }
}

/** Canonical public base domain (apex), e.g. pikko.id */
export function appBaseDomain(): string {
  const explicit =
    process.env.NEXT_PUBLIC_APP_BASE_DOMAIN || process.env.APP_BASE_DOMAIN;
  if (explicit) return explicit.replace(/^\.+/, "").toLowerCase();
  const marketing = process.env.NEXT_PUBLIC_MARKETING_URL || process.env.APP_URL;
  if (marketing) {
    try {
      return new URL(marketing).hostname.replace(/^www\./, "").toLowerCase();
    } catch {
      /* fall through */
    }
  }
  return "localhost";
}

/** Marketing / APP_URL absolute origin */
export function marketingOrigin(): string {
  const fromEnv =
    process.env.NEXT_PUBLIC_MARKETING_URL ||
    process.env.APP_URL ||
    process.env.MARKETING_URL;
  if (fromEnv) return trimSlash(fromEnv);
  const domain = appBaseDomain();
  if (domain === "localhost") return "http://localhost:3000";
  return `https://${domain}`;
}

export function dashboardOrigin(): string {
  const fromEnv =
    process.env.NEXT_PUBLIC_DASHBOARD_URL || process.env.DASHBOARD_URL;
  if (fromEnv) return trimSlash(fromEnv);
  const domain = appBaseDomain();
  if (domain === "localhost") return "http://localhost:3000";
  return `https://app.${domain}`;
}

export function storefrontOrigin(): string {
  const fromEnv =
    process.env.NEXT_PUBLIC_STOREFRONT_URL || process.env.STOREFRONT_URL;
  if (fromEnv) return trimSlash(fromEnv);
  const domain = appBaseDomain();
  if (domain === "localhost") return "http://localhost:3000";
  return `https://shop.${domain}`;
}

/** Tim Pikko HQ BO origin (hq.pikko.site) — not the merchant dashboard. */
export function hqOrigin(): string {
  const fromEnv = process.env.NEXT_PUBLIC_HQ_URL || process.env.HQ_URL;
  if (fromEnv) return trimSlash(fromEnv);
  const domain = appBaseDomain();
  if (domain === "localhost") return "http://localhost:3000";
  return `https://hq.${domain}`;
}

export function isHQHost(host: string): boolean {
  const h = host.split(":")[0].toLowerCase();
  try {
    const hq = new URL(hqOrigin()).hostname.toLowerCase();
    if (h === hq) return true;
  } catch {
    /* fall through */
  }
  const domain = appBaseDomain();
  return domain !== "localhost" && h === `hq.${domain}`;
}

/** Host label shown in onboarding slug preview (no scheme). */
export function storefrontHost(): string {
  return hostFromURL(
    process.env.NEXT_PUBLIC_STOREFRONT_URL || process.env.STOREFRONT_URL,
    appBaseDomain() === "localhost" ? "localhost:3000" : `shop.${appBaseDomain()}`,
  );
}

export function supportEmail(): string {
  return (
    process.env.NEXT_PUBLIC_SUPPORT_EMAIL ||
    process.env.EMAIL_REPLY_TO ||
    `support@${appBaseDomain() === "localhost" ? "pikko.site" : appBaseDomain()}`
  );
}

/** Prior domain that should 301 to APP_BASE_DOMAIN (cutover). Empty = no app-level redirect. */
export function legacyBaseDomain(): string {
  return (
    process.env.NEXT_PUBLIC_LEGACY_BASE_DOMAIN ||
    process.env.LEGACY_BASE_DOMAIN ||
    ""
  )
    .replace(/^\.+/, "")
    .toLowerCase();
}

/**
 * Map a request Host on the legacy apex/subdomains to the new domain host.
 * e.g. app.pikko.site → app.pikko.id when legacy=pikko.site and base=pikko.id
 */
export function mapLegacyHost(host: string): string | null {
  const legacy = legacyBaseDomain();
  const next = appBaseDomain();
  if (!legacy || !next || legacy === next) return null;
  const h = host.split(":")[0].toLowerCase();
  if (h === legacy) return next;
  if (h.endsWith(`.${legacy}`)) {
    const sub = h.slice(0, -(legacy.length + 1));
    return sub ? `${sub}.${next}` : next;
  }
  return null;
}
