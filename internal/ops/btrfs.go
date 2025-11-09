package ops

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func (e Executor) handleBtrfsOffer(ctx context.Context, src, dest string, opts Options) error {
	if opts.BtrfsDecider == nil {
		fmt.Fprintf(opts.stdout(), "btrfs offer skipped (no prompt configured) for %s\n", src)
		return nil
	}
	accept, err := opts.BtrfsDecider(src, dest)
	if err != nil {
		return err
	}
	if !accept {
		fmt.Fprintf(opts.stdout(), "btrfs offer declined for %s\n", src)
		return nil
	}
	if err := btrfsReplicate(ctx, src, dest, opts.stdout(), opts.stderr()); err != nil {
		return err
	}
	e.completed[stepKey(src, dest)] = true
	return nil
}

func btrfsReplicate(ctx context.Context, src, dest string, stdout, stderr io.Writer) error {
	if _, err := exec.LookPath("btrfs"); err != nil {
		return fmt.Errorf("btrfs tool not found: %w", err)
	}
	info, err := statPath(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("btrfs source %s must be a directory or subvolume", src)
	}
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("destination %s already exists; remove it first", dest)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := ensureParentDir(dest); err != nil {
		return err
	}
	snapName := fmt.Sprintf(".recopy-snap-%d", time.Now().UnixNano())
	snapshotPath := filepath.Join(filepath.Dir(src), snapName)
	if err := runCommand(ctx, []string{"btrfs", "subvolume", "snapshot", "-r", src, snapshotPath}, stdout, stderr); err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}
	defer runCommand(ctx, []string{"btrfs", "subvolume", "delete", snapshotPath}, stdout, stderr)
	recvParent := parentDir(dest)
	recvPath := filepath.Join(recvParent, filepath.Base(snapshotPath))
	_ = os.RemoveAll(recvPath)
	if err := runBtrfsPipeline(ctx, snapshotPath, recvParent, stdout, stderr); err != nil {
		return err
	}
	if err := os.Rename(recvPath, dest); err != nil {
		return fmt.Errorf("finalize btrfs receive: %w", err)
	}
	return nil
}

func runBtrfsPipeline(ctx context.Context, snapshotPath, destParent string, stdout, stderr io.Writer) error {
	recvCmd := exec.CommandContext(ctx, "btrfs", "receive", destParent)
	sendCmd := exec.CommandContext(ctx, "btrfs", "send", snapshotPath)
	env := append(os.Environ(), "LC_ALL=C")
	recvCmd.Env = env
	sendCmd.Env = env
	pr, pw := io.Pipe()
	recvCmd.Stdin = pr
	recvCmd.Stdout = stdout
	recvCmd.Stderr = stderr
	sendCmd.Stdout = pw
	sendCmd.Stderr = stderr
	if err := recvCmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return err
	}
	if err := sendCmd.Start(); err != nil {
		recvCmd.Process.Kill()
		recvCmd.Wait()
		pr.Close()
		pw.Close()
		return err
	}
	sendErr := make(chan error, 1)
	recvErr := make(chan error, 1)
	go func() {
		err := sendCmd.Wait()
		pw.CloseWithError(err)
		sendErr <- err
	}()
	go func() {
		err := recvCmd.Wait()
		pr.CloseWithError(err)
		recvErr <- err
	}()
	if err := <-sendErr; err != nil {
		return err
	}
	if err := <-recvErr; err != nil {
		return err
	}
	return nil
}
