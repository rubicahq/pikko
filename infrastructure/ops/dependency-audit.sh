#!/usr/bin/env bash
# Dependency audit for Go modules and npm packages. Fails on high/critical npm findings.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

echo "== go vet / module verify =="
(cd services/api && go mod verify && go vet ./...)

echo "== npm audit (apps/web) =="
(cd apps/web && npm audit --omit=dev --audit-level=high)

echo "[dependency-audit] OK"
