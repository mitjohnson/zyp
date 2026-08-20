package rclone

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zyp/internal/collector"
	"zyp/internal/engine"
	"zyp/internal/target"
)

// TODO: Find better way to test without needing a shell script fake
const fakeBinaryScript = `#!/bin/sh
if [ -n "$HELPER_DONE_LOG" ]; then
  echo "$@" >> "$HELPER_DONE_LOG"
fi
exit 0
`

func writeFakeBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-rclone")
	if err := os.WriteFile(path, []byte(fakeBinaryScript), 0o755); err != nil {
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

func newDump(t *testing.T, dir, name string, compress bool) collector.Dump {
	t.Helper()
	path := filepath.Join(dir, name+".dump")
	if err := os.WriteFile(path, []byte("data-"+name), 0o644); err != nil {
		t.Fatalf("write dump file: %v", err)
	}
	return collector.Dump{
		Target: target.Target{Name: name, Compress: compress},
		Path:   path,
	}
}

func TestBackup_CopiesEachDump(t *testing.T) {
	doneLog := filepath.Join(t.TempDir(), "done.log")
	t.Setenv("HELPER_DONE_LOG", doneLog)

	r := newRunner(t, "remote:bucket")
	dir := t.TempDir()
	dumps := []collector.Dump{
		newDump(t, dir, "alpha", false),
		newDump(t, dir, "beta", false),
	}

	if err := r.Backup(context.Background(), dumps); err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	logged, err := os.ReadFile(doneLog)
	if err != nil {
		t.Fatalf("read done log: %v", err)
	}
	for _, d := range dumps {
		want := "copy " + d.Path + " remote:bucket/" + d.Target.Name
		if !strings.Contains(string(logged), want) {
			t.Errorf("expected call %q not found in log:\n%s", want, logged)
		}
	}
}

func TestBackup_CompressesWhenRequested(t *testing.T) {
	doneLog := filepath.Join(t.TempDir(), "done.log")
	t.Setenv("HELPER_DONE_LOG", doneLog)

	r := newRunner(t, "remote:bucket")
	dir := t.TempDir()
	d := newDump(t, dir, "alpha", true)

	if err := r.Backup(context.Background(), []collector.Dump{d}); err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	logged, err := os.ReadFile(doneLog)
	if err != nil {
		t.Fatalf("read done log: %v", err)
	}
	if !strings.Contains(string(logged), d.Path+".gz") {
		t.Errorf("expected compressed path in call log, got:\n%s", logged)
	}
	if _, err := os.Stat(d.Path + ".gz"); !os.IsNotExist(err) {
		t.Errorf("expected temporary compressed file to be removed, stat err = %v", err)
	}
}
