# recopy TODOs

Tracking per-phase tasks from specs/recopy_plan.md. Use checkboxes to mark progress; keep wording concise.

## Phase 0 – Foundations
- [ ] T0.1 Init repo, go.mod, CI skeleton (go.mod ready; CI deferred per MVP rules)
- [ ] T0.2 Docs: README, DESIGN, CONTRIBUTING

## Phase 1 – CLI & Profiles
- [ ] T1.1 Implement CLI flags, shared option structs, and surface `--profile`
- [ ] T1.2 cp/mv semantics resolver for SRC... DEST rules

## Phase 2 – Filesystem Probe
- [ ] T2.1 Device ID probe + SameDevice helper
- [ ] T2.2 Reflink probe (FICLONE/clonefile)
- [ ] T2.3 Btrfs subvolume probe

## Phase 3 – Operation Planner
- [ ] T3.1 Planner DAG + deterministic ordering
- [ ] T3.2 Safety checks (dest inside src, trailing slash normalization)

## Phase 4 – Rsync Capability + Args
- [ ] T4.1 Local/remote rsync version detector
- [ ] T4.2 Arg builder with profile-aware compression
- [ ] T4.3 FIEMAP sparse detector integration

## Phase 5 – Rsync Output Parsing
- [ ] T5.1 progress2 parser feeding UI
- [ ] T5.2 `--out-format` parser for item events
- [ ] T5.3 Golden tests with fixture logs

## Phase 6 – TUI Shell
- [ ] T6.1 Bubble Tea shell (header/body/footer)
- [ ] T6.2 Log drawer + keybinds
- [ ] T6.3 Non-TTY fallback mode

## Phase 7 – Copy Ops
- [ ] T7.1 Reflink copy op + metadata fixup
- [ ] T7.2 Rsync copy op integration

## Phase 8 – Move Ops
- [ ] T8.1 Rename fast-path orchestrator
- [ ] T8.2 Rsync + prune move path

## Phase 9 – Remote Paths
- [ ] T9.1 Remote path normalization + parsing
- [ ] T9.2 Remote feature probe wiring

## Phase 10 – Btrfs Offers
- [ ] T10.1 Offer gating + heuristics
- [ ] T10.2 TUI modal for offer accept/refuse
- [ ] T10.3 Send/receive pipeline + fallback

## Phase 11 – Modes (Dry/Mirror)
- [ ] T11.1 `--dry-run` plumbing + UI badge
- [ ] T11.2 `--mirror` plan adjustments

## Phase 12 – Parallelism
- [ ] T12.1 Manual `--parallel` orchestrator
- [ ] T12.2 Aggregated UI state for workers

## Phase 13 – Errors & Signals
- [ ] T13.1 Error model + exit codes
- [ ] T13.2 Ctrl-C graceful shutdown

## Phase 14 – Testing Matrix
- [ ] T14.1 Unit tests for CLI, planner, parser
- [ ] T14.2 Integration tests local/remote matrix

## Phase 15 – Performance
- [ ] T15.1 Bench scripts + datasets
- [ ] T15.2 Baseline perf numbers published

## Phase 16 – Packaging & Docs
- [ ] T16.1 Static builds + shell completions
- [ ] T16.2 Man page + `recopy doctor`

## Cross-Cutting Decisions
- [ ] Upgrade toolchain to Go 1.25.3 and update build scripts
- [ ] Adopt Bubble Tea v2 RC + matching Bubbles/Lip Gloss imports
- [ ] Define minimum supported rsync (3.2.7) and OpenSSH (9.6p1) versions for Ubuntu 24.04 hosts
