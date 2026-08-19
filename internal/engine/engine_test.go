package engine

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TODO: Find better way to test without needing a shell script fake
const fakeBinaryScript = `#!/bin/sh
if [ -n "$HELPER_LOG" ]; then
  {
    echo "ARGS:$@"
    echo "ENV_MY_TEST_VAR:$MY_TEST_VAR"
  } >> "$HELPER_LOG"
fi
if [ -n "$HELPER_EXIT_CODE" ]; then
  echo "boom" >&2
  exit "$HELPER_EXIT_CODE"
fi
exit 0
`

func writeFakeBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-bin")
	if err := os.WriteFile(path, []byte(fakeBinaryScript), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	return path
}

func TestRunCommand(t *testing.T) {
	tests := []struct {
		name            string
		emptyBinPath    bool
		args            []string
		env             map[string]string
		exitCode        string
		wantErr         bool
		wantErrContains string
		wantLogContains []string
	}{
		{
			name:            "empty bin path fails fast",
			emptyBinPath:    true,
			wantErr:         true,
			wantErrContains: "binary not found",
		},
		{
			name:            "passes args and env through to the command",
			args:            []string{"foo", "bar"},
			env:             map[string]string{"MY_TEST_VAR": "hello"},
			wantLogContains: []string{"ARGS:foo bar", "ENV_MY_TEST_VAR:hello"},
		},
		{
			name:            "wraps failure with stderr",
			exitCode:        "1",
			wantErr:         true,
			wantErrContains: "boom",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			callLog := filepath.Join(t.TempDir(), "calls.log")
			t.Setenv("HELPER_LOG", callLog)
			if test.exitCode != "" {
				t.Setenv("HELPER_EXIT_CODE", test.exitCode)
			}

			r := Runner{Env: test.env}
			if !test.emptyBinPath {
				r.BinPath = writeFakeBinary(t)
			}

			err := RunCommand(context.Background(), r, test.args)

			if (err != nil) != test.wantErr {
				t.Fatalf("RunCommand() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), test.wantErrContains)) {
				t.Errorf("expected error to contain %q, got: %v", test.wantErrContains, err)
			}

			for _, want := range test.wantLogContains {
				logged, readErr := os.ReadFile(callLog)
				if readErr != nil {
					t.Fatalf("read call log: %v", readErr)
				}
				if !strings.Contains(string(logged), want) {
					t.Errorf("expected %q in call log, got:\n%s", want, logged)
				}
			}
		})
	}
}

func TestRunCommand_ConcurrentCallsDoNotRace(t *testing.T) {
	r := Runner{BinPath: writeFakeBinary(t)}

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for range 10 {
		wg.Go(func() {
			errs <- RunCommand(context.Background(), r, []string{"concurrent"})
		})
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("RunCommand() error = %v", err)
		}
	}
}

func TestCompress(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		missingSrc bool
		wantErr    bool
	}{
		{
			name:    "compresses and preserves content",
			content: "the quick brown fox jumps over the lazy dog",
		},
		{
			name:       "missing source file returns error",
			missingSrc: true,
			wantErr:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "source.txt")
			if !test.missingSrc {
				if err := os.WriteFile(src, []byte(test.content), 0o644); err != nil {
					t.Fatalf("write source file: %v", err)
				}
			}
			dst := filepath.Join(dir, "source.txt.gz")

			err := Compress(src, dst)
			if (err != nil) != test.wantErr {
				t.Fatalf("Compress() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}

			f, err := os.Open(dst)
			if err != nil {
				t.Fatalf("open compressed file: %v", err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					t.Errorf("close compressed file: %v", err)
				}
			}()

			gz, err := gzip.NewReader(f)
			if err != nil {
				t.Fatalf("gzip.NewReader() error = %v", err)
			}
			defer func() {
				if err := gz.Close(); err != nil {
					t.Errorf("close gzip reader: %v", err)
				}
			}()

			var got bytes.Buffer
			const maxDecompressedSize = 10 << 20 // test fixtures are tiny; this just bounds the read
			if _, err := io.Copy(&got, io.LimitReader(gz, maxDecompressedSize)); err != nil {
				t.Fatalf("decompress: %v", err)
			}
			if got.String() != test.content {
				t.Errorf("decompressed content = %q, want %q", got.String(), test.content)
			}
		})
	}
}
