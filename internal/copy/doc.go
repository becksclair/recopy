// Package copy implements high-performance local file copying using
// kernel zero-copy mechanisms (reflink, copy_file_range, sendfile)
// with automatic fallback to buffered I/O when necessary.
//
// The copy engine attempts operations in this order:
// 1. FICLONE/FICLONERANGE (reflink) for instant CoW clones
// 2. copy_file_range for kernel-assisted zero-copy
// 3. sendfile for efficient data transfer
// 4. Buffered read/write as final fallback
//
// Metadata (permissions, timestamps, ownership) is preserved separately
// after data transfer completes.
package copy
