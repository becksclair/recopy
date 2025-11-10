# Performance Profiling Guide

This document describes the profiling workflow for measuring and optimizing recopy's local copy performance.

## Overview

recopy aims to match or exceed GNU `cp`/`mv` throughput on local Linux filesystems (ext4/xfs/btrfs). This guide covers:
- Benchmarking datasets and scenarios
- Profiling tools (hyperfine, perf, strace, BCC)
- Analysis workflow
- Regression detection

## Quick Start

### Run All Benchmarks

```bash
mise run bench-all-datasets
```

This runs hyperfine across all dataset types (mixed, tiny, large, sparse) and stores JSON + Markdown reports in `bench-results/`.

### Single Dataset Benchmark

```bash
# Default (mixed dataset: 64 files × 2 MiB)
mise run bench

# Specific dataset
go run ./cmd/bench --dataset=tiny
go run ./cmd/bench --dataset=large
go run ./cmd/bench --dataset=sparse
```

### Profile with perf stat

```bash
mise run bench-profile
```

Shows CPU cycles, cache misses, syscalls, and other hardware counters via `perf stat -d -d`.

### Trace with perf record

```bash
mise run bench-trace
```

Records call stacks at 99 Hz. After completion:
```bash
perf report
```

## Dataset Types

### mixed (default)
- 64 files × 2 MiB (128 MiB total)
- 4 subdirectories
- Tests moderate file sizes with directory structure

### tiny
- 10,000 files × 4 KiB (40 MiB total)
- 100 subdirectories
- Stresses metadata operations (mkdir, stat, open/close)

### large
- 1 file × 1 GiB
- Pure sequential throughput test
- Measures copy_file_range / reflink / rsync efficiency

### sparse
- 1 sparse file: 10 GiB logical size, 2 MiB data (at start + end)
- Tests sparse file detection and SEEK_HOLE handling

## Profiling Tools

### hyperfine
Primary benchmarking tool for wall-clock comparisons.

```bash
# Custom runs/warmup
go run ./cmd/bench --runs=10 --warmup=2
```

### perf stat
Hardware counters + syscall stats:
```bash
perf stat -d -d ./recopy --no-ui src dest
```

Key metrics:
- `task-clock`: CPU time used
- `cache-misses`: L1/L2/L3 misses
- `page-faults`: minor (cached) vs major (disk)
- `instructions` / `cycles`: IPC (instructions per cycle)

### perf record / perf report
Call-graph profiling:
```bash
perf record -F 99 -g ./recopy --no-ui src dest
perf report --stdio
```

Focus on hot paths: `copy_file_range`, `FICLONE`, rsync spawning.

### strace
Syscall tracing (use sparingly—high overhead):
```bash
strace -c ./recopy --no-ui src dest
```

Count syscalls by type to identify inefficiencies (e.g., excessive `stat`, `open`).

### BCC/bpftrace (Linux ≥ 5.10)
Kernel-level observability:

**filetop** (file I/O by process):
```bash
sudo filetop -C
```

**biolatency** (block I/O latency distribution):
```bash
sudo biolatency
```

**trace copy_file_range**:
```bash
sudo bpftrace -e 'tracepoint:syscalls:sys_enter_copy_file_range { @count[comm] = count(); }'
```

### Perfetto / tracebox
System-wide tracing with UI (optional, heavier setup):
```bash
# Record 10s trace
perfetto --txt -o trace.perfetto -c tracebox.cfg

# View in https://ui.perfetto.dev
```

Useful for diagnosing TUI responsiveness and GC pauses.

## Analysis Workflow

### 1. Establish Baseline
Run `cp -a` vs `recopy` for each dataset:
```bash
for ds in mixed tiny large sparse; do
  go run ./cmd/bench --dataset=$ds
done
```

Compare `hyperfine` results (mean, stddev). recopy should be within 5% for large files, 10% for tiny files.

### 2. Profile Hot Paths
If recopy is slower:
```bash
mise run bench-profile
```

Check:
- **High `task-clock` but low throughput?** → CPU bottleneck (excessive copying, parsing)
- **High `cache-misses`?** → Poor memory locality (large buffers, scattered metadata)
- **High `page-faults`?** → Swap thrashing or cold pages

### 3. Trace Syscalls
```bash
perf record -F 99 -g ./recopy --no-ui src dest
perf report
```

Look for:
- Frequent `stat`/`lstat` → cache file metadata in walker
- Many small `write` calls → increase buffer size
- Slow `copy_file_range` → kernel/FS limitation; fall back to sendfile

### 4. Compare Kernel Features
Use `recopy doctor` (extended in this performance plan) to verify:
- `copy_file_range` support
- reflink/FICLONE availability
- io_uring presence (future optimization)

### 5. Regression Detection
Store JSON results with timestamps:
```bash
go run ./cmd/bench --dataset=mixed --export-json=bench-results/mixed-$(date +%s).json
```

Compare against previous runs:
```bash
# Manual check: hyperfine JSON has "mean" field in seconds
jq '.results[] | select(.command | contains("recopy")) | .mean' bench-results/*.json
```

Future: automated CI checks with ≤5% regression tolerance.

## Target Performance (Linux ≥ 5.10)

| Dataset | cp -a (MiB/s) | recopy Target | Status |
|---------|---------------|---------------|--------|
| large (1 GiB) | ~3000 | ≥2850 (95%) | 🚧 |
| mixed (128 MiB) | ~2000 | ≥1900 (95%) | 🚧 |
| tiny (10k × 4K) | ~500 | ≥450 (90%) | 🚧 |
| sparse (10 GiB) | ~instant | instant | 🚧 |

(Numbers are illustrative; actual results depend on hardware/FS.)

## Optimization Checklist

- [ ] Phase LC-1: `copy_file_range` + reflink per-file attempt
- [ ] Phase LC-2: Directory walker with worker pool
- [ ] Phase LC-3: io_uring for large files (optional)
- [ ] posix_fadvise (SEQUENTIAL, WILLNEED) for prefetching
- [ ] Batch metadata operations (reduce stat calls)
- [ ] Parallel workers for tiny files (future)

## References

- GNU coreutils cp source: [coreutils/src/copy.c](https://git.savannah.gnu.org/cgit/coreutils.git/tree/src/copy.c)
- copy_file_range(2): `man 2 copy_file_range`
- perf-tools: `man perf-stat`, `man perf-record`
- BCC toolkit: [iovisor/bcc](https://github.com/iovisor/bcc)
- Perfetto docs: [perfetto.dev/docs](https://perfetto.dev/docs/)

## See Also

- `specs/perf_plan.md` — Detailed milestones (LC-1, LC-2, LC-3)
- `research/2025-11-10-performance-recon.md` — External findings and kernel notes
- `cmd/bench/main.go` — Benchmark harness implementation
