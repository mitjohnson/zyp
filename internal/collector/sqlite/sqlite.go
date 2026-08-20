package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	// Register SQLite driver.
	_ "modernc.org/sqlite"

	"zyp/internal/collector"
	"zyp/internal/target"
	"zyp/internal/workdir"
)

type Collector struct{}

func init() {
	collector.Register(target.KindSQLite, &Collector{})
}

func (s *Collector) Collect(ctx context.Context, t target.Target, wd *workdir.WorkDir) (collector.Dump, error) {
	if t.Kind != target.KindSQLite {
		return collector.Dump{}, fmt.Errorf("invalid target kind: %s", t.Kind)
	}

	con, err := sql.Open("sqlite", t.Source)
	if err != nil {
		return collector.Dump{}, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	defer func() {
		if closeErr := con.Close(); closeErr != nil {
			slog.Warn("failed to close sqlite connection", "error", closeErr)
		}
	}()

	dest, err := wd.Path(t.Name, filepath.Base(t.Source))
	if err != nil {
		return collector.Dump{}, fmt.Errorf("failed to prepare scratch path: %w", err)
	}

	if _, err = con.ExecContext(ctx, "VACUUM INTO ?", dest); err != nil {
		return collector.Dump{}, fmt.Errorf("failed to create sqlite backup: %w", err)
	}

	// Stamp the scratch copy with the source's own mtime to produce a stable mtime across runs.
	// This assists restics ability to detect changes and avoid rehashing unmodified files.
	srcInfo, err := os.Stat(t.Source)
	if err != nil {
		return collector.Dump{}, fmt.Errorf("failed to stat sqlite source: %w", err)
	}

	if err := os.Chtimes(dest, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return collector.Dump{}, fmt.Errorf("failed to set scratch file mtime: %w", err)
	}

	return collector.Dump{Target: t, Path: dest}, nil
}
