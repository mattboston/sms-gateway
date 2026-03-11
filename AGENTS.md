# Agent Guidelines for sms-gateway

SMS Gateway is a Go application with an embedded React WebUI and REST API for sending/receiving SMS via USB GSM modem AT commands. SQLite is default, PostgreSQL is optional, and Swagger docs are generated from code annotations.

## Architecture and Layout

- Frontend build output is embedded into the Go binary via `go:embed`.
- Production runs a single binary serving API + WebUI.
- Development uses separate Vite dev server with API proxy.
- Primary paths:
  - `src/cmd/sms-gateway/` Go entrypoint
  - `src/internal/` backend domain code (api, auth, config, database, modem, models)
  - `src/web/` React + TypeScript frontend
  - `src/migrations/` goose SQL migrations
  - `src/docs/` generated Swagger artifacts
  - `deploy/` deployment assets (systemd, etc.)
  - `.github/workflows/` CI/release workflows

## Operating Rules

- Use `.tmp` for scratch work.
- Use `wt` (worktrunk) for worktree workflows; worktrees live under `.worktrees/<branch-name>`.
- When working from Linear, use the provided branch name.
- For Linear CLI tasks, use the appropriate Linear skill/CLI guidance.
- PR cleanup: run `wt remove <branch-name>`.
- PR title/body are squash-merged into `main`; keep them conventional-commit compatible and plain text (no markdown in description).

## Agent Orchestration

- Use specialized agents when available; run independent work in parallel.
- Gather context before implementation (codebase exploration first; external research when needed).
- Ask clarifying questions only when ambiguity affects implementation decisions.
- Prefer domain-specific agents:
  - Go backend/API/modem logic
  - Shell/automation scripts
  - CI/CD workflows and release pipelines
  - Documentation generation
  - Debugging and production troubleshooting

## Key Commands

```bash
just dev                 # Backend + frontend dev mode
just build               # Build frontend + backend
just test                # Run test suite
just lint                # Run linters
just format              # Run formatters
just swagger             # Regenerate Swagger docs
just migrate             # Apply migrations
just migrate-down        # Roll back last migration
just migrate-new <name>  # Create migration
just release <version>   # Tag and trigger release workflow (e.g. 0.0.1)
just release-dry-run     # Preview recent tags/commits
```
