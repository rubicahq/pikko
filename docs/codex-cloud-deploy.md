# Codex Cloud → GitHub Actions production deploy

This guide wires **Codex Cloud / automation** to trigger Pikko production deploys on the VPS via GitHub Actions — without storing personal interactive SSH keys in CI.

**Production was not deployed by the PR that added this workflow.** First use the dry-run path only.

| Item | Value |
| --- | --- |
| Repo | `rubicahq/pikko` |
| Workflow | `.github/workflows/deploy-production.yml` |
| Environment | `production` |
| VPS | `ubuntu@43.163.99.95` port `2021` |
| Checkout | `~/pikko` |
| Live site | `https://pikko.site` |
| Deploy command | `./scripts/deploy.sh --public-smoke` |

## Design choices

1. **Dispatch-first.** Only `workflow_dispatch` triggers the workflow (no auto-deploy on push to `main`). Codex / operators run it explicitly.
2. **Dry-run by default.** `dry_run` defaults to `true`; production `deploy.sh` runs only when `confirm_deploy=true`.
3. **Single git source of truth.** `scripts/deploy.sh` pulls `origin/<ref>` itself (`DO_PULL=1`). Actions does **not** pass `--no-pull`. Use `--no-pull` only for manual VPS sessions already at tip.
4. **Host key verification.** SSH uses `StrictHostKeyChecking=yes` and `DEPLOY_SSH_KNOWN_HOSTS` (from `ssh-keyscan`).
5. **Concurrency.** Group `pikko-production-deploy` — one deploy at a time (`cancel-in-progress: false`).
6. **Repo guard.** Job refuses to run unless `GITHUB_REPOSITORY == rubicahq/pikko`.

## Secrets (GitHub Environment `production`)

Create Environment **production**, then add these **Environment secrets** (not repository secrets preferred, so Environment protection applies):

| Secret | Example / notes |
| --- | --- |
| `DEPLOY_SSH_PRIVATE_KEY` | Full PEM of a **dedicated** deploy keypair (not a personal laptop key) |
| `DEPLOY_SSH_KNOWN_HOSTS` | Output of `ssh-keyscan -p 2021 43.163.99.95` (all lines) |
| `DEPLOY_SSH_HOST` | `43.163.99.95` |
| `DEPLOY_SSH_PORT` | `2021` |
| `DEPLOY_SSH_USER` | `ubuntu` |

Dry-run fails gracefully with a Step Summary hint if any secret is missing (values are never printed).

### Generate deploy keypair (owner machine)

```bash
# Dedicated identity — do not reuse personal interactive keys
ssh-keygen -t ed25519 -f ./pikko-deploy-ed25519 -C "pikko-gha-deploy" -N ""

# Private key → GitHub Environment secret DEPLOY_SSH_PRIVATE_KEY
# Public key  → append ONE line to VPS ~/.ssh/authorized_keys (ubuntu)
cat ./pikko-deploy-ed25519.pub
```

On the VPS (as `ubuntu`):

```bash
mkdir -p ~/.ssh && chmod 700 ~/.ssh
# Append the .pub line only (do not overwrite existing keys)
echo 'ssh-ed25519 AAAA... pikko-gha-deploy' >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

This doc does **not** claim the public key was installed; that is an owner action.

### Capture known_hosts

```bash
ssh-keyscan -p 2021 43.163.99.95
# Paste the full multi-line output into DEPLOY_SSH_KNOWN_HOSTS
```

Prefer pinning the scan you captured yourself. Re-run after host key rotation.

## GitHub Environment setup (UI)

1. Open [https://github.com/rubicahq/pikko/settings/environments](https://github.com/rubicahq/pikko/settings/environments)
2. **New environment** → name: `production`
3. **Environment protection rules**
   - Enable **Required reviewers** (add owner account(s)) so `confirm_deploy=true` waits for approval
   - Optionally restrict to selected branches (e.g. `main` only)
4. **Environment secrets** → add the five secrets above
5. Save

Both `dry-run` and `deploy` jobs use `environment: production`. With required reviewers, even dry-run waits for approval unless you later split environments (e.g. `production-dry-run` vs `production`). Start with one Environment; tighten if dry-run approval is too noisy.

## Workflow inputs

| Input | Default | Meaning |
| --- | --- | --- |
| `ref` | `main` | Passed to `deploy.sh --ref=…` on the VPS |
| `dry_run` | `true` | Connectivity job only (always runs) |
| `confirm_deploy` | `false` | When `true`, runs `deploy` after successful dry-run |

## Commands Codex Cloud can run later

Prerequisite: `gh` authenticated as an identity with Actions write + Contents as needed; secrets + deploy pubkey already installed; firewall allows GitHub Actions egress (or a self-hosted runner).

### 1) Dry-run only (safe)

```bash
gh workflow run deploy-production.yml \
  --repo rubicahq/pikko \
  --ref main \
  -f dry_run=true \
  -f confirm_deploy=false \
  -f ref=main

# Wait for the latest run of this workflow
gh run list --repo rubicahq/pikko --workflow=deploy-production.yml --limit 1
gh run watch --repo rubicahq/pikko $(gh run list --repo rubicahq/pikko --workflow=deploy-production.yml --limit 1 --json databaseId -q '.[0].databaseId')

# Logs + job summary
gh run view --repo rubicahq/pikko --log-failed
gh run view --repo rubicahq/pikko --json displayTitle,conclusion,url,headSha
```

### 2) Real production deploy (owner-approved)

```bash
gh workflow run deploy-production.yml \
  --repo rubicahq/pikko \
  --ref main \
  -f dry_run=false \
  -f confirm_deploy=true \
  -f ref=main

# If Environment has required reviewers, approve in the GitHub UI:
# Actions → run → Review deployments → Approve
gh run watch --repo rubicahq/pikko <run-id>
gh run view --repo rubicahq/pikko <run-id> --json conclusion,url
```

Expect the Step Summary to show deployed SHA and smoke outcome.

## Permissions audit

### Cursor Cloud Agent (this automation session)

| Check | Result |
| --- | --- |
| GitHub identity | `rubicahq` (user account connected to Cursor) |
| `rubicahq/pikko` permissions | `admin`, `maintain`, `push`, `pull` — **can push branches and open PRs** |
| PR for this change | Opened from branch `cursor/codex-cloud-deploy-actions-4029` |
| SSH from agent VM to `:2021` | TCP reaches sshd; **Permission denied (publickey)** without deploy key (expected). Sibling Codex “Connection refused” may be a different egress IP / firewall allowlist — see VPS diag. |

### What Codex Cloud needs (minimum)

| Capability | Why | Where to grant |
| --- | --- | --- |
| **Contents: Read and write** | Push branches / open PRs with workflow + docs | Repo **Settings → Collaborators** (user) or org **Member privileges** / GitHub App permissions |
| **Actions: Read and write** | `gh workflow run`, read run status/logs | Repo **Settings → Actions → General** (Actions permissions) + token/App scopes |
| **Workflows** (PAT fine-grained / App) | Editing `.github/workflows/*` | Fine-grained PAT: Repository permissions → **Workflows: Read and write**; classic PAT: `workflow` scope |
| **Environments** | Secrets + required reviewers | Repo **Settings → Environments → production** (admin) |
| **GitHub App / Cursor / Codex approval** | If org or account requires approving third-party apps | User: [https://github.com/settings/applications](https://github.com/settings/applications) · Org (if moved under an org): **Organization settings → Third-party access** / **GitHub Apps** · Cursor dashboard connection for GitHub |

Personal account `rubicahq` currently owns the repo (not an org). If the repo later moves under an organization, also check:

- **Organization settings → Member privileges** (base permissions)
- **Organization settings → Third-party application access policy**
- **Organization settings → Installed GitHub Apps** (approve Cursor / Codex)

### `gh` auth scopes (classic PAT reference)

For a human or Codex-held classic PAT used only for deploy automation:

- `repo` (private) or public-repo write as needed
- `workflow` (to push workflow file changes)
- Actions dispatch works with Actions API access via `repo` on classic tokens

Fine-grained PAT minimums: Contents R/W, Actions R/W, Workflows R/W, Metadata R.

## Owner action checklist

Do these **before** relying on Codex to deploy:

### A. Firewall / SSH reachability

- [ ] Confirm provider/security-group / `ufw` / cloud firewall allows **inbound TCP 2021** from **GitHub-hosted Actions egress** (or use a self-hosted runner on the VPS / allowed network)
- [ ] If Codex Cloud previously saw **Connection refused** on 2021, fix allowlists for that egress IP range (not only the Cursor agent IP)
- [ ] VPS diag sibling report: see Project store `internal/codex-deploy-vps-ssh-diag.md` when available (**pending VPS diag** if that file is not yet present)
- [ ] Do **not** open SSH to the world broader than needed; prefer allowlist or self-hosted runner

### B. Deploy identity on VPS

- [ ] Generate dedicated ed25519 keypair (`pikko-gha-deploy`)
- [ ] Append **public** key to `ubuntu@…:~/.ssh/authorized_keys`
- [ ] Store **private** key only in GitHub Environment secret `DEPLOY_SSH_PRIVATE_KEY`
- [ ] Verify: `ssh -i pikko-deploy-ed25519 -p 2021 -o StrictHostKeyChecking=yes ubuntu@43.163.99.95 'hostname; git -C ~/pikko rev-parse HEAD'`

### C. GitHub Environment + secrets

- [ ] Create Environment `production`
- [ ] Set required reviewers (owner)
- [ ] Set all five secrets listed above
- [ ] Capture `DEPLOY_SSH_KNOWN_HOSTS` via `ssh-keyscan -p 2021 43.163.99.95`

### D. Codex / GitHub App permissions

- [ ] Ensure Codex / Cursor GitHub connection can push to `rubicahq/pikko` and dispatch workflows
- [ ] Grant Actions Read/write + Contents Read/write (+ Workflows if editing YAML)
- [ ] Approve apps under GitHub **Settings → Applications** (or org Third-party access) if prompted

### E. First verification (no production deploy)

- [ ] Merge this PR (or run from the PR branch once workflow file is on the default ref you dispatch)
- [ ] `gh workflow run … -f confirm_deploy=false` and confirm dry-run Step Summary shows remote HEAD
- [ ] Only then run with `confirm_deploy=true` after Environment approval

## Related docs

- `docs/DEPLOYMENT.md` — VPS topology, surfaces, first-time setup
- `docs/DEPLOYMENT_RUNBOOK.md` — day-2 ops entrypoint
- `scripts/deploy.sh` — pull, selective rebuild, smoke, rollback
