#!/usr/bin/env bash
# Non-destructive restore validation: restore a dump into a temporary Postgres
# container, run a smoke query, then tear down. Never touches pikko-postgres data.
set -euo pipefail

DUMP="${1:-}"
if [[ -z "$DUMP" || ! -f "$DUMP" ]]; then
  echo "Usage: $0 /path/to/postgres.dump" >&2
  exit 2
fi

NAME="pikko-restore-test-$$"
PASS="restore-test-$$"
PORT=55432

cleanup() {
  docker rm -f "$NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "[restore-test] starting ephemeral postgres ${NAME}"
docker run -d --name "$NAME" \
  -e POSTGRES_PASSWORD="$PASS" \
  -e POSTGRES_USER=pikko \
  -e POSTGRES_DB=pikko \
  -p "127.0.0.1:${PORT}:5432" \
  postgres:16-alpine >/dev/null

for i in $(seq 1 30); do
  if docker exec "$NAME" pg_isready -U pikko -d pikko >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

docker exec -i "$NAME" pg_restore -U pikko -d pikko --clean --if-exists < "$DUMP" \
  || echo "[restore-test] pg_restore reported warnings (common with --clean on empty DB)"

COUNT="$(docker exec "$NAME" psql -U pikko -d pikko -Atc "SELECT count(*) FROM schema_migrations;")"
echo "[restore-test] schema_migrations rows=${COUNT}"
if [[ -z "$COUNT" || "$COUNT" -lt 1 ]]; then
  echo "[restore-test] FAIL: migrations table empty or missing" >&2
  exit 1
fi

echo "[restore-test] OK — dump restores and schema_migrations is readable"
