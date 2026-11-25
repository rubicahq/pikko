#!/usr/bin/env bash
# Fast post-deploy smoke for Pikko (local loopback + optional public URLs).
# Does not touch Emplicon. Exit 0 only when required checks pass.
#
# Domain-agnostic (E2E-N): public URLs come from env only.
# Defaults match current production (pikko.site). Never hardcodes pikko.id.
# Cutover: set MARKETING_URL / DASHBOARD_URL / STOREFRONT_URL / API_BASE_URL
# (or SMOKE_PUBLIC_*) to the new hosts — do not flip DNS from this script.
set -euo pipefail

API_URL="${SMOKE_API_URL:-http://127.0.0.1:8100}"
WEB_URL="${SMOKE_WEB_URL:-http://127.0.0.1:3100}"
PUBLIC="${SMOKE_PUBLIC:-0}"
TIMEOUT_SEC="${SMOKE_TIMEOUT_SEC:-45}"
CHECK_READYZ="${SMOKE_READYZ:-1}"

# Public surfaces — env-only; defaults = active production domain (pikko.site).
PUBLIC_API_URL="${SMOKE_PUBLIC_API_URL:-${API_BASE_URL:-https://api.pikko.site}}"
PUBLIC_MARKETING_URL="${SMOKE_PUBLIC_MARKETING_URL:-${MARKETING_URL:-https://pikko.site}}"
PUBLIC_APP_URL="${SMOKE_PUBLIC_APP_URL:-${DASHBOARD_URL:-https://app.pikko.site}}"
PUBLIC_SHOP_URL="${SMOKE_PUBLIC_SHOP_URL:-${STOREFRONT_URL:-https://shop.pikko.site}}"
PUBLIC_HQ_URL="${SMOKE_PUBLIC_HQ_URL:-${HQ_URL:-https://hq.pikko.site}}"

need() {
  local url="$1" expect="${2:-200}"
  local code
  code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 "$url" || echo 000)"
  if [[ "$code" != "$expect" ]]; then
    echo "[smoke] FAIL $url → HTTP $code (want $expect)" >&2
    return 1
  fi
  echo "[smoke] ok  $url → $code"
}

wait_need() {
  local url="$1" expect="${2:-200}"
  local deadline=$((SECONDS + TIMEOUT_SEC))
  while (( SECONDS < deadline )); do
    if need "$url" "$expect"; then
      return 0
    fi
    sleep 1
  done
  echo "[smoke] FAIL $url — timed out after ${TIMEOUT_SEC}s" >&2
  return 1
}

echo "[smoke] local loopback (api=${API_URL} web=${WEB_URL})"
wait_need "${API_URL}/healthz" 200
wait_need "${API_URL}/livez" 200
if [[ "$CHECK_READYZ" == "1" ]]; then
  wait_need "${API_URL}/readyz" 200
fi
wait_need "${WEB_URL}/" 200

if [[ "$PUBLIC" == "1" ]]; then
  echo "[smoke] public surfaces (env-driven; not an automatic domain cutover)"
  need "${PUBLIC_API_URL%/}/healthz" 200
  need "${PUBLIC_API_URL%/}/livez" 200
  need "${PUBLIC_MARKETING_URL%/}/" 200
  need "${PUBLIC_APP_URL%/}/" 200
  # Shop root may 200 or redirect; accept 200/301/302/308.
  shop_code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 "${PUBLIC_SHOP_URL%/}/" || echo 000)"
  case "$shop_code" in
    200|301|302|308) echo "[smoke] ok  ${PUBLIC_SHOP_URL%/}/ → $shop_code" ;;
    *)
      echo "[smoke] FAIL ${PUBLIC_SHOP_URL%/}/ → HTTP $shop_code" >&2
      exit 1
      ;;
  esac
  # HQ BO — auth gate may redirect to /auth; accept 200/301/302/308.
  hq_code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 "${PUBLIC_HQ_URL%/}/" || echo 000)"
  case "$hq_code" in
    200|301|302|308) echo "[smoke] ok  ${PUBLIC_HQ_URL%/}/ → $hq_code" ;;
    *)
      echo "[smoke] FAIL ${PUBLIC_HQ_URL%/}/ → HTTP $hq_code" >&2
      exit 1
      ;;
  esac
fi

echo "[smoke] passed"
