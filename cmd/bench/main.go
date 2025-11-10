package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	var (
		fileCount = flag.Int("files", 64, "number of data files")
		fileSize  = flag.Int("file-size-mib", 2, "size per file in MiB")
		parallel  = flag.Int("parallel", 1, "recopy --parallel value")
		profile   = flag.String("profile", "auto", "recopy --profile value")
		keep      = flag.Bool("keep", false, "keep benchmark workspace")
	)
	flag.Parse()

	repo := repoRoot()
	fmt.Printf("recopy bench: files=%d size=%dMiB parallel=%d profile=%s\n", *fileCount, *fileSize, *parallel, *profile)

	workspace := must(os.MkdirTemp("", "recopy-bench-"))
	if !*keep {
		defer os.RemoveAll(workspace)
	}

	src := filepath.Join(workspace, "src")
	ensure(os.MkdirAll(src, 0o755))
	ensure(seedDataset(src, *fileCount, *fileSize))

	cpDest := filepath.Join(workspace, "cp")
	recopyDest := filepath.Join(workspace, "recopy")
	ensure(os.MkdirAll(cpDest, 0o755))
	ensure(os.MkdirAll(recopyDest, 0o755))

	totalBytes := int64(*fileCount) * int64(*fileSize) * 1024 * 1024

	cpDuration := mustDuration(execDuration("cp", "-a", src, cpDest))
	fmt.Printf("cp -a duration: %s (%.2f MiB/s)\n", cpDuration, throughput(totalBytes, cpDuration))

	recopyBin := filepath.Join(repo, "recopy")
	if _, err := os.Stat(recopyBin); err != nil {
		fmt.Println("building recopy binary (go build ./cmd/recopy)...")
		build := exec.Command("go", "build", "./cmd/recopy")
		build.Dir = repo
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		ensure(build.Run())
	}

	recopyDuration := mustDuration(execDuration(recopyBin,
		fmt.Sprintf("--parallel=%d", *parallel),
		"--profile", *profile,
		src,
		recopyDest,
	))
	fmt.Printf("recopy duration: %s (%.2f MiB/s)\n", recopyDuration, throughput(totalBytes, recopyDuration))

	fmt.Printf("workspace: %s\n", workspace)
	if !*keep {
		fmt.Println("(use --keep to inspect files)")
	}
}

func seedDataset(root string, files, sizeMiB int) error {
	rand.Seed(time.Now().UnixNano())
	chunk := make([]byte, 1<<20)
	if _, err := rand.Read(chunk); err != nil {
		return err
	}
	for i := 0; i < files; i++ {
		sub := filepath.Join(root, fmt.Sprintf("dir-%02d", i%4))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			return err
		}
		path := filepath.Join(sub, fmt.Sprintf("file-%03d.bin", i))
		if err := writeRepeating(path, chunk, sizeMiB); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(root, "README.txt"), []byte("recopy benchmark dataset\n"), 0o644)
}

func writeRepeating(path string, block []byte, copies int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := 0; i < copies; i++ {
		if _, err := f.Write(block); err != nil {
			return err
		}
	}
	return f.Sync()
}

func execDuration(bin string, args ...string) (time.Duration, error) {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	start := time.Now()
	err := cmd.Run()
	return time.Since(start), err
}

func throughput(bytes int64, dur time.Duration) float64 {
	if dur <= 0 {
		return 0
	}
	mib := float64(bytes) / (1024 * 1024)
	return mib / dur.Seconds()
}

func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			panic("go.mod not found")
		}
		dir = next
	}
}

func ensure(err error) {
	if err != nil {
		panic(err)
	}
}

func must(path string, err error) string {
	if err != nil {
		panic(err)
	}
	return path
}

func mustDuration(d time.Duration, err error) time.Duration {
	if err != nil {
		panic(err)
	}
	return d
}
