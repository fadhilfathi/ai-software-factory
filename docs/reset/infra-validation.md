# Infrastructure validation report — E-001

**Owner:** Ops
**Date:** 2026-06-14
**Scope:** Sprint 5 cleanup, end-of-sprint infra readiness check
**Method:** static analysis (no Docker available in CI shell)

## Summary

| Check | Result |
|---|---|
| `docker-compose.yml` parses as valid YAML | ✅ pass |
| All required services present (`db`, `redis`, `api`, `frontend`) | ✅ pass |
| All services have `healthcheck` directives | ✅ pass |
| `Dockerfile` exists for `api` and `frontend` builds | ✅ pass |
| `Dockerfile`s declare `HEALTHCHECK` and switch to non-root user | ✅ pass |
| All env vars referenced by compose are defined in `.env.example` | ✅ pass |
| `scripts/deploy.sh` uses `docker compose` | ✅ pass |
| `scripts/healthcheck.sh` probes the `/healthz` endpoint | ✅ pass |
| `scripts/validate-infra.py` (this validator) is idempotent | ✅ pass |
| `.env.example` contains no real-secret patterns | ✅ pass |
| **Errors** | **0** |
| **Warnings** | **4** |

## Warnings (non-blocking)

### W-1, W-2, W-3, W-4: services do not use `env_file: .env`

The `db`, `redis`, `api`, and `frontend` services do not declare an `env_file:` block. They use inline `environment:` blocks with shell-variable substitution (`${POSTGRES_USER:-postgres}`) and built-in defaults.

**Decision:** accepted. This is the team's current convention — see `docker-compose.yml` lines:

```yaml
api:
  environment:
    PORT:        ${API_PORT:-8080}
    DB_HOST:     db
    JWT_SECRET:  ${JWT_SECRET:-dev_only_secret_32_chars_minimum_for_local_testing}
```

Operators supply overrides via a project-root `.env` (per the header comment in `docker-compose.yml`). The `api` service does **not** inject `.env` into the container; only the named `environment` keys are exported. This is intentional — it keeps the container's environment surface small.

**Caveat:** if a new operator-only secret is added to `.env` (e.g. a future `GITHUB_TOKEN` for the agent), it must be added to the `environment:` block explicitly. The `.env` is only consumed by compose at interpolation time, not by the running container. Recommend documenting this in the runbook.

## How to re-run

```
python scripts/validate-infra.py
```

Exit code 0 = clean, exit code 1 = errors found (warnings still allow 0). The script is idempotent and read-only.

## What this does NOT check

- **Live `docker compose up` smoke test** — Docker is not installed in the CI shell that ran this validator. The validator is static-only by design.
- **Runtime health** — the `scripts/healthcheck.sh` performs live probing but requires a running stack. See `infra-runbook.md` for the operator procedure.
- **Image-build success** — `docker compose build` requires Docker. The CI workflow `ci.yml` runs `npm run build` for the frontend and `go build ./...` for the backend, but not Docker builds.

## Recommended follow-ups (out of scope for E-001)

- [ ] Add a `docker compose config --quiet` step to `ci.yml` (validates interpolation without building).
- [ ] Add a `docker build` smoke step to a separate nightly workflow (slow; don't gate PRs on it).
- [ ] Add a `docker compose up --wait` integration test in a dedicated runner with Docker (E-001's only gap).
- [ ] Document the `env_file` vs `environment` convention in `docs/operations/infra.md` (the runbook already mentions it).
