package workdir

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
)

type WorkDir struct {
	root string
	lock *os.File
}

func Open(root string) (*WorkDir, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, fmt.Errorf("create work dir %s: %w", root, err)
	}

	//nolint:gosec // the "root" is generated from DefaultWorkDirPath generated in app/run.go
	lock, err := os.OpenFile(filepath.Join(root, ".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if closeErr := lock.Close(); closeErr != nil {
			slog.Warn("failed to close lock file after flock failure", "error", closeErr)
		}
		return nil, fmt.Errorf("encountered lockfile, aborting: %w", err)
	}

	return &WorkDir{root: root, lock: lock}, nil
}

func (w *WorkDir) Path(targetName, filename string) (string, error) {
	dir := filepath.Join(w.root, targetName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create scratch dir for %s: %w", targetName, err)
	}

	dest := filepath.Join(dir, filename)
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("clear stale file %s: %w", dest, err)
	}

	return dest, nil
}

func (w *WorkDir) Close() error {
	if entries, err := os.ReadDir(w.root); err == nil {
		for _, e := range entries {
			if e.Name() == ".lock" {
				continue
			}
			if err := os.RemoveAll(filepath.Join(w.root, e.Name())); err != nil {
				slog.Warn("failed to remove scratch directory", "path", filepath.Join(w.root, e.Name()), "error", err)
			}
		}
	}

	if err := syscall.Flock(int(w.lock.Fd()), syscall.LOCK_UN); err != nil {
		slog.Warn("failed to unlock work dir lock file", "error", err)
	}
	return w.lock.Close()
}
