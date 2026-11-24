#!/usr/bin/env bash
# Scheduled PostgreSQL + object-storage backup for Pikko (Phase 6).
# Intended for the VPS cron as ubuntu. Never deletes production volumes.
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-$HOME/backups/pikko}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-14}"
POSTGRES_CONTAINER="${BACKUP_POSTGRES_CONTAINER:-pikko-postgres}"
POSTGRES_USER="${POSTGRES_USER:-pikko}"
POSTGRES_DB="${POSTGRES_DB:-pikko}"
STORAGE_VOLUME="${BACKUP_STORAGE_VOLUME:-pikko_api_storage}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DEST="${BACKUP_DIR}/${STAMP}"

# Prefer unprivileged docker; fall back to sudo (VPS ubuntu often lacks docker group).
if docker info >/dev/null 2>&1; then
  DOCKER=(docker)
else
  DOCKER=(sudo docker)
fi

mkdir -p "$DEST"
umask 077

echo "[pikko-backup] starting ${STAMP}"

if ! "${DOCKER[@]}" inspect "$POSTGRES_CONTAINER" >/dev/null 2>&1; then
  echo "[pikko-backup] ERROR: container ${POSTGRES_CONTAINER} not found" >&2
  exit 1
fi

"${DOCKER[@]}" exec "$POSTGRES_CONTAINER" \
  pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc \
  > "${DEST}/postgres.dump"

# Local object storage volume (Compose volume). Skip quietly if absent.
# Mountpoints under /var/lib/docker are often root-only — use sudo for -d/tar.
VOL_PATH="$("${DOCKER[@]}" volume inspect -f '{{ .Mountpoint }}' "$STORAGE_VOLUME" 2>/dev/null || true)"
if [[ -n "${VOL_PATH}" ]] && sudo test -d "${VOL_PATH}"; then
  sudo tar -C "$(dirname "$VOL_PATH")" -czf "${DEST}/storage.tgz" "$(basename "$VOL_PATH")"
  echo "[pikko-backup] storage archived from ${VOL_PATH}"
else
  echo "[pikko-backup] storage volume ${STORAGE_VOLUME} not found — DB-only backup"
fi

sha256sum "${DEST}/postgres.dump" > "${DEST}/SHA256SUMS"
if [[ -f "${DEST}/storage.tgz" ]]; then
  sha256sum "${DEST}/storage.tgz" >> "${DEST}/SHA256SUMS"
fi

# Prune old backups (directory names are UTC stamps).
if [[ "${RETENTION_DAYS}" =~ ^[0-9]+$ ]] && [[ "${RETENTION_DAYS}" -gt 0 ]]; then
  find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d -mtime "+${RETENTION_DAYS}" -print -exec rm -rf {} +
fi

echo "[pikko-backup] complete → ${DEST}"
ls -lah "$DEST"
