# recopy (prototype)

recopy is a fast, resumable replacement for `cp`/`mv` that stitches together local rename/reflink paths, rsync orchestration, and an upcoming Bubble Tea TUI. This repo currently tracks the MVP scaffold referenced in `specs/recopy_plan.md`.

## Quick start

```bash
# Install the pinned toolchain
mise install

# Build the CLI
GOEXPERIMENT=all go build ./cmd/recopy

# Show help/flags
./recopy --help
```

Requirements:
- `mise` (reads `.mise.toml` and installs Go 1.25.3)
- Runtime tools: `rsync`, `ssh`, optional `btrfs` utils (future phases)

## Development workflow
1. Read `specs/recopy_plan.md` for the authoritative roadmap.
2. Review the living checklist in `TODO.md` to see per-phase progress.
3. Skim the `/research` notes for recent decisions (toolchain, deps, probes).
4. Implement the smallest piece that demonstrates value; update the checklist and research notes as you go.

## Status
- ✅ Go 1.25 toolchain + CLI parser + planner/executor stack (reflink, rsync, move paths, Bubble Tea shell)
- ✅ Remote path normalization plus rsync capability probing on remote hosts
- ✅ Btrfs offers gated via CLI prompt or TUI modal with snapshot + send/receive pipeline
- 🚧 Dry-run/mirror plumbing, parallel workers, errors/signals, and packaging/docs

## License
MIT — see `LICENSE`.
