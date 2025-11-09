# Research Notes — 2025-11-08

## Toolchain Decisions
- Go 1.25.3 (released 2025-10-13) becomes the baseline toolchain; install via `mise` (`mise install` reads `.mise.toml` and fetches the pinned Go build).
- Bubble Tea stack moves to v2.0.0-rc.1 (import path `charm.land/bubbletea/v2`); pair with latest Bubbles/Lip Gloss RCs to keep APIs compatible.

## Runtime Dependencies (Ubuntu 24.04 LTS)
- rsync 3.2.7-1ubuntu1.2 is the current patched build; feature gating should assume 3.2.7 locally but probe remotes before enabling zstd or `--mkpath`. Non-Ubuntu hosts also target rsync ≥ 3.2.7 so we can avoid code forks.
- OpenSSH 9.6p1 (package version 1:9.6p1-3ubuntu13.11) ships by default; new routing-domain syntax is available, SHA-1 SSHFP deprecations underway. Remote hosts inherit the same minimum (9.6p1) for consistency.

## Filesystem Capabilities
- Reflink detection relies on `ioctl(FICLONE/FICLONERANGE)` on Linux (xfs, btrfs, ext4 w/ reflink) and `clonefile()` on macOS; fall back to copies when ioctl returns `EOPNOTSUPP`.
- Btrfs send/receive path requires read-only snapshots as inputs and empty directories for receive targets; ongoing send-stream v2 work is not GA yet but informs future optimization.
- Sparse-file detection should use `FIEMAP` to decide when to pass `--sparse` to rsync, avoiding corruption on filesystems lacking sparse support.
- FIEMAP-based sparse detection now implemented in `fsprobe.HasSparseData`, enabling rsync `--sparse` automatically when holes are found.
- Btrfs subvolume quick-check: inode 256 indicates a subvolume root on most installations; treat it as heuristic, not a guarantee.

## Planner + UI Implications
- Profiles: auto defaults to no compression unless WAN heuristics trigger zstd L1; LAN profile is compression-off/prescan-off; WAN profile enforces zstd L1 and optional prescan.
- Bubble Tea RC introduces renderer refactors and keyboard enhancement messages, so `/internal/tui` should abstract over these APIs early to simplify future upgrades.
- Planner baseline: deterministic ordering (alphabetical sources), rename when `--move` and SameDevice succeeds, fallback to reflink when allowed, else rsync; auto transport adds btrfs offer step when both ends are btrfs and source looks like a subvolume.
- CLI now normalizes root/relative paths (filepath.Clean) and forbids destinations that reside inside any source tree to prevent recursive hazards.
- Remote syntax (`user@host:/path`) is parsed via `internal/remotepath`; we currently allow only one remote side per run and downgrade cap flags based on `ssh host rsync --version` probes.
- Btrfs offers prompt on the CLI (TTY only) before execution; accepting snapshots the source, runs `btrfs send | btrfs receive` into the destination parent, renames into place, and marks the rsync step as complete. TUI modal still TODO.
- Rsync detector parses `rsync --version` output to gate optional flags (zstd, `--mkpath`, `--preallocate`, `--compress-choice`), and the arg builder emits baseline flags from the spec. CLI prints synthesized rsync commands for each `rsync` step to aid debugging.
- Bubble Tea v2 shell now renders header/body/footer summaries of plan steps (MODE/Profile/Transport badges) with up/down navigation, `v`-toggle log drawer (last 5 rsync lines), `?` help overlay, `F` freeze badge, plus non-TTY fallback that reuses the plain-text plan output while `--no-ui` still forces the fallback.
- Plan execution now routes through `internal/ops`: StepReflink attempts `FICLONE` per file (preserving ownership + mtimes) and falls back to rsync on unsupported filesystems/dirs, while StepRsync shells out via the existing arg builder; `--dry-run` sticks to planning/TUI only.
- Rsync parser now covers `--info=progress2` totals plus `--out-format=%i|%l|%n%L` events, emitting structured counters for the forthcoming UI (unit tests use golden fixture logs).

## Outstanding Questions
1. (resolved) `mise` covers installing Go 1.25.3; Charm RC deps stay managed via `go mod`.
2. (resolved) Minimum remote rsync/OpenSSH versions match Ubuntu (3.2.7 / 9.6p1) so no duplicate code paths are required.
