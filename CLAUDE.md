# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Sentinel is a centralized authentication and authorization service built in Go. It provides JWT RS256 tokens, RBAC, audit logging, and an admin dashboard for internal applications.

## Conventions

- All project documentation lives in `docs/`.
- Specs live in `docs/specs/` — one file per domain.
- Design decisions are marked with `> **Decisión de diseño (YYYY-MM-DD):**` inside spec files. Search with: `grep -rn "Decision de diseno" docs/specs/`
- SQL queries must always use prepared statements — never string concatenation.
- Passwords and refresh tokens are always stored as bcrypt hashes (cost >= 12), never in plain text.
- Normalize passwords to NFC (`golang.org/x/text/unicode/norm`) before hashing or comparing.
- Audit events must be written asynchronously (buffered channel) — never block the HTTP response.

## Project Status

All five phases are complete:

| Phase | Status | Description |
|---|---|---|
| 1 — Analysis & Specs | Done | 42 user stories / 196 SP across 8 epics. Specs in `docs/specs/`. |
| 2 — Infrastructure | Done | Docker Compose (Go + PostgreSQL 15 + Redis 7), Dockerfile, Makefile, config.yaml, 12-table schema. |
| 3 — Backend | Done | Full Go implementation: domain, repositories (pgx + redis), services, handlers, middlewares, JWT RS256, bootstrap. |
| 4 — Frontend | Done | React 18 + Vite + TypeScript + Tailwind. 9 admin pages. Token refresh interceptor. |
| 5 — QA | Done | 62 unit tests + 11 integration tests (testcontainers) + 3 k6 load scenarios + 25 E2E tests (Playwright). |

## Key Architecture Decisions

- **Framework:** Fiber v2 (not Chi).
- **JWT:** RS256 only — keys loaded from PEM files (`keys/private.pem`, `keys/public.pem`). Never HS256.
- **Refresh tokens:** UUID v4 raw is used as the Redis key (bcrypt is non-deterministic). The bcrypt hash is stored as the value in Redis and in PostgreSQL `refresh_tokens.token_hash`.
- **client_type:** Explicit enum field (`web` | `mobile` | `desktop`) in the login request body — never inferred from User-Agent. Determines refresh token TTL (7d for web, 30d for mobile/desktop).
- **Lockout:** `lockout_count` + `lockout_date` columns on `users` table. Runtime reset (no cron). 3 lockouts in the same day = permanent lock (`locked_until = NULL`).
- **Password history:** Dedicated `password_history` table indexed by `user_id`. Last 5 hashes checked on change.
- **Bootstrap:** Runs once when `applications` table is empty. `BOOTSTRAP_ADMIN_PASSWORD` is exempt from password policy.
- **Pagination:** Offset-based. `page`, `page_size`, `total`, `total_pages`. Max `page_size` = 100.
- **Permissions map signature:** Canonical JSON (keys sorted lexicographically, no whitespace) signed with RSA-SHA256.
- **Audit:** Async via buffered channel (size 1000). Failure to write audit log must never fail the main operation.

## Critical Files

| File | Purpose |
|---|---|
| `cmd/server/main.go` | Entry point — Fiber setup, router, CORS, graceful shutdown |
| `internal/config/config.go` | Config loading — YAML + env var overrides + validation |
| `internal/token/jwt.go` | JWT RS256 generation, validation, JWKS |
| `internal/service/auth_service.go` | Login, refresh, logout, change-password logic |
| `internal/service/authz_service.go` | HasPermission, permissions map, cache |
| `internal/bootstrap/initializer.go` | One-time system bootstrap |
| `internal/handler/swagger_types.go` | Swagger-only request/response types (never used in runtime logic) |
| `migrations/001_initial_schema.sql` | Full DDL — 12 tables + indexes |
| `docs/api/` | Generated OpenAPI spec — docs.go, swagger.json, swagger.yaml |
| `docs/specs/` | Source of truth for all business rules |

## Common Commands

```bash
make keys        # Generate RSA key pair in keys/
make docker-up   # Start full stack (Go + PG + Redis)
make build       # Compile to bin/auth-service
make test        # go test ./... -v -cover
make migrate     # Run database migrations

# Regenerate Swagger docs after changing handler annotations
~/go/bin/swag init --generalInfo cmd/server/main.go --output docs/api \
  --parseDependency --parseInternal --exclude web/
```

## Running Tests

```bash
# Unit tests (62 tests)
go test ./internal/... -v -cover

# Integration tests — requires Docker
go test ./tests/integration/... -v -timeout 300s

# Load tests — requires k6 and running service
k6 run --env BASE_URL=http://localhost:8080 --env APP_KEY=<key> \
   tests/load/scenarios/mixed_load.js

# E2E tests — requires full stack running (frontend + backend)
cd web && npm run test:e2e
```

## API Documentation

- Swagger UI is served at `GET /swagger/*` (no auth required).
- Spec files are committed to `docs/api/` — run `swag init ...` to regenerate after annotation changes.
- Swagger types live exclusively in `internal/handler/swagger_types.go`. Never embed Swagger annotations in domain or service structs.
- CORS is enabled globally in `main.go` (`AllowOrigins: "*"`) so Swagger UI can call the API directly.

## Security Constraints

- Never log passwords, tokens, or secret keys.
- All SQL must use parameterized queries (pgx `$1`, `$2`, ...).
- `X-App-Key` validation is required on all endpoints except `GET /health` and `GET /.well-known/jwks.json`.
- Security headers middleware (`Strict-Transport-Security`, `X-Content-Type-Options`, `X-Frame-Options`) must apply to all responses.
- RSA keys are never baked into the Docker image — always mounted as a volume.
