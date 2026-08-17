package collector

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"zyp/internal/provider"
)

type SqliteCollector struct{}

func init() {
	Register(provider.KindSQLite, &SqliteCollector{})
}

func (s *SqliteCollector) Collect(ctx context.Context, t provider.Target) (Dump, error) {
	if t.Kind != provider.KindSQLite {
		return Dump{}, fmt.Errorf("invalid target kind: %s", t.Kind)
	}

	con, err := sql.Open("sqlite", t.Source)

	if err != nil {
		return Dump{}, fmt.Errorf("failed to open sqlite database: %w", err)
	}
	defer con.Close()
	
	tmpDir, err := os.MkdirTemp("", "zyp-"+t.Name+"-")
	if err != nil {
		return Dump{}, fmt.Errorf("failed to create temporary directory: %w", err)
	}
	dest := filepath.Join(tmpDir, filepath.Base(t.Source)+".backup")

	_, err = con.ExecContext(ctx, "VACUUM INTO ?", dest)

	if err != nil {
		return Dump{}, fmt.Errorf("failed to create sqlite backup: %w", err)
	}

	return Dump{Target: t, Path: dest}, nil
}
