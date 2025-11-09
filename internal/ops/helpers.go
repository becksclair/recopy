package ops

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	return cmd.Run()
}
