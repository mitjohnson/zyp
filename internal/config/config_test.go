package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		conf    Config
		wantErr bool
	}{
		{
			name: "valid config with default repository",
			conf: Config{
				DefaultRepository: "prod",
				Repositories: map[string]Repository{
					"prod": {Engine: "restic", Remote: "b2:bucket:prod"},
				},
			},
		},
		{
			name: "valid config without default repository",
			conf: Config{
				Repositories: map[string]Repository{
					"prod": {Engine: "restic", Remote: "b2:bucket:prod"},
				},
			},
		},
		{
			name: "multiple valid repositories",
			conf: Config{
				DefaultRepository: "prod",
				Repositories: map[string]Repository{
					"prod":     {Engine: "restic", Remote: "b2:bucket:prod"},
					"personal": {Engine: "restic", Remote: "b2:bucket:personal"},
				},
			},
		},
		{
			name:    "no repositories defined",
			conf:    Config{},
			wantErr: true,
		},
		{
			name: "default repository not found",
			conf: Config{
				DefaultRepository: "missing",
				Repositories: map[string]Repository{
					"prod": {Engine: "restic", Remote: "b2:bucket:prod"},
				},
			},
			wantErr: true,
		},
		{
			name: "unsupported engine",
			conf: Config{
				Repositories: map[string]Repository{
					"prod": {Engine: "borg", Remote: "b2:bucket:prod"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing remote",
			conf: Config{
				Repositories: map[string]Repository{
					"prod": {Engine: "restic"},
				},
			},
			wantErr: true,
		},
		{
			name: "one bad repository among valid ones",
			conf: Config{
				Repositories: map[string]Repository{
					"prod":     {Engine: "restic", Remote: "b2:bucket:prod"},
					"personal": {Engine: "restic"},
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.conf)
			if (err != nil) != test.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "zyp.yaml")
		yamlContent := `
defaultRepository: prod
repositories:
  prod:
    engine: restic
    repo: b2:my-bucket:prod
    env:
      RESTIC_PASSWORD: secret
`
		if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		conf, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if conf.DefaultRepository != "prod" {
			t.Errorf("DefaultRepository = %q, want %q", conf.DefaultRepository, "prod")
		}

		repo, ok := conf.Repositories["prod"]
		if !ok {
			t.Fatalf("expected repository %q to be present", "prod")
		}
		if repo.Remote != "b2:my-bucket:prod" {
			t.Errorf("Remote = %q, want %q", repo.Remote, "b2:my-bucket:prod")
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
		if err == nil {
			t.Error("Load() error = nil, want an error for a nonexistent file")
		}
	})

	t.Run("malformed yaml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "zyp.yaml")
		if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		_, err := Load(path)
		if err == nil {
			t.Error("Load() error = nil, want an error for malformed yaml")
		}
	})

	t.Run("valid yaml but invalid config", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "zyp.yaml")
		yamlContent := `
repositories:
  prod:
    engine: restic
`
		if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}

		_, err := Load(path)
		if err == nil {
			t.Error("Load() error = nil, want a validation error")
		}
	})
}
