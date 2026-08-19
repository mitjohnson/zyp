package rclone

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"

	"golang.org/x/sync/errgroup"

	"zyp/internal/collector"
	"zyp/internal/config"
	"zyp/internal/engine"
)

// how many rclone copy processes run at once.
const maxConcurrentCopies = 8

func init() {
	engine.Register("rclone", func(repo config.Repository) engine.Engine {
		rclonePath, err := exec.LookPath("rclone")
		if err != nil {
			slog.Warn("rclone binary not found in $PATH, rclone engine will not be available", "error", err)
		}

		return Runner{
			engine.Runner{
				Repository: repo.Remote,
				Env:        repo.Env,
				BinPath:    rclonePath,
			},
		}
	})
}

type Runner struct {
	engine.Runner
}

func (r Runner) Backup(ctx context.Context, dumps []collector.Dump) error {
	var g errgroup.Group
	g.SetLimit(maxConcurrentCopies)

	for _, dump := range dumps {
		g.Go(func() error {
			return r.processDump(ctx, dump)
		})
	}

	return g.Wait()
}

func (r Runner) processDump(ctx context.Context, dump collector.Dump) error {
	localPath := dump.Path

	if dump.Target.Compress {
		compressedPath := localPath + ".gz"
		if err := engine.Compress(localPath, compressedPath); err != nil {
			return fmt.Errorf("compress %s: %w", localPath, err)
		}
		localPath = compressedPath
		defer func() {
			if rmErr := os.Remove(compressedPath); rmErr != nil {
				slog.Warn("failed to remove temporary compressed file", "path", compressedPath, "error", rmErr)
			}
		}()
	}

	dest := path.Join(r.Repository, dump.Target.Name)
	args := []string{"copy", localPath, dest}
	if err := engine.RunCommand(ctx, r.Runner, args); err != nil {
		return fmt.Errorf("rclone copy %s to %s: %w", localPath, dest, err)
	}

	return nil
}
