/** Indonesian mobile / WhatsApp norms aligned with API `validate.NormalizeIndonesianPhone`. */

export function normalizeIndonesianPhone(raw: string): string | null {
  const digits = String(raw || "").replace(/\D/g, "");
  if (!digits) return null;
  let n = digits;
  if (n.startsWith("0")) n = "62" + n.slice(1);
  else if (n.startsWith("8") && n.length >= 9 && n.length <= 13) n = "62" + n;
  if (!n.startsWith("62")) return null;
  const national = n.slice(2);
  if (!national.startsWith("8")) return null;
  if (national.length < 9 || national.length > 13) return null;
  return n;
}

export function isValidIndonesianPhone(raw: string): boolean {
  return normalizeIndonesianPhone(raw) !== null;
}
