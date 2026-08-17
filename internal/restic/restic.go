package restic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"zyp/internal/collector"
	"zyp/internal/config"
	"zyp/internal/engine"
)

type Runner struct {
	Repository string
	Env        map[string]string
}

func init() {
	engine.Register("restic", func(repo config.Repository) engine.Engine {
		return Runner{Repository: repo.Remote, Env: repo.Env}
	})
}

func (r Runner) Backup(ctx context.Context, dumps []collector.Dump) error {
	resticPath, err := exec.LookPath("restic")

	if err != nil {
		return fmt.Errorf("restic binary not found in $PATH: %w", err)
	}

	paths := make([]string, len(dumps))
	for i, dump := range dumps {
		paths[i] = dump.Path
	}

	// collecters can modify the mtime of the file in the working dir to match the source
	// however other metadata like inode and ctime will always be different.
	// --ignore-inode is used to avoid restic rehashing the file every time
	args := append([]string{"backup", "--ignore-inode"}, paths...)
	args = append(args, "-r", r.Repository)

	cmd := exec.CommandContext(ctx, resticPath, args...)
	cmd.Env = os.Environ()

	for key, value := range r.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	cmd.Stdout = os.Stdout

	var stderr bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("restic backup failed: %w: %s", err, stderr.String())
	}

	return nil
}
