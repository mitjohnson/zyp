package collector

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"zyp/internal/provider"
	"zyp/internal/workdir"
)

func createTestSqliteDB(path string) (err error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("failed to create test sqlite database: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close test sqlite connection: %w", closeErr)
		}
	}()

	if _, err := conn.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		return fmt.Errorf("failed to create test table: %w", err)
	}

	return nil
}

func TestSqliteCollector_Collect(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, wd *workdir.WorkDir) provider.Target
		wantErr bool
	}{
		{
			name: "valid sqlite target",
			setup: func(t *testing.T, _ *workdir.WorkDir) provider.Target {
				dbPath := filepath.Join(t.TempDir(), "test.db")
				if err := createTestSqliteDB(dbPath); err != nil {
					t.Fatalf("failed to create test sqlite database: %v", err)
				}
				return provider.Target{Name: "test-sqlite", Kind: provider.KindSQLite, Source: dbPath}
			},
		},
		{
			name: "wrong kind",
			setup: func(_ *testing.T, _ *workdir.WorkDir) provider.Target {
				return provider.Target{Name: "test-postgres", Kind: provider.KindPostgres}
			},
			wantErr: true,
		},
		{
			name: "source path does not exist",
			setup: func(t *testing.T, _ *workdir.WorkDir) provider.Target {
				dbPath := filepath.Join(t.TempDir(), "missing-dir", "test.db")
				return provider.Target{Name: "test-sqlite", Kind: provider.KindSQLite, Source: dbPath}
			},
			wantErr: true,
		},
		{
			name: "stale scratch file from a previous run is cleared, not treated as an error",
			setup: func(t *testing.T, wd *workdir.WorkDir) provider.Target {
				dbPath := filepath.Join(t.TempDir(), "test.db")
				if err := createTestSqliteDB(dbPath); err != nil {
					t.Fatalf("failed to create test sqlite database: %v", err)
				}

				target := provider.Target{Name: "test-sqlite", Kind: provider.KindSQLite, Source: dbPath}

				stale, err := wd.Path(target.Name, filepath.Base(dbPath))
				if err != nil {
					t.Fatalf("failed to prepare stale destination: %v", err)
				}
				if err := os.WriteFile(stale, []byte("not a valid sqlite database"), 0644); err != nil {
					t.Fatalf("failed to create stale backup file: %v", err)
				}

				return target
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			wd, err := workdir.Open(root)
			if err != nil {
				t.Fatalf("failed to open workdir: %v", err)
			}
			defer func() {
				if err := wd.Close(); err != nil {
					t.Errorf("close workdir: %v", err)
				}
			}()

			target := test.setup(t, wd)

			collector := &SqliteCollector{}
			got, err := collector.Collect(context.Background(), target, wd)

			if (err != nil) != test.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, test.wantErr)
			}

			if !test.wantErr {
				wantPath := filepath.Join(root, target.Name, filepath.Base(target.Source))
				if got.Path != wantPath {
					t.Errorf("Collect() got path = %v, want %v", got.Path, wantPath)
				}
			}
		})
	}
}
