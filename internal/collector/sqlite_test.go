package collector

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"zyp/internal/provider"
)

func createTestSqliteDB(path string) error {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("failed to create test sqlite database: %w", err)
	}

	if _, err := conn.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		conn.Close()
		return fmt.Errorf("failed to create test table: %w", err)
	}

	conn.Close()

	return nil
}

func TestSqliteCollector_Collect(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) provider.Target
		wantErr bool
	}{
		{
			name: "valid sqlite target",
			setup: func(t *testing.T) provider.Target {
				dbPath := filepath.Join(t.TempDir(), "test.db")
				if err := createTestSqliteDB(dbPath); err != nil {
					t.Fatalf("failed to create test sqlite database: %v", err)
				}
				return provider.Target{Name: "test-sqlite", Kind: provider.KindSQLite, Source: dbPath}
			},
		},
		{
			name: "wrong kind",
			setup: func(t *testing.T) provider.Target {
				return provider.Target{Name: "test-postgres", Kind: provider.KindPostgres}
			},
			wantErr: true,
		},
		{
			name: "source path does not exist",
			setup: func(t *testing.T) provider.Target {
				dbPath := filepath.Join(t.TempDir(), "missing-dir", "test.db")
				return provider.Target{Name: "test-sqlite", Kind: provider.KindSQLite, Source: dbPath}
			},
			wantErr: true,
		},
		{
			name: "destination already contains non-database content",
			setup: func(t *testing.T) provider.Target {
				dbPath := filepath.Join(t.TempDir(), "test.db")
				if err := createTestSqliteDB(dbPath); err != nil {
					t.Fatalf("failed to create test sqlite database: %v", err)
				}
				if err := os.WriteFile(dbPath+".backup", []byte("not a valid sqlite database"), 0644); err != nil {
					t.Fatalf("failed to create stale backup file: %v", err)
				}
				return provider.Target{Name: "test-sqlite", Kind: provider.KindSQLite, Source: dbPath}
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := test.setup(t)

			collector := &SqliteCollector{}
			got, err := collector.Collect(context.Background(), target)

			if (err != nil) != test.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, test.wantErr)
			}

			if !test.wantErr {
				wantPath := target.Source + ".backup"
				if got.Path != wantPath {
					t.Errorf("Collect() got path = %v, want %v", got.Path, wantPath)
				}
			}
		})
	}
}
