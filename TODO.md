# recopy TODOs

Tracking per-phase tasks from specs/recopy_plan.md. Use checkboxes to mark progress; keep wording concise.

## Phase 0 – Foundations

- [ ] T0.1 Init repo, go.mod, CI skeleton (go.mod ready; CI deferred per MVP rules)
- [ ] T0.2 Docs: README, DESIGN, CONTRIBUTING

## Phase 1 – CLI & Profiles

- [ ] T1.1 Implement CLI flags, shared option structs, and surface `--profile`
- [ ] T1.2 cp/mv semantics resolver for SRC... DEST rules

## Phase 2 – Filesystem Probe

- [x] T2.1 Device ID probe + SameDevice helper
- [x] T2.2 Reflink probe (FICLONE/clonefile)
- [x] T2.3 Btrfs subvolume probe (basic heuristics)

## Phase 3 – Operation Planner

- [x] T3.1 Planner DAG + deterministic ordering (rename/reflink/rsync heuristics wired)
- [x] T3.2 Safety checks (dest inside src guard + trailing slash normalization)

## Phase 4 – Rsync Capability + Args

- [x] T4.1 Local rsync version detector (remote TBD)
- [x] T4.2 Arg builder with profile-aware compression + CLI preview
- [x] T4.3 FIEMAP sparse detector integration

## Phase 5 – Rsync Output Parsing

- [x] T5.1 progress2 parser feeding UI
- [x] T5.2 `--out-format` parser for item events
- [x] T5.3 Golden tests with fixture logs

## Phase 6 – TUI Shell

- [x] T6.1 Bubble Tea shell (header/body/footer)
- [x] T6.2 Log drawer + keybinds
- [x] T6.3 Non-TTY fallback mode

## Phase 7 – Copy Ops

- [x] T7.1 Reflink copy op + metadata fixup
- [x] T7.2 Rsync copy op integration

## Phase 8 – Move Ops

- [x] T8.1 Rename fast-path orchestrator
- [x] T8.2 Rsync + prune move path

## Phase 9 – Remote Paths

- [x] T9.1 Remote path normalization + parsing
- [x] T9.2 Remote feature probe wiring

## Phase 10 – Btrfs Offers

- [x] T10.1 Offer gating + heuristics
- [x] T10.2 TUI modal for offer accept/refuse
- [x] T10.3 Send/receive pipeline + fallback

## Phase 11 – Modes (Dry/Mirror)

- [x] T11.1 `--dry-run` plumbing + UI badge
- [x] T11.2 `--mirror` plan adjustments

## Phase 12 – Parallelism

- [x] T12.1 Manual `--parallel` orchestrator
- [x] T12.2 Aggregated UI state for workers

## Phase 13 – Errors & Signals

- [x] T13.1 Error model + exit codes
- [x] T13.2 Ctrl-C graceful shutdown

## Phase 14 – Testing Matrix

- [x] T14.1 Unit tests for CLI, planner, parser
- [x] T14.2 Integration tests local/remote matrix

## Phase 15 – Performance

- [x] T15.1 Bench scripts + datasets
- [x] T15.2 Baseline perf numbers published

## Phase 16 – Packaging & Docs

- [ ] T16.1 Static builds + shell completions
- [ ] T16.2 Man page + `recopy doctor`

## Cross-Cutting Decisions

- [x] Upgrade toolchain to Go 1.25.3 and update build scripts
- [x] Adopt Bubble Tea v2 RC + matching Bubbles/Lip Gloss imports
- [x] Define minimum supported rsync (3.2.7) and OpenSSH (9.6p1) versions for Ubuntu 24.04 hosts
- [x] Add shared `mise run` tasks for fmt/vet/test/build/install/ci
