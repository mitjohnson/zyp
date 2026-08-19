package collector

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	// Register SQLite driver.
	_ "modernc.org/sqlite"

	"zyp/internal/provider"
	"zyp/internal/workdir"
)

type SqliteCollector struct{}

func init() {
	Register(provider.KindSQLite, &SqliteCollector{})
}

func (s *SqliteCollector) Collect(ctx context.Context, t provider.Target, wd *workdir.WorkDir) (Dump, error) {
	if t.Kind != provider.KindSQLite {
		return Dump{}, fmt.Errorf("invalid target kind: %s", t.Kind)
	}

	con, err := sql.Open("sqlite", t.Source)

	if err != nil {
		return Dump{}, fmt.Errorf("failed to open sqlite database: %w", err)
	}
	defer func() {
		if closeErr := con.Close(); closeErr != nil {
			slog.Warn("failed to close sqlite connection", "error", closeErr)
		}
	}()

	dest, err := wd.Path(t.Name, filepath.Base(t.Source))
	if err != nil {
		return Dump{}, fmt.Errorf("failed to prepare scratch path: %w", err)
	}

	_, err = con.ExecContext(ctx, "VACUUM INTO ?", dest)

	if err != nil {
		return Dump{}, fmt.Errorf("failed to create sqlite backup: %w", err)
	}

	// Stamp the scratch copy with the source's own mtime to produce a stable mtime across runs.
	// This assists restics ability to detect changes and avoid rehashing unmodified files.
	srcInfo, err := os.Stat(t.Source)
	if err != nil {
		return Dump{}, fmt.Errorf("failed to stat sqlite source: %w", err)
	}

	if err := os.Chtimes(dest, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return Dump{}, fmt.Errorf("failed to set scratch file mtime: %w", err)
	}

	return Dump{Target: t, Path: dest}, nil
}
