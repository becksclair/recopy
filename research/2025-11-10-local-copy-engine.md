# Local Copy Engine Implementation — 2025-11-10

## Overview
Implemented phases LC-1 and LC-2 of the performance plan to enable local Linux copies via kernel zero-copy mechanisms, bypassing rsync for same-device transfers.

## Changes Summary

### 1. Benchmarking Infrastructure (LC-1 prep)
- **cmd/bench**: Extended with 4 dataset types:
  - `mixed` (default): 64 files × 2 MiB with subdirectories
  - `tiny`: 10k files × 4 KiB (metadata stress test)
  - `large`: Single 1 GiB file (throughput test)
  - `sparse`: 10 GiB sparse file with 2 MiB data

- **Added .mise.toml tasks**:
  - `bench-profile`: Runs hyperfine with `perf stat` for hardware counters
  - `bench-trace`: Records perf data for detailed profiling (`perf report`)
  - `bench-all-datasets`: Iterates through all dataset types with timestamped JSON/Markdown outputs

- **.gitignore**: Added `bench-results/`, `perf.data`
- **bench-results/**: Directory for storing benchmark JSON outputs

### 2. Profiling Workflow Documentation
- **docs/perf.md**: Comprehensive profiling guide covering:
  - Quick start commands (bench-all-datasets, bench-profile, bench-trace)
  - Dataset descriptions and use cases
  - Tool usage (hyperfine, perf, strace, BCC/bpftrace, Perfetto)
  - Analysis workflow for identifying bottlenecks
  - Performance targets (95%/90% of cp -a)
  - Optimization checklist

### 3. Kernel Feature Detection (recopy doctor)
- **internal/cli/doctor_linux.go**: Added kernel feature probes:
  - `copy_file_range`: Tests syscall availability via temp file copy attempt
  - `io_uring`: Placeholder for future liburing integration (currently returns false)
- **internal/cli/doctor_other.go**: Stubs for non-Linux platforms
- **internal/cli/doctor.go**: Integrated probes into `recopy doctor` output with advice

### 4. Local Copy Engine (Phase LC-1)
- **internal/copy** package created with fallback hierarchy:
  1. **tryReflink**: FICLONE ioctl for instant CoW clones (btrfs/xfs)
  2. **tryCopyFileRange**: Zero-copy via `copy_file_range(2)` syscall
  3. **trySendfile**: Alternative zero-copy for older kernels (unused in current path)
  4. **bufferedCopy**: Final fallback with 1 MiB buffer via `io.CopyBuffer`

- **Metadata preservation**:
  - Ownership (uid/gid), permissions, timestamps (atime/mtime)
  - Reuses reflink metadata preservation logic

- **Error handling**:
  - Returns `ErrNotSupported` for EOPNOTSUPP/EXDEV/EINVAL/ENOSYS
  - Removes partial destinations on failure
  - Loops copy_file_range until full file transferred (handles partial copies)

- **Platform support**:
  - `copy_linux.go`: Full implementation with unix syscalls
  - `copy_other.go`: Stubs returning ErrNotSupported

- **Test coverage**: `copy_test.go` validates file copy with metadata preservation

### 5. Planner Integration
- **plan.StepCopy** added to planner step kinds
- **Planner logic**:
  - Chooses `StepCopy` for local same-device copies when reflink unavailable
  - Respects explicit `--transport=rsync` to force rsync (test compatibility)
  - Precedence: rename (move) > reflink > copy > rsync
- **Reason string**: "local copy engine (copy_file_range)"

### 6. Executor Integration (Phase LC-2)
- **ops.handleCopy**: Routes StepCopy to copy engine:
  - Single files → `copy.File` with PreserveAll=true
  - Directories → `copyDirectory` walker pipeline
- **Directory walker**:
  - Uses `filepath.Walk` to enumerate files/dirs
  - Creates directories with `mkdirAll`
  - Copies files via `copy.File` with context cancellation checks
  - Emits per-file errors wrapped with context

### 7. CLI Entry Point
- **cmd/recopy/main.go**: Created minimal entrypoint calling `cli.Run`
- Fixes integration tests that expected `go run ./cmd/recopy`

## Performance Implications
- **Zero-copy paths**: copy_file_range avoids userspace buffer round-trips for same-device copies
- **Reflink per-file**: Unlike planner-level reflink probe, copy engine attempts FICLONE per file, catching cases where directory probe fails but files succeed
- **Fallback safety**: Graceful degradation to buffered copy ensures compatibility on older kernels (< 4.5) or unsupported filesystems

## Testing
- All existing tests pass (`go test ./...`)
- New test: `internal/copy/copy_test.go` validates basic file copy with metadata
- Integration tests (cli) now succeed with cmd/recopy entrypoint

## Future Work (Phase LC-3)
- io_uring integration for large sequential transfers (requires liburing or Go wrapper)
- posix_fadvise hints (SEQUENTIAL, WILLNEED) for prefetching
- Worker pool with separate queues for large vs. small files (currently directory walker is sequential)
- Sparse file optimization via SEEK_DATA/SEEK_HOLE
- Acceptance threshold regression checks in CI

## References
- GNU coreutils copy.c: copy_file_range + reflink usage patterns
- Linux kernel docs: copy_file_range(2), ioctl_ficlonerange(2)
- perf_plan.md: Phases LC-1, LC-2, LC-3 definitions
