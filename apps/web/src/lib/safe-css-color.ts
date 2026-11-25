/** Return a CSS hex color, or fallback when the value is missing/corrupt. */
export function safeCssColor(value: string | null | undefined, fallback: string): string {
  const v = (value || "").trim();
  if (/^#[0-9A-Fa-f]{6}$/.test(v)) return v;
  if (/^#[0-9A-Fa-f]{3}$/.test(v)) return v;
  return fallback;
}
