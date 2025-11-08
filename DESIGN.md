# recopy design (snapshot)

This document mirrors the multi-phase plan in `specs/recopy_plan.md` but distills immediate architecture choices for the prototype.

## Modules
- `cmd/recopy`: CLI entry, delegates to `internal/cli` and future planners.
- `internal/cli`: Flag parsing, profile defaults, normalization of SRC...DEST semantics.
- `internal/fsprobe`: Platform-specific helpers for st_dev, reflink, and btrfs capabilities (stubbed today).
- `internal/plan`: Planner that decides between rename/reflink/rsync/btrfs steps (stubbed today).
- `internal/execx`: Process runner for rsync/ssh/btrfs commands (stubbed today).
- `internal/rsync`: Version probe, arg builder, and output parser (stubbed today).
- `internal/tui`: Bubble Tea v2 models providing progress UI (stubbed today).
- `internal/mv`: Move orchestration gluing rename + rsync + prune paths (stubbed today).
- `internal/util`: Shared helpers (path, env, bytes, clocks) (stubbed today).
- `test`: Future integration fixtures.

## Near-term priorities
1. Solidify CLI contract (argument validation, profile presets, help text).
2. Implement fsprobe helpers so planner can reason about devices/reflinks/btrfs.
3. Build the planner DAG + safety checks.
4. Integrate rsync capability detection + arg builder.
5. Stand up the Bubble Tea shell once transfer plumbing is ready.

## Decision log
- Toolchain baseline: Go 1.25.3.
- UI stack: Bubble Tea v2.0.0-rc.1 (+ latest Bubbles/Lip Gloss RCs).
- Runtime dependencies targeted for Ubuntu 24.04 (rsync 3.2.7, OpenSSH 9.6p1).
- Project remains prototype-first: prioritize happy path, single smoke test max per feature.
