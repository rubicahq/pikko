#!/usr/bin/env bash
# Light load smoke against health + public marketing/API surfaces.
# Not a full capacity test — catches gross regressions before pilot traffic.
set -euo pipefail

BASE="${1:-https://pikko.site}"
API="${2:-https://api.pikko.site}"
N="${LOAD_SMOKE_REQUESTS:-40}"
CONCURRENCY="${LOAD_SMOKE_CONCURRENCY:-8}"

echo "[load-smoke] base=${BASE} api=${API} n=${N} c=${CONCURRENCY}"

fail=0
check() {
  local url="$1"
  local code
  code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 15 "$url" || echo 000)"
  echo "  ${code}  ${url}"
  if [[ "$code" != "200" && "$code" != "301" && "$code" != "302" ]]; then
    fail=1
  fi
}

check "${API}/livez"
check "${API}/healthz"
check "${API}/readyz"
check "${BASE}/"
check "${BASE}/pricing"
check "${BASE}/faq"
check "${BASE}/privacy"
check "${BASE}/terms"
check "${BASE}/robots.txt"
check "${BASE}/sitemap.xml"

echo "[load-smoke] concurrent health probes"
seq 1 "$N" | xargs -P "$CONCURRENCY" -I{} curl -sS -o /dev/null -w '%{http_code}\n' --max-time 10 "${API}/healthz" \
  | sort | uniq -c

if [[ "$fail" -ne 0 ]]; then
  echo "[load-smoke] FAIL: one or more endpoints returned unexpected status" >&2
  exit 1
fi
echo "[load-smoke] OK"
