# recopy — Multi‑Phase Build Plan (v1 → v1.1)

**Goal:** Replace `cp`/`mv` with a fast, resumable, delta‑smart tool with a clean, modern TUI, smart local fast paths (rename/reflink), robust rsync orchestration for cross‑FS/SSH, and an optional btrfs send/receive path when safe.

---

## 0) Project Foundations

**Deliverables**
- Repository structure
  - `/cmd/recopy` — CLI entrypoint
  - `/internal/cli` — flag parsing, profiles, validation
  - `/internal/fsprobe` — filesystem + reflink/btrfs detection
  - `/internal/plan` — operation planning (rename/reflink/rsync/btrfs)
  - `/internal/execx` — child process runner (rsync, ssh, btrfs)
  - `/internal/rsync` — args builder, version features, output parser
  - `/internal/tui` — Bubble Tea models: UI state, progress, keybinds
  - `/internal/mv` — move orchestration (rename + rsync + prune)
  - `/internal/util` — shared helpers (path, env, bytes, time)
  - `/test` — integration tests, fixtures
- LICENSE (MIT), CODEOWNERS, CONTRIBUTING.md
- `README.md` (purpose, quick start, safety guarantees)
- `DESIGN.md` (short narrative of architecture)

**Acceptance**
- `go build ./cmd/recopy` produces a static binary
- `recopy --help` prints CLI synopsis and profiles

**Dependencies**
- Go ≥ 1.22
- External binaries present at runtime: `rsync`, `ssh`, optionally `btrfs` tools

---

## 1) CLI & Profiles (Surface Area Freeze for v1)

**CLI**
```
recopy [--move] [--dry-run] [--mirror]
       [--profile auto|lan|wan]
       [--no-reflink] [--inplace]
       [--parallel N]
       [--transport auto|rsync|btrfs]
       [--prescan]
       [--verify]
       [--one-file-system]
       [--no-ui]
       SRC... DEST
```

**Profiles**
- `auto` (default): local/LAN → no compression; WAN → zstd L1; parallel=1
- `lan`: force no compression, no prescan, parallel=1
- `wan`: zstd L1, prescan optional (off by default in v1), parallel=1

**Semantics**
- cp‑style: copy contents of directories (normalize trailing slash behavior)
- mv‑style: `--move` implies rename on same FS; else rsync + prune

**Acceptance**
- Argument validation for multiple sources, dest dir/file rules
- Profiles applied deterministically and reflect in planned run

**Dependencies**
- none

---

## 2) Filesystem/Path Probing

**Features**
- Determine device IDs (st_dev) for sources and destination parent
- Detect reflink capability (Linux: `FICLONE` ioctl; macOS: clonefile)
- Detect btrfs subvolume eligibility (is subvol; can snapshot; ro snapshot path)
- Normalize cp semantics: resolve `src/` vs `src` consistently (copy contents)
- Guardrails: prevent `dest` inside `src` (recursive hazard), resolve symlinks per `-a` semantics

**Acceptance**
- `fsprobe.SameDevice(src, destParent)`
- `fsprobe.SupportsReflink(path)` → bool
- `fsprobe.BtrfsInfo(path)` → {isBtrfs, isSubvolume}
- Unit tests with tmpfs/ext4/xfs/btrfs as available (mock when not)

**Dependencies**
- OS syscalls (`stat`, `ioctl`), btrfs helpers

---

## 3) Operation Planner (DAG)

**Goal:** Produce an execution plan independent of UI.

**Inputs**
- Paths (SRC… DEST), options, profile, probes

**Outputs**
- Ordered steps: one of
  - `RenameBatch` (per device)
  - `ReflinkCopyBatch` (local, reflink‑capable)
  - `RsyncBatch` (local cross‑FS or SSH)
  - `BtrfsOffer` (only if auto‑probed safe)

**Rules**
- For `--move`: group by device → `RenameBatch` for same‑device moves; remainder plan as `RsyncBatch` with `--remove-source-files`
- For copy: prefer `ReflinkCopyBatch` when local + capable; else `RsyncBatch`
- If both ends btrfs and `src` is subvol or snapshot‑able: include `BtrfsOffer` ahead of corresponding `RsyncBatch`

**Acceptance**
- Deterministic plan given fixed options
- Planner unit tests for mixed inputs (files/dirs, mixed devices)

**Dependencies**
- fsprobe

---

## 4) Rsync Capability Detection & Arg Builder (v1)

**Detect**
- Local `rsync` version; remote version (if SSH paths present)
- Feature gates: zstd (`-zz`/`--compress-choice=zstd`), `--mkpath`, `--checksum-choice`, `--sparse`, `--preallocate`

**Arg Builder**
- Baseline flags (v1):
  - `-aHAX` (preserve), `--info=progress2`, `--itemize-changes`
  - `--out-format=%i|%l|%n%L`, `--outbuf=L`, `--human-readable`
  - `--partial --partial-dir=.rsync-partial` (temp chunks), `--preallocate`
  - compression: per profile — none (local/LAN) or `--compress --compress-choice=zstd --zl=1` (WAN)
  - add `--sparse` if FIEMAP detects sparse inputs
  - add `--one-file-system` if requested
  - **DO NOT** set `--inplace` by default (opt‑in)
- Move: add `--remove-source-files`; later prune empties
- Mirror: add `--delete-delay` (explicit opt‑in)

**Acceptance**
- Given an input scenario, builder returns a stable argv slice
- Version downgrades (no zstd) handled without failure

**Dependencies**
- execx to call `rsync --version`; optional remote probe via `ssh host rsync --version`

---

## 5) Rsync Output Parser (progress & events)

**Streams**
- Read combined stdout/stderr with `--outbuf=L` and `--msgs2stderr`

**Parse**
- `--info=progress2` lines → total bytes sent/received, rate, ETA, `(xfr#N, to-chk=R/T)`
- `--out-format` events → `%i` change flags, `%l` length, `%n%L` path (w/ symlink target)

**Model**
- Maintain counters: total items (T), remaining (R), checked (T‑R), transferred (N)
- Current file (from last item event with active transfer)
- Overall byte progress = from progress2 total

**Acceptance**
- Golden‑file tests: feed captured rsync outputs; parser yields stable state transitions
- Locale forced via `LC_ALL=C` to ensure consistency

**Dependencies**
- none

---

## 6) TUI v1 (Bubble Tea)

**Layout**
- Header: MODE (COPY/MOVE/DRY‑RUN) • TRANSPORT badge (`reflink/rename/rsync`) • Overall progress bar • Speed • ETA • Compression badge (`zstd L1` or `none`) • ALERT badge if `--mirror`
- Body row: current filename (middle‑elided), per‑file mini bar, file size + instantaneous MB/s
- Footer: counters `done/total` (checked), `xfr#` transferred; key help: `q` quit, `v` log, `F` freeze, `?` help
- Drawer: collapsible log tail (raw rsync lines)

**Behavior**
- Non‑TTY → plain line output (no ANSI)
- `--no-ui` → force plain output
- Btrfs offer modal appears if planner included `BtrfsOffer` (single‑key accept/refuse)

**Acceptance**
- Renders without flicker under resize
- 60Hz updates under heavy output; never blocks the rsync pump

**Dependencies**
- rsync parser state

---

## 7) Core Operations: Copy (Local)

**Reflink Copy (v1)**
- Attempt `FICLONE` per file (or clonefile on macOS)
- On `EOPNOTSUPP`/`EXDEV`: fall back to rsync path for that file/batch
- Preserve metadata post‑clone (`utimes`, `chmod`, `chown` as needed)

**Rsync Copy (v1)**
- Use arg builder from §4

**Acceptance**
- Measured: large file copy on btrfs/xfs is near‑instant (bytes on write only upon CoW)
- Fallback is seamless without user prompts

**Dependencies**
- fsprobe, rsync builder, execx

---

## 8) Core Operations: Move

**Same‑device Moves**
- Batch by device; perform `rename(2)` per entry; preserve structure

**Cross‑device Moves**
- Rsync with `--remove-source-files` then prune empty dirs

**Prune**
- Walk source directories bottom‑up; remove empties; skip non‑empties silently

**Acceptance**
- Atomic rename on same FS
- Cross‑device leaves no stray files; empty dirs removed

**Dependencies**
- fsprobe, mv helpers, rsync builder

---

## 9) Remote Paths (SSH)

**Syntax**
- Accept `user@host:/path` on either side; normalize to rsync remote syntax

**Transport**
- v1: use OpenSSH defaults (no forced ControlMaster/ciphers)
- Optional future flag `--ssh-fast` to enable CM/Persist/cipher tweaks

**Acceptance**
- Remote copies succeed with default user SSH config
- Proper escaping: build argv arrays; never shell‑concat

**Dependencies**
- rsync builder

---

## 10) Btrfs Send/Receive (Offer‑When‑Safe)

**Auto‑Probe**
- Both ends btrfs? Source is subvolume or can snapshot ro? Dest can receive under target?

**User Offer (TUI)**
- Modal: “Faster snapshot replication available (btrfs send/receive). Dest will be a subvolume.”
- Keys: `[Y] use btrfs` / `[N] keep rsync`

**Flow**
- If accepted: create ro snapshot (if needed) → `btrfs send` (optionally piped over SSH) → `btrfs receive` under dest → set desired ro/rw
- Else: proceed with planned rsync

**Acceptance**
- Only offered when safe; fallback cleanly otherwise
- Snapshot names include timestamp + recopy tag

**Dependencies**
- fsprobe, execx

---

## 11) Dry‑Run & Mirror

**Dry‑Run**
- `--dry-run` flips rsync `-n` and still drives the TUI using out‑format events; no data changes

**Mirror**
- `--mirror` adds deletions (`--delete-delay`); a red `DELETE ON` badge shows in UI

**Acceptance**
- Dry‑run shows accurate planned counters
- Mirror deletes only on explicit flag; no accidental destructive ops

**Dependencies**
- rsync builder/parser

---

## 12) Parallelism (Manual in v1)

**Feature**
- `--parallel N` spawns N rsync workers with partitioned file lists; isolates partial dirs per worker (`.rsync-partial.Wk`)
- UI aggregates progress across workers; errors collapsed into drawer

**Acceptance**
- Deterministic partitioning; no duplicate work
- No partial collisions; clean shutdown on error/cancel

**Dependencies**
- rsync parser, execx, plan partitioner

---

## 13) Safety, Guards & Errors

**Guards**
- Block dest inside src
- Respect `--one-file-system` if set
- Ensure temp/partial dirs excluded from traversal

**Errors**
- Propagate rsync exit codes; summarize per batch
- Graceful Ctrl‑C: first SIGINT to child; second SIGINT escalates

**Acceptance**
- No foot‑guns; clear failure summaries; partials preserved for resume

**Dependencies**
- execx

---

## 14) Testing Strategy

**Unit**
- CLI parsing, planner rules, parser golden tests

**Integration (Containers/VMs)**
- Local: tmpfs/ext4/xfs/btrfs where possible
- Remote: dockerized rsyncd/SSH server for latency injection

**Scenarios**
- Millions of tiny files; a few huge files (10–100GB); mixed trees
- Same‑device move; cross‑device move; reflink supported/unsupported; WAN link emulation

**Acceptance**
- CI runs green; flakiness < 1%

**Dependencies**
- GitHub Actions (or similar), test fixtures

---

## 15) Performance & Benchmarks

**Metrics**
- Throughput (MB/s), CPU %, syscalls/sec, disk IO wait, network RTT
- TUI update latency (< 50 ms)

**Targets**
- Local reflink copy: near‑instant metadata‑only for large files
- Remote WAN (100 Mbps): zstd L1 within 95% of net line rate on compressible sets
- Parser overhead: < 2% CPU on typical transfers

**Tools**
- `fio`, `pv`, `perf`, `tc netem` for latency, synthetic datasets

**Acceptance**
- Benchmarks documented with reproducible scripts

---

## 16) Packaging & Docs

**Deliverables**
- Static builds for Linux x86_64/arm64 (musl/glibc variants)
- Man page (`recopy(1)`)
- Shell completions (bash/zsh/fish)
- `README`: safety, examples, profiles, caveats

**Acceptance**
- `curl | install` recipe; `recopy doctor` probes environment and prints fast‑path availability

---

## 17) v1.1 Roadmap (deferred features)

- Adaptive compression (batch‑level retune L1→L3/L5)
- Auto parallelism when small‑file+high‑RTT detected
- `--prescan` heuristics and better ETA smoothing
- `--verify=spot` (sampled checksum) and `--verify=full`
- Incremental btrfs send (`--parent`) with snapshot catalog
- Optional `--ssh-fast` (ControlMaster/Persist, cipher tuning)
- JSON event stream (`--json`) for logs and programmatic use
- Atomic directory swap for small trees (`--atomic`)

---

## Risk Register & Mitigations

- **Rsync versions differ (remote older):** feature detection; degrade compression/checksum; warn once
- **Locale parsing issues:** force `LC_ALL=C` for rsync child
- **Sparse file mis‑detection:** gated by FIEMAP; offer manual `--sparse`
- **Btrfs edge cases:** offer only when strict checks pass; otherwise hide the option
- **Parallel workers overload slow NAS:** default parallel=1; manual opt‑in only in v1
- **Inplace corruption on crash:** `--inplace` opt‑in only; default temp+rename
- **Trailing‑slash semantics confusion:** normalize to cp semantics; doc `--rsync-style` for experts (future)

---

## Work Graph (High‑Level Dependencies)

1. Foundations (0) → CLI & Profiles (1)
2. FS Probe (2) → Planner (3) → Copy/Move Ops (7,8)
3. Rsync Builder (4) → Parser (5) → TUI (6) → Dry‑Run/Mirror (11)
4. Remote Paths (9) → integrated with Builder/Parser
5. Btrfs Offer (10) depends on FS Probe and Exec
6. Parallelism (12) depends on Parser + Exec stable
7. Safety/Errors (13) integrates across exec path
8. Testing (14) spans all; start as soon as modules exist
9. Performance (15) after baseline passes
10. Packaging/Docs (16) last before release

---

## Phase-by-Phase Tickets (Executable Units)

**Phase 0**
- T0.1 Init repo, go.mod, CI skeleton
- T0.2 `README`, `DESIGN.md`, `CONTRIBUTING.md`

**Phase 1**
- T1.1 Implement CLI flags + profiles
- T1.2 Semantics resolver for cp/mv behavior

**Phase 2**
- T2.1 st_dev probe & same‑device check
- T2.2 Reflink capability probe (FICLONE)
- T2.3 Btrfs subvol probe

**Phase 3**
- T3.1 Planner rules + unit tests
- T3.2 Safety checks (dest inside src)

**Phase 4**
- T4.1 Rsync version detector (local/remote)
- T4.2 Arg builder (baseline + profiles)
- T4.3 FIEMAP sparse detector hook

**Phase 5**
- T5.1 Rsync progress parser (progress2)
- T5.2 Out‑format parser (item events)
- T5.3 Golden tests with fixture logs

**Phase 6**
- T6.1 TUI shell (header/body/footer)
- T6.2 Log drawer + keybinds
- T6.3 Non‑TTY fallback mode

**Phase 7**
- T7.1 Reflink copy op + metadata fixup
- T7.2 Rsync copy op integration

**Phase 8**
- T8.1 Move (rename path)
- T8.2 Move (rsync+prune path)

**Phase 9**
- T9.1 Remote path normalization
- T9.2 Remote feature probe integration

**Phase 10**
- T10.1 Btrfs offer gating logic
- T10.2 TUI modal + accept/refuse
- T10.3 Send/receive pipeline + fallback

**Phase 11**
- T11.1 `--dry-run` plumbing
- T11.2 `--mirror` plumbing (+ badge)

**Phase 12**
- T12.1 Manual `--parallel` orchestrator
- T12.2 Aggregated UI state for N workers

**Phase 13**
- T13.1 Error model + exit codes
- T13.2 Ctrl‑C graceful shutdown policy

**Phase 14**
- T14.1 Unit tests: cli/planner/parser
- T14.2 Integration tests: local/remote matrices

**Phase 15**
- T15.1 Bench scripts + datasets
- T15.2 Publish baseline perf numbers

**Phase 16**
- T16.1 Packaging (static builds) + completions
- T16.2 Man page + `recopy doctor`

---

## Acceptance Criteria (Release v1)

- `recopy` drop‑in alias for `cp` and `mv` on local + remote paths
- Reflink fast‑path works on btrfs/xfs (auto) with clean fallback
- Rsync orchestrations are stable with clear TUI progress
- Btrfs send/receive appears only when safe and works end‑to‑end
- Non‑TTY mode is script‑friendly
- Integration tests pass across core scenarios
- Performance meets targets in §15

---

## Notes for Future AI Agents

- Keep modules pure and testable; avoid UI logic in planner/builder
- Do not parse human‑readable numbers from rsync; rely on numeric fields
- Always set `LC_ALL=C` for child processes
- Avoid shell‑string command construction; pass argv arrays
- Maintain a feature gate struct from probes to disable unsupported flags

---

**End of Plan (v1 → v1.1)**

