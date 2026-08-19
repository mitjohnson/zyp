package restic

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"zyp/internal/collector"
	"zyp/internal/config"
	"zyp/internal/engine"
)

func init() {
	engine.Register("restic", func(repo config.Repository) engine.Engine {
		resticPath, err := exec.LookPath("restic")
		if err != nil {
			slog.Warn("restic binary not found in $PATH, restic engine will not be available", "error", err)
		}
		return Runner{
			engine.Runner{
				Repository: repo.Remote,
				Env:        repo.Env,
				BinPath:    resticPath,
			},
		}
	})
}

type Runner struct {
	engine.Runner
}

func (r Runner) Backup(ctx context.Context, dumps []collector.Dump) error {

	// restic backup handles concurrency, so we can just pass all the paths at once.
	paths := make([]string, len(dumps))
	for i, dump := range dumps {
		paths[i] = dump.Path
	}

	// collecters can modify the mtime of the file in the working dir to match the source
	// however other metadata like inode and ctime will always be different.
	// --ignore-inode is used to avoid restic rehashing the file every time
	args := append([]string{"backup", "--ignore-inode"}, paths...)
	args = append(args, "-r", r.Repository)

	if err := engine.RunCommand(ctx, r.Runner, args); err != nil {
		return fmt.Errorf("restic backup %v to %s: %w", paths, r.Repository, err)
	}

	return nil
}
