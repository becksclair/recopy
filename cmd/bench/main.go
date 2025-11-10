package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	var (
		fileCount  = flag.Int("files", 64, "number of data files")
		fileSize   = flag.Int("file-size-mib", 2, "size per file in MiB")
		parallel   = flag.Int("parallel", 1, "recopy --parallel value")
		profile    = flag.String("profile", "auto", "recopy --profile value")
		keep       = flag.Bool("keep", false, "keep benchmark workspace")
		warmup     = flag.Int("warmup", 1, "hyperfine warmup runs")
		runs       = flag.Int("runs", 5, "hyperfine run count")
		exportMD   = flag.String("export-md", "", "path for hyperfine Markdown (default workspace/hyperfine.md)")
		exportJSON = flag.String("export-json", "", "path for hyperfine JSON (default workspace/hyperfine.json)")
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

	hyperfineExports := resolveExports(workspace, *exportMD, *exportJSON)
	runHyperfine(hyperfineOptions{
		Src:        src,
		CpDest:     cpDest,
		RecopyDest: recopyDest,
		RecopyBin:  recopyBin,
		Parallel:   *parallel,
		Profile:    *profile,
		Runs:       *runs,
		Warmup:     *warmup,
		ExportMD:   hyperfineExports.markdown,
		ExportJSON: hyperfineExports.json,
	})

	fmt.Printf("workspace: %s\n", workspace)
	if hyperfineExports.markdown != "" {
		fmt.Printf("markdown report: %s\n", hyperfineExports.markdown)
	}
	if hyperfineExports.json != "" {
		fmt.Printf("json report: %s\n", hyperfineExports.json)
	}
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

type hyperfineOptions struct {
	Src        string
	CpDest     string
	RecopyDest string
	RecopyBin  string
	Parallel   int
	Profile    string
	Runs       int
	Warmup     int
	ExportMD   string
	ExportJSON string
}

type exportPaths struct {
	markdown string
	json     string
}

func resolveExports(workspace, md, jsonPath string) exportPaths {
	result := exportPaths{}
	if md == "" {
		result.markdown = filepath.Join(workspace, "hyperfine.md")
	} else if md != "-" {
		result.markdown = md
	}
	if jsonPath == "" {
		result.json = filepath.Join(workspace, "hyperfine.json")
	} else if jsonPath != "-" {
		result.json = jsonPath
	}
	return result
}

func runHyperfine(opts hyperfineOptions) {
	if _, err := exec.LookPath("hyperfine"); err != nil {
		panic("hyperfine not found in PATH")
	}
	prepare := fmt.Sprintf("rm -rf %s %s && mkdir -p %s %s",
		shellQuote(opts.CpDest), shellQuote(opts.RecopyDest), shellQuote(opts.CpDest), shellQuote(opts.RecopyDest))
	cpCmd := fmt.Sprintf("cp -a %s %s", shellQuote(opts.Src), shellQuote(opts.CpDest))
	recopyCmd := fmt.Sprintf("%s --parallel=%d --profile %s --no-ui %s %s",
		shellQuote(opts.RecopyBin), opts.Parallel, shellQuote(opts.Profile), shellQuote(opts.Src), shellQuote(opts.RecopyDest))
	hfArgs := []string{
		fmt.Sprintf("--runs=%d", opts.Runs),
		fmt.Sprintf("--warmup=%d", opts.Warmup),
		"--style", "color",
		"--prepare", prepare,
	}
	if opts.ExportMD != "" {
		hfArgs = append(hfArgs, "--export-markdown", opts.ExportMD)
	}
	if opts.ExportJSON != "" {
		hfArgs = append(hfArgs, "--export-json", opts.ExportJSON)
	}
	hfArgs = append(hfArgs,
		"-n", "cp -a", cpCmd,
		"-n", "recopy", recopyCmd,
	)
	cmd := exec.Command("hyperfine", hfArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ensure(cmd.Run())
}

func shellQuote(input string) string {
	if input == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(input, "'", "'\"'\"'") + "'"
}
