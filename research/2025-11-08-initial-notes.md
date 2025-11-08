# Research Notes — 2025-11-08

## Toolchain Decisions
- Go 1.25.3 (released 2025-10-13) becomes the baseline toolchain; install via official tarball because Ubuntu 24.04 apt repos top out at 1.22/1.23.
- Bubble Tea stack moves to v2.0.0-rc.1 (import path `charm.land/bubbletea/v2`); pair with latest Bubbles/Lip Gloss RCs to keep APIs compatible.

## Runtime Dependencies (Ubuntu 24.04 LTS)
- rsync 3.2.7-1ubuntu1.2 is the current patched build; feature gating should assume 3.2.7 locally but probe remotes before enabling zstd or `--mkpath`.
- OpenSSH 9.6p1 (package version 1:9.6p1-3ubuntu13.11) ships by default; new routing-domain syntax is available, SHA-1 SSHFP deprecations underway.

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
- Rsync detector parses `rsync --version` output to gate optional flags (zstd, `--mkpath`, `--preallocate`, `--compress-choice`), and the arg builder emits baseline flags from the spec. CLI prints synthesized rsync commands for each `rsync` step to aid debugging.
- Bubble Tea v2 shell now renders header/body/footer summaries of plan steps (MODE/Profile/Transport badges) with up/down navigation and non-TTY fallback that reuses the plain-text plan output while `--no-ui` still forces the fallback.
- Rsync parser now covers `--info=progress2` totals plus `--out-format=%i|%l|%n%L` events, emitting structured counters for the forthcoming UI (unit tests use golden fixture logs).

## Outstanding Questions
1. Need scripted instructions for installing Go 1.25.3 + Charm RC deps in CI/dev images.
2. Confirm minimum remote rsync/OpenSSH expectations beyond Ubuntu 24.04 (e.g., macOS, Debian stable) for cross-platform support.
