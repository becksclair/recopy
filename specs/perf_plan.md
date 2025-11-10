# recopy Performance Plan (Local Linux Parity)

## Goal

Match or exceed GNU `cp`/`mv` throughput on local Linux filesystems (ext4/xfs/btrfs) without regressing existing safety guarantees, then establish a repeatable profiling workflow for future optimizations.

## Scope

- Local, same-host transfers only (remote rsync/Btrfs send excluded).
- Focus on copy/move hot paths (rename/reflink/rsync replacements) plus benchmarking + tooling.
- Linux kernels ≥ 5.10; filesystems ext4/xfs/btrfs covering reflink/no-reflink permutations.

## Constraints

- Preserve cp-style semantics (metadata, sparse files, trailing slash behavior).
- Honor existing CLI flags (`--move`, `--parallel`, `--no-reflink`, etc.).
- Stay within MVP charter: minimal dependencies, single happy-path tests per feature.

## Research Highlights (2025-11-10)

- GNU coreutils uses `copy_file_range` + reflink + sparse heuristics by default; we must match those zero-copy paths. citeturn1search1
- `copy_file_range` can fall back to buffered loops when fs/kernel refuses; treat `EOPNOTSUPP`, `EXDEV`, `EINVAL` as downgrade signals. citeturn1search2
- Kernel work continues to extend CFR across superblocks, so feature detection must be runtime, not compile-time. citeturn1search6turn1search7
- CFR omits metadata; we must explicitly preserve mode/ownership/timestamps. citeturn1search10
- Perfetto/heapprofd combo gives us heap + trace visibility once we build heavier in-process pipelines. citeturn5search0

## Deliverables & Milestones

### 1. Instrumentation & Baselines

- Extend `cmd/bench` datasets (tiny files, mixed tree, large sparse file).
- Add `bench-profile`/`bench-trace` mise tasks that wrap `hyperfine`, `perf stat/record`, and `tracebox` configs.
- Capture baseline JSON (cp vs recopy) per dataset; store under `bench-results/` for diffing.

### 2. Profiling Workflow Docs

- Create `docs/perf.md` describing step-by-step profiling (hyperfine, perf, strace, BCC filetop/biolatency).
- Tie into `recopy doctor` to report availability of `copy_file_range`, io_uring, reflink support.

### 3. Local Copy Engine (Phase LC-1)

- Add `plan.StepCopy` for same-device, non-reflink copies.
- Implement `internal/copy` package:
  - Try reflink per file (FICLONERANGE) even when directory probe failed.
  - Fall back to `copy_file_range` loops; on error, downgrade to `sendfile` or buffered readers.
  - Reuse `preserveMetadata` so behavior matches current reflink path.
- Wire executor to batch files, create target directories once, and stream data in-process.

### 4. Directory Walker & Scheduling (Phase LC-2)

- Build a single `WalkDir` pipeline that emits file tasks with metadata to avoid repeated `stat` calls.
- Introduce worker pool honoring `--parallel`: separate queues for large files (streaming) and small files (batch copy) to keep CPUs busy.

### 5. Advanced IO (Phase LC-3)

- Investigate io_uring-based copy (via `liburing` or Go wrappers) for large sequential transfers; gate behind runtime detection.
- Add optional posix_fadvise hints (`SEQUENTIAL`, `WILLNEED`) and `fdatasync` policy for `--verify`/`--preserve` modes.

### 6. Regression Guardrails

- Define acceptance thresholds per dataset (e.g., recopy within 5% of `cp` for ≥1 GiB single file, within 10% for 10k×4 KiB trees).
- Automate comparison scripts that fail CI/manual checks if throughput drops beyond tolerance.
- Record perf stats + JSON outputs in `/bench-results` with timestamps for historical trend.

## Verification Strategy

1. Run `mise run bench` (all datasets) + `recopy doctor` after each optimization.
2. Capture `perf stat` counters for both `cp` and `recopy` to ensure comparable syscall/IO mix.
3. Use Perfetto traces selectively to confirm TUI responsiveness and absence of long GC pauses.
4. Update research notes with each completed phase, including benchmark deltas and profiling artifacts.

## Exit Criteria

- Recopy matches or beats `cp -a` wall-clock throughput on the defined datasets under Linux (ext4/xfs/btrfs) with kernel ≥ 5.10.
- Profiling workflow (docs + scripts) reproducibly highlights regressions and is referenced in README/CONTRIBUTING.
- `recopy doctor` reports relevant kernel feature availability to set user expectations.
