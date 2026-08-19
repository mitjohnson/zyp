package restic

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zyp/internal/collector"
	"zyp/internal/engine"
	"zyp/internal/provider"
)

// TODO: Find better way to test without needing a shell script fake
const fakeBinaryScript = `#!/bin/sh
if [ -n "$HELPER_LOG" ]; then
  echo "$@" >> "$HELPER_LOG"
fi
if [ -n "$HELPER_EXIT_CODE" ]; then
  exit "$HELPER_EXIT_CODE"
fi
exit 0
`

func writeFakeBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-restic")
	if err := os.WriteFile(path, []byte(fakeBinaryScript), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	return path
}

func newRunner(t *testing.T, repository string) Runner {
	t.Helper()
	return Runner{engine.Runner{
		Repository: repository,
		BinPath:    writeFakeBinary(t),
	}}
}

func TestBackup(t *testing.T) {
	tests := []struct {
		name            string
		dumps           []collector.Dump
		exitCode        string
		wantErr         bool
		wantErrContains string
		wantCalls       []string
	}{
		{
			name: "invokes restic once with all paths",
			dumps: []collector.Dump{
				{Target: provider.Target{Name: "alpha"}, Path: "/tmp/alpha.dump"},
				{Target: provider.Target{Name: "beta"}, Path: "/tmp/beta.dump"},
			},
			wantCalls: []string{
				"backup --ignore-inode /tmp/alpha.dump /tmp/beta.dump -r s3:bucket/repo",
			},
		},
		{
			name: "wraps command error",
			dumps: []collector.Dump{
				{Target: provider.Target{Name: "alpha"}, Path: "/tmp/alpha.dump"},
			},
			exitCode:        "1",
			wantErr:         true,
			wantErrContains: "restic backup",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callLog := filepath.Join(t.TempDir(), "calls.log")
			t.Setenv("HELPER_LOG", callLog)
			if test.exitCode != "" {
				t.Setenv("HELPER_EXIT_CODE", test.exitCode)
			}

			r := newRunner(t, "s3:bucket/repo")

			err := r.Backup(context.Background(), test.dumps)

			if (err != nil) != test.wantErr {
				t.Fatalf("Backup() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), test.wantErrContains)) {
				t.Errorf("expected error to contain %q, got: %v", test.wantErrContains, err)
			}

			if test.wantCalls != nil {
				logged, readErr := os.ReadFile(callLog)
				if readErr != nil {
					t.Fatalf("read call log: %v", readErr)
				}
				calls := strings.Split(strings.TrimSpace(string(logged)), "\n")
				if len(calls) != len(test.wantCalls) {
					t.Fatalf("expected %d restic invocation(s), got %d:\n%s", len(test.wantCalls), len(calls), logged)
				}
				for i, want := range test.wantCalls {
					if calls[i] != want {
						t.Errorf("call %d: got %q, want %q", i, calls[i], want)
					}
				}
			}
		})
	}
}
