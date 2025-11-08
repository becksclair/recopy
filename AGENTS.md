# Repository Guidelines

## Project Structure & Module Organization
- `cmd/recopy`: CLI entry point calling `internal/cli`.
- `internal/*`: Feature modules (`fsprobe`, `plan`, `rsync`, `tui`, etc.); keep logic pure and testable.
- `research/`: Dated notes capturing decisions and external references.
- `specs/recopy_plan.md`: Authoritative roadmap; read it plus `/research` and `TODO.md` before every session.
- `test/`: Holds future integration fixtures.

## Build, Test, and Development Commands
- `go build ./cmd/recopy` — compile the CLI.
- `go test ./...` — run all Go tests (happy-path smoke tests only unless fixing a regression).
- `gofmt -w <files>` — enforce formatting before committing.

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
