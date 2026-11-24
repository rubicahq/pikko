#!/usr/bin/env bash
# Fast post-deploy smoke for Pikko (local loopback + optional public URLs).
# Does not touch Emplicon. Exit 0 only when required checks pass.
set -euo pipefail

API_URL="${SMOKE_API_URL:-http://127.0.0.1:8100}"
WEB_URL="${SMOKE_WEB_URL:-http://127.0.0.1:3100}"
PUBLIC="${SMOKE_PUBLIC:-0}"
TIMEOUT_SEC="${SMOKE_TIMEOUT_SEC:-45}"

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
  return 1
}

echo "[smoke] local loopback"
wait_need "${API_URL}/healthz" 200
wait_need "${API_URL}/livez" 200
wait_need "${WEB_URL}/" 200

if [[ "$PUBLIC" == "1" ]]; then
  echo "[smoke] public surfaces"
  need "https://api.pikko.site/healthz" 200
  need "https://pikko.site/" 200
  need "https://app.pikko.site/" 200
fi

echo "[smoke] passed"
