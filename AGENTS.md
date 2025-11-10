# Repository Guidelines

## Project Structure & Module Organization
- `cmd/recopy`: CLI entry point calling `internal/cli`.
- `internal/*`: Feature modules (`fsprobe`, `plan`, `rsync`, `tui`, etc.); keep logic pure and testable.
- `research/`: Dated notes capturing decisions and external references.
- `specs/recopy_plan.md`: Authoritative roadmap; read it plus `/research` and `TODO.md` before every session.
- `test/`: Holds future integration fixtures.

## Build, Test, and Development Commands
- `mise run fmt` — run `gofmt -s` over all tracked Go files.
- `mise run vet` — `go vet ./...` sanity checks.
- `mise run test` — execute the happy-path `go test ./...` suite.
- `mise run ci` — convenience task that runs fmt, vet, then test.
- `mise run build` — compile the CLI (`go build ./cmd/recopy`).
- `mise run install` — install `recopy` into your Go bin path.
- You can still call the underlying `go build`, `go test`, or `gofmt` commands directly when faster.

## Coding Style & Naming Conventions
- Go 1.25.3 toolchain; rely on stdlib unless a dependency saves ≥30 minutes.
- Follow Go naming (exported types/functions start with uppercase); prefer small, composable packages.
- Keep comments concise; add only when behavior is non-obvious.
- No CI or container changes per MVP charter.

## Testing Guidelines
- Max one happy-path smoke test per feature unless addressing a confirmed bug.
- Use Go’s built-in `testing` package; name tests `TestFeatureBehavior`.
- Update `/research` with any noteworthy test findings or environment quirks.

## Commit & Pull Request Guidelines
- Commit messages mirror existing style (e.g., `init scaffold`): short, lowercase, action-first.
- Each change should update `TODO.md` checkboxes and `/research` as needed.
- PRs (if used) should state scope, highlight affected phases, and mention verification commands run.

## Agent-Specific Instructions
- Start every session by reviewing `specs/recopy_plan.md`, the entire `/research` directory, and `TODO.md`.
- Prefer incremental progress over perfection; note open questions in `/research` when blocking issues arise.
