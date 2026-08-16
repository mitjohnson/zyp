package rclone

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"

	"zyp/internal/collector"
	"zyp/internal/config"
	"zyp/internal/engine"
)

type Runner struct {
	Repository string
	Env        map[string]string
}

func init() {
	engine.Register("rclone", func(repo config.Repository) engine.Engine {
		return Runner{Repository: repo.Remote, Env: repo.Env}
	})
}

func compressFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", srcPath, err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", dstPath, err)
	}
	defer dst.Close()

	gz := gzip.NewWriter(dst)

	if _, err := io.Copy(gz, src); err != nil {
		return fmt.Errorf("compress %s: %w", srcPath, err)
	}

	if err := gz.Close(); err != nil {
		return fmt.Errorf("finalize gzip for %s: %w", srcPath, err)
	}

	return nil
}

func (r Runner) Backup(ctx context.Context, dumps []collector.Dump) error {
	rclonePath, err := exec.LookPath("rclone")
	if err != nil {
		return fmt.Errorf("rclone binary not found in $PATH: %w", err)
	}

	for _, dump := range dumps {
		localPath := dump.Path
		var compressedPath string

		if dump.Target.Compress {
			compressedPath = localPath + ".gz"
			if err := compressFile(localPath, compressedPath); err != nil {
				return fmt.Errorf("compress %s: %w", localPath, err)
			}
			localPath = compressedPath
		}

		dest := path.Join(r.Repository, dump.Target.Name)

		args := []string{"copy", localPath, dest}
		cmd := exec.CommandContext(ctx, rclonePath, args...)

		cmd.Env = os.Environ()
		for key, value := range r.Env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}

		cmd.Stdout = os.Stdout

		var stderr bytes.Buffer
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

		runErr := cmd.Run()

		if compressedPath != "" {
			if rmErr := os.Remove(compressedPath); rmErr != nil {
				slog.Warn("failed to remove temporary compressed file", "path", compressedPath, "error", rmErr)
			}
		}

		if runErr != nil {
			return fmt.Errorf("rclone copy failed for %s: %w: %s", localPath, runErr, stderr.String())
		}
	}

	return nil
}
