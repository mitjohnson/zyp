package engine

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"zyp/internal/collector"
	"zyp/internal/config"
)

type Runner struct {
	Repository string
	Env        map[string]string
	BinPath    string
}

type Engine interface {
	Backup(ctx context.Context, dumps []collector.Dump) error
}

type Constructor func(repo config.Repository) Engine

var registry = map[string]Constructor{}

func Register(name string, constructor Constructor) {
	registry[name] = constructor
}

func Registered() map[string]Constructor {
	return registry
}

var runCommandMutex sync.Mutex

func RunCommand(ctx context.Context, r Runner, args []string) error {
	if r.BinPath == "" {
		return fmt.Errorf("binary not found in $PATH")
	}

	//nolint:gosec
	// Bin path is generated from implementatons init cycle
	// args are either hardcoded or non-externally generated file locations.
	// risks are limited to general $PATH risks.
	cmd := exec.CommandContext(ctx, r.BinPath, args...)

	cmd.Env = os.Environ()
	for key, value := range r.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// Locking stdout and stderr writes to avoid interleaving output from concurrent commands
	runCommandMutex.Lock()
	if _, err := os.Stdout.Write(stdout.Bytes()); err != nil {
		slog.Warn("failed to write to stdout", "error", err)
	}

	if _, stdErrErr := os.Stderr.Write(stderr.Bytes()); stdErrErr != nil {
		slog.Warn("failed to write to stderr", "error", stdErrErr)
	}
	runCommandMutex.Unlock()

	if runErr != nil {
		return fmt.Errorf("command failed for %s: %w: %s", r.BinPath, runErr, stderr.String())
	}

	return nil
}

func Compress(srcPath, dstPath string) (err error) {
	src, err := os.Open(srcPath) //nolint:gosec // srcPath is derived from the workdir/config, not external input
	if err != nil {
		return fmt.Errorf("open %s: %w", srcPath, err)
	}
	defer func() {
		if cerr := src.Close(); cerr != nil {
			slog.Warn("failed to close source file", "path", srcPath, "error", cerr)
		}
	}()

	dst, err := os.Create(dstPath) //nolint:gosec // dstPath is derived from the workdir/config, not external input
	if err != nil {
		return fmt.Errorf("create %s: %w", dstPath, err)
	}
	defer func() {
		if cerr := dst.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close %s: %w", dstPath, cerr)
		}
	}()

	gz := gzip.NewWriter(dst)

	if _, err := io.Copy(gz, src); err != nil {
		return fmt.Errorf("compress %s: %w", srcPath, err)
	}

	if err := gz.Close(); err != nil {
		return fmt.Errorf("finalize gzip for %s: %w", srcPath, err)
	}

	return nil
}
