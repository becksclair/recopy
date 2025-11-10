package ops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type fileInfo = os.FileInfo

func getFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func parentDir(path string) string {
	dir := filepath.Dir(path)
	if dir == "" {
		return "."
	}
	return dir
}

func mkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func runCommand(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("empty command")
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		return wrapCommandExit(args, err)
	case <-ctx.Done():
		interruptProcess(cmd.Process)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
		return ctx.Err()
	}
}

func interruptProcess(proc *os.Process) {
	if proc == nil {
		return
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		_ = proc.Kill()
	}
}

func wrapCommandExit(args []string, err error) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		if code == 0 {
			code = 1
		}
		copyArgs := append([]string(nil), args...)
		return CommandError{Name: args[0], Args: copyArgs, Code: code, Err: err}
	}
	return err
}
