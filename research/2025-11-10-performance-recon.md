# Performance Recon — 2025-11-10

## Goals & Scope
- Match or exceed GNU `cp`/`mv` throughput for local Linux filesystems while preserving recopy semantics (reflink, rsync fallback, TUI, safety guards).
- Focus on same-host transfers first; remote rsync/Btrfs send paths remain out-of-scope until local parity is achieved.

## Current Implementation Snapshot
- `internal/plan` only emits `StepRename`, `StepReflink`, or `StepRsync`. Local copies without reflink end up shelling out to rsync (`internal/ops.handleRsync`), so the hot path still pays process startup, tree walks, and rsync's single-thread work queue per step.
- `ops.Executor` schedules at most one rsync per source when `--parallel=1`; parallel copies simply fan rsync invocations across separate goroutines without batching metadata or pipelining directories.
- Metadata normalization (normalizePaths, fsprobe) already guards against dest-in-src hazards, leaving room for more aggressive in-process copy strategies.

## External Findings
- GNU coreutils 9.x now uses `copy_file_range` even for sparse files and treats reflink/transient errors eagerly, yielding better zero-copy throughput on filesystems like XFS/Btrfs/ZFS (`cp --sparse=auto` default). citeturn1search1
- `copy_file_range` can leverage filesystem/server-side copy and hardware offload, making it preferable to `sendfile`/`splice` for same-fs transfers when supported; fall back to `sendfile`/user-space loops if the syscall returns `-EXDEV`/`-EOPNOTSUPP`. citeturn1search2
- Kernel work continues to expand `copy_file_range` (e.g., cross-superblock allowances, FUSE 64-bit support), so recopy should gracefully detect kernel/FileSystem quirks and downgrade when necessary. citeturn1search6turn1search7
- Glibc docs emphasize that `copy_file_range` skips metadata; tools must separately preserve mode/ownership/timestamps—mirroring what our reflink path already does. citeturn1search10
- Perfetto/`heapprofd` documentation shows how to mix heap sampling with system tracing, useful once we build heavier in-process pipelines that need memory regressions tracked. citeturn5search0

## Tooling & Benchmark Notes
- Keep `hyperfine` harness (`cmd/bench`) for macro-level regression tracking; extend with fixture sizes (small files, many tiny files, deep trees).
- For kernel-level visibility, plan to use `perf stat/record`, `perf c2c`, and eBPF/BCC tools (e.g., `biolatency`, `filetop`) to validate IO submission patterns when we move away from rsync.
- Consider Perfetto/tracebox captures when instrumenting io_uring vs. copy_file_range fallback logic to ensure we don’t regress UI responsiveness.
