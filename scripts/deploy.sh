#!/usr/bin/env bash
# Idempotent Pikko VPS deploy. Never touches Emplicon containers/volumes.
#
# Always uses:
#   docker compose -p pikko -f docker-compose.yml -f docker-compose.deploy.yml
# so host ports 127.0.0.1:8100 / :3100 stay published.
#
# Hardening (Agent 15):
#   - BuildKit + parallel selective rebuild
#   - pg_dump backup before migration-bearing deploys (and with --backup)
#   - wait for postgres healthy before migrate/backup
#   - pre-* image tags + rollback on smoke/health failure
#   - domain-agnostic public smoke via env (never auto-cutover to pikko.id)
#
# Usage:
#   ./scripts/deploy.sh                 # pull main, selective rebuild, smoke
#   ./scripts/deploy.sh --no-pull       # deploy current checkout
#   ./scripts/deploy.sh --force-all     # rebuild api+web regardless of diff
#   ./scripts/deploy.sh --backup        # force DB backup even without migrations
#   ./scripts/deploy.sh --skip-smoke    # skip smoke (default runs fast local smoke)
#   ./scripts/deploy.sh --public-smoke  # also hit public HTTPS endpoints (env URLs)
#   ./scripts/deploy.sh --dry-run       # print plan only
#   ./scripts/deploy.sh --ref=main      # deploy a specific ref (default: main)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

REF="${DEPLOY_REF:-main}"
DO_PULL=1
FORCE_ALL=0
FORCE_BACKUP=0
RUN_SMOKE=1
PUBLIC_SMOKE=0
DRY_RUN=0
PROFILE_PROXY=1
HEALTH_TIMEOUT_SEC="${DEPLOY_HEALTH_TIMEOUT_SEC:-90}"

for arg in "$@"; do
  case "$arg" in
    --no-pull) DO_PULL=0 ;;
    --force-all) FORCE_ALL=1 ;;
    --backup) FORCE_BACKUP=1 ;;
    --skip-smoke) RUN_SMOKE=0 ;;
    --public-smoke) PUBLIC_SMOKE=1 ;;
    --dry-run) DRY_RUN=1 ;;
    --no-proxy) PROFILE_PROXY=0 ;;
    --ref=*) REF="${arg#--ref=}" ;;
    -h|--help)
      sed -n '2,28p' "$0"
      exit 0
      ;;
    *)
      echo "Unknown arg: $arg" >&2
      exit 2
      ;;
  esac
done

# BuildKit (cache mounts in Dockerfiles rely on this)
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1
export BUILDKIT_PROGRESS="${BUILDKIT_PROGRESS:-plain}"

if docker info >/dev/null 2>&1; then
  DOCKER=(docker)
else
  DOCKER=(sudo docker)
fi

COMPOSE=(
  "${DOCKER[@]}" compose
  -p pikko
  -f docker-compose.yml
  -f docker-compose.deploy.yml
)

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
LOG_PREFIX="[pikko-deploy ${STAMP}]"
TIMINGS=()
BACKUP_DIR="${BACKUP_DIR:-$HOME/backups/pikko}"
# Deploy-time dumps live under deploy/ so cron directory backups stay separate.
DEPLOY_BACKUP_DIR="${DEPLOY_BACKUP_DIR:-$BACKUP_DIR/deploy}"

log() { echo "${LOG_PREFIX} $*"; }
time_step() {
  local name="$1"
  shift
  local start end elapsed rc
  start=$(date +%s)
  log "→ ${name}"
  if (( DRY_RUN )); then
    log "  (dry-run) $*"
    TIMINGS+=("${name}=0s")
    return 0
  fi
  set +e
  "$@"
  rc=$?
  set -e
  end=$(date +%s)
  elapsed=$((end - start))
  if (( rc != 0 )); then
    TIMINGS+=("${name}=${elapsed}s(fail)")
    log "← ${name} FAILED after ${elapsed}s (exit ${rc})"
    return "$rc"
  fi
  TIMINGS+=("${name}=${elapsed}s")
  log "← ${name} ${elapsed}s"
}

ensure_emplicon_untouched() {
  # Refuse destructive Emplicon project ops; this script never names emplicon
  # as a compose project. Network attach is additive-only (see connect_emplicon_network).
  if [[ "$*" == *emplicon* ]]; then
    echo "Refusing command that mentions emplicon" >&2
    exit 3
  fi
}

preflight() {
  [[ -f docker-compose.yml ]] || { echo "missing docker-compose.yml" >&2; exit 1; }
  [[ -f docker-compose.deploy.yml ]] || { echo "missing docker-compose.deploy.yml" >&2; exit 1; }
  [[ -f scripts/smoke.sh ]] || { echo "missing scripts/smoke.sh" >&2; exit 1; }
  if [[ ! -f .env ]] && (( ! DRY_RUN )); then
    log "WARN: no .env in $ROOT — compose may fail on required secrets (POSTGRES_PASSWORD)"
  fi
  local head_mig
  head_mig="$(ls -1 database/migrations/*_*.up.sql 2>/dev/null | sed 's|.*/||' | sort | tail -1 || true)"
  log "migration tree tip file: ${head_mig:-none}"
}

tag_rollback_images() {
  if (( DRY_RUN )); then return 0; fi
  if "${DOCKER[@]}" image inspect pikko-api:latest >/dev/null 2>&1; then
    "${DOCKER[@]}" tag pikko-api:latest "pikko-api:pre-${STAMP}"
  fi
  if "${DOCKER[@]}" image inspect pikko-web:latest >/dev/null 2>&1; then
    "${DOCKER[@]}" tag pikko-web:latest "pikko-web:pre-${STAMP}"
  fi
  log "rollback tags: pikko-api:pre-${STAMP} pikko-web:pre-${STAMP}"
}

rollback() {
  log "HEALTH CHECK FAILED — rolling back images"
  if "${DOCKER[@]}" image inspect "pikko-api:pre-${STAMP}" >/dev/null 2>&1; then
    "${DOCKER[@]}" tag "pikko-api:pre-${STAMP}" pikko-api:latest
  fi
  if "${DOCKER[@]}" image inspect "pikko-web:pre-${STAMP}" >/dev/null 2>&1; then
    "${DOCKER[@]}" tag "pikko-web:pre-${STAMP}" pikko-web:latest
  fi
  local up_args=(up -d --no-build)
  if (( PROFILE_PROXY )); then
    up_args=(--profile proxy "${up_args[@]}")
  fi
  "${COMPOSE[@]}" "${up_args[@]}" api web || true
  connect_emplicon_network || true
  log "rollback attempted with pre-${STAMP} tags"
}

connect_emplicon_network() {
  # Additive only — attach Pikko containers to Emplicon nginx network if present.
  if ! "${DOCKER[@]}" network inspect emplicon_emplicon >/dev/null 2>&1; then
    log "emplicon_emplicon network not found — skip attach"
    return 0
  fi
  "${DOCKER[@]}" network connect emplicon_emplicon pikko-web 2>/dev/null || true
  "${DOCKER[@]}" network connect emplicon_emplicon pikko-api 2>/dev/null || true
}

paths_changed() {
  local from="$1" to="$2"
  # Empty "from" (first run / unknown) → rebuild everything.
  # Identical SHAs → empty diff (no rebuild) unless --force-all.
  if [[ -z "$from" ]]; then
    echo "ALL"
    return 0
  fi
  if [[ "$from" == "$to" ]]; then
    echo ""
    return 0
  fi
  git diff --name-only "$from" "$to"
}

plan_services() {
  local diff="$1"
  NEED_API=0
  NEED_WEB=0
  NEED_MIGRATE=0
  if (( FORCE_ALL )) || [[ "$diff" == "ALL" ]]; then
    NEED_API=1
    NEED_WEB=1
    NEED_MIGRATE=1
    return 0
  fi
  while IFS= read -r f; do
    [[ -z "$f" ]] && continue
    case "$f" in
      database/migrations/*)
        NEED_MIGRATE=1
        NEED_API=1
        ;;
      services/api/*|Dockerfile.api|database/*)
        NEED_API=1
        ;;
      apps/web/*|packages/*|Dockerfile.web)
        NEED_WEB=1
        ;;
      docker-compose.yml|docker-compose.deploy.yml|.dockerignore|scripts/deploy.sh|scripts/smoke.sh)
        NEED_API=1
        NEED_WEB=1
        ;;
    esac
  done <<< "$diff"
}

wait_postgres_healthy() {
  local deadline=$((SECONDS + HEALTH_TIMEOUT_SEC))
  local st
  if ! "${DOCKER[@]}" inspect pikko-postgres >/dev/null 2>&1; then
    log "postgres container missing — will start via compose"
    return 0
  fi
  while (( SECONDS < deadline )); do
    st="$("${DOCKER[@]}" inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' pikko-postgres 2>/dev/null || echo missing)"
    if [[ "$st" == "healthy" || "$st" == "running" ]]; then
      # Prefer pg_isready when healthy/running
      if "${DOCKER[@]}" exec pikko-postgres pg_isready -U "${POSTGRES_USER:-pikko}" -d "${POSTGRES_DB:-pikko}" >/dev/null 2>&1; then
        log "postgres ready ($st)"
        return 0
      fi
    fi
    sleep 2
  done
  log "ERROR: postgres not ready within ${HEALTH_TIMEOUT_SEC}s (last=$st)"
  return 1
}

backup_db() {
  mkdir -p "$DEPLOY_BACKUP_DIR"
  chmod 700 "$BACKUP_DIR" 2>/dev/null || true
  chmod 700 "$DEPLOY_BACKUP_DIR" 2>/dev/null || true
  local dest="${DEPLOY_BACKUP_DIR}/pikko-${STAMP}.dump"
  if ! "${DOCKER[@]}" inspect pikko-postgres >/dev/null 2>&1; then
    if (( NEED_MIGRATE )); then
      log "ERROR: postgres missing but migrations planned — cannot backup"
      return 1
    fi
    log "postgres container missing — skip backup"
    return 0
  fi
  wait_postgres_healthy
  umask 077
  "${DOCKER[@]}" exec pikko-postgres \
    pg_dump -U "${POSTGRES_USER:-pikko}" -d "${POSTGRES_DB:-pikko}" -Fc \
    > "$dest"
  if [[ ! -s "$dest" ]]; then
    log "ERROR: backup empty at $dest"
    rm -f "$dest"
    return 1
  fi
  # Keep a small SHA beside the dump for restore verification.
  (cd "$(dirname "$dest")" && sha256sum "$(basename "$dest")" > "pikko-${STAMP}.sha256")
  log "backup → $dest ($(du -h "$dest" | awk '{print $1}'))"
}

log_schema_version() {
  if (( DRY_RUN )); then return 0; fi
  if ! "${DOCKER[@]}" inspect pikko-postgres >/dev/null 2>&1; then
    return 0
  fi
  local ver
  ver="$("${DOCKER[@]}" exec pikko-postgres \
    psql -U "${POSTGRES_USER:-pikko}" -d "${POSTGRES_DB:-pikko}" -Atc \
    "SELECT COALESCE(MAX(version),0) FROM schema_migrations WHERE dirty = false;" 2>/dev/null || echo "?")"
  log "schema_migrations head (clean) = ${ver}"
}

wait_local_health() {
  local deadline=$((SECONDS + HEALTH_TIMEOUT_SEC))
  local code
  while (( SECONDS < deadline )); do
    code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 3 http://127.0.0.1:8100/healthz 2>/dev/null || echo 000)"
    if [[ "$code" == "200" ]]; then
      log "api healthz ready"
      return 0
    fi
    sleep 2
  done
  log "ERROR: api healthz not 200 within ${HEALTH_TIMEOUT_SEC}s (last=$code)"
  return 1
}

# --- main ---
ensure_emplicon_untouched
preflight
log "root=$ROOT ref=$REF buildkit=${DOCKER_BUILDKIT} dry_run=${DRY_RUN}"

OLD_SHA="$(git rev-parse HEAD)"
if (( DO_PULL )); then
  time_step "git_fetch_checkout" bash -c "
    git fetch origin '${REF}'
    git checkout '${REF}'
    git pull --ff-only origin '${REF}'
  "
fi
NEW_SHA="$(git rev-parse HEAD)"
log "sha ${OLD_SHA:0:7} → ${NEW_SHA:0:7}"
# Surface measured deploy version in HQ System Health (never invent when unset).
export DEPLOY_VERSION="${DEPLOY_VERSION:-$NEW_SHA}"
log "DEPLOY_VERSION=${DEPLOY_VERSION:0:12}"

DIFF="$(paths_changed "$OLD_SHA" "$NEW_SHA")"
plan_services "$DIFF"
log "plan api=${NEED_API} web=${NEED_WEB} migrate=${NEED_MIGRATE} force_all=${FORCE_ALL}"

# Ensure postgres is up before backup/migrate when containers already exist.
if (( ! DRY_RUN )); then
  if "${DOCKER[@]}" inspect pikko-postgres >/dev/null 2>&1; then
    time_step "wait_postgres" wait_postgres_healthy
  fi
fi

if (( NEED_MIGRATE || FORCE_BACKUP )); then
  time_step "backup_db" backup_db
else
  log "skip backup (no new migrations; pass --backup to force)"
  TIMINGS+=("backup_db=skipped")
fi

time_step "tag_rollback" tag_rollback_images

BUILD_SERVICES=()
if (( NEED_API || NEED_MIGRATE )); then
  # Compose tags migrate separately (pikko-migrate); must rebuild it or
  # `migrate up` can run a stale image missing new SQL files.
  BUILD_SERVICES+=(api migrate)
fi
if (( NEED_WEB )); then BUILD_SERVICES+=(web); fi

if (( ${#BUILD_SERVICES[@]} > 0 )); then
  time_step "build_parallel" "${COMPOSE[@]}" build --parallel "${BUILD_SERVICES[@]}"
else
  log "no image rebuild needed"
  TIMINGS+=("build_parallel=skipped")
fi

UP_SERVICES=()
if (( NEED_API || NEED_MIGRATE )); then UP_SERVICES+=(api); fi
if (( NEED_WEB )); then UP_SERVICES+=(web); fi

up_deploy() {
  local args=(up -d --no-build)
  if (( PROFILE_PROXY )); then
    args=(--profile proxy "${args[@]}")
  fi
  # Always ensure postgres is healthy; recreate only app services that changed.
  if (( ${#UP_SERVICES[@]} == 0 )); then
    "${COMPOSE[@]}" "${args[@]}"
  else
    "${COMPOSE[@]}" "${args[@]}" postgres
    wait_postgres_healthy
    if (( NEED_API || NEED_MIGRATE )); then
      # migrate is one-shot; compose waits for service_completed_successfully before api.
      "${COMPOSE[@]}" "${args[@]}" migrate api
    fi
    if (( NEED_WEB )); then
      "${COMPOSE[@]}" "${args[@]}" --no-deps web
    fi
  fi
  connect_emplicon_network
  log_schema_version
}

if (( ${#BUILD_SERVICES[@]} > 0 || NEED_MIGRATE )); then
  time_step "compose_up" up_deploy
else
  log "skip compose up (nothing to restart); ensuring network attach"
  if (( ! DRY_RUN )); then
    connect_emplicon_network
    log_schema_version
  fi
  TIMINGS+=("compose_up=skipped")
fi

# Verify published ports when deploy overlay is in use
if (( ! DRY_RUN )); then
  ports="$("${DOCKER[@]}" ps --filter name=pikko-api --filter name=pikko-web --format '{{.Names}} {{.Ports}}')"
  log "ports: $(echo "$ports" | tr '\n' ' | ')"
  if ! echo "$ports" | grep -q '8100'; then
    log "ERROR: api host port 8100 missing — deploy overlay not applied?"
    rollback
    exit 1
  fi
  if ! echo "$ports" | grep -q '3100'; then
    log "ERROR: web host port 3100 missing — deploy overlay not applied?"
    rollback
    exit 1
  fi
  time_step "wait_healthz" wait_local_health || { rollback; exit 1; }
fi

load_surface_urls_from_env() {
  # Domain-agnostic (E2E-N): pull only surface URL keys from .env for public smoke.
  # Never flips DNS or rewrites production hosts to pikko.id.
  local f="$ROOT/.env" line key val
  [[ -f "$f" ]] || return 0
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "${line// }" ]] && continue
    key="${line%%=*}"
    val="${line#*=}"
    case "$key" in
      MARKETING_URL|DASHBOARD_URL|STOREFRONT_URL|HQ_URL|API_BASE_URL|APP_BASE_DOMAIN)
        val="${val%$'\r'}"
        if [[ "$val" == \"*\" ]]; then val="${val:1:${#val}-2}"; fi
        if [[ "$val" == \'*\' ]]; then val="${val:1:${#val}-2}"; fi
        export "${key}=${val}"
        ;;
    esac
  done < "$f"
}

smoke() {
  if (( PUBLIC_SMOKE )); then
    export SMOKE_PUBLIC=1
  fi
  load_surface_urls_from_env
  bash "$ROOT/scripts/smoke.sh"
}

if (( RUN_SMOKE )); then
  if ! time_step "smoke" smoke; then
    rollback
    exit 1
  fi
else
  TIMINGS+=("smoke=skipped")
fi

log "done sha=${NEW_SHA:0:7}"
log "timings: ${TIMINGS[*]}"
log "note: pikko.id cutover is owner-gated — see docs/LAUNCH.md (this script never flips DNS/env to pikko.id)"
