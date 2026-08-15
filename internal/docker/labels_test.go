package docker

import (
	"reflect"
	"testing"

	"zyp/internal/provider"

	"github.com/docker/docker/api/types/container"
)

func TestParseLabels(t *testing.T) {
	t.Run("not opted in", func(t *testing.T) {
		_, ok, err := parseLabels("test-container", "test-id", map[string]string{}, nil)
		if ok {
			t.Errorf("ok = %v, want false", ok)
		}
		if err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})

	t.Run("unknown backup.kind errors", func(t *testing.T) {
		_, ok, err := parseLabels("test-container", "test-id", map[string]string{
			"backup.enable": "true",
			"backup.kind":   "mongodb",
		}, nil)
		if ok {
			t.Errorf("ok = %v, want false", ok)
		}
		if err == nil {
			t.Error("err = nil, want an error")
		}
	})

	t.Run("sqlite", func(t *testing.T) {
		tests := []struct {
			name       string
			labels     map[string]string
			mounts     []container.MountPoint
			wantOk     bool
			wantErr    bool
			wantTarget provider.Target
		}{
			{
				name: "missing backup.path errors",
				labels: map[string]string{
					"backup.enable": "true",
					"backup.kind":   "sqlite",
				},
				wantOk:  false,
				wantErr: true,
			},
			{
				name: "resolves host path via matching mount",
				labels: map[string]string{
					"backup.enable": "true",
					"backup.kind":   "sqlite",
					"backup.path":   "/data/db.sqlite3",
				},
				mounts: []container.MountPoint{
					{Destination: "/data", Source: "/srv/prod/vaultwarden/data"},
				},
				wantOk: true,
				wantTarget: provider.Target{
					Name:         "test-container",
					Kind:         provider.KindSQLite,
					Source:       "/srv/prod/vaultwarden/data/db.sqlite3",
					ContainerRef: "test-id",
					Labels: map[string]string{
						"backup.enable": "true",
						"backup.kind":   "sqlite",
						"backup.path":   "/data/db.sqlite3",
					},
				},
			},
			{
				name: "backup.name overrides container name",
				labels: map[string]string{
					"backup.enable": "true",
					"backup.kind":   "sqlite",
					"backup.path":   "/data/db.sqlite3",
					"backup.name":   "vaultwarden-db",
				},
				mounts: []container.MountPoint{
					{Destination: "/data", Source: "/srv/prod/vaultwarden/data"},
				},
				wantOk: true,
				wantTarget: provider.Target{
					Name:         "vaultwarden-db",
					Kind:         provider.KindSQLite,
					Source:       "/srv/prod/vaultwarden/data/db.sqlite3",
					ContainerRef: "test-id",
					Labels: map[string]string{
						"backup.enable": "true",
						"backup.kind":   "sqlite",
						"backup.path":   "/data/db.sqlite3",
						"backup.name":   "vaultwarden-db",
					},
				},
			},
			{
				name: "no matching mount errors",
				labels: map[string]string{
					"backup.enable": "true",
					"backup.kind":   "sqlite",
					"backup.path":   "/data/db.sqlite3",
				},
				mounts: []container.MountPoint{
					{Destination: "/other", Source: "/srv/prod/other/data"},
				},
				wantOk:  false,
				wantErr: true,
			},
			{
				name: "mount destination prefix boundary is respected",
				labels: map[string]string{
					"backup.enable": "true",
					"backup.kind":   "sqlite",
					"backup.path":   "/data2/db.sqlite3",
				},
				mounts: []container.MountPoint{
					{Destination: "/data", Source: "/srv/prod/vaultwarden/data"},
				},
				wantOk:  false,
				wantErr: true,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				got, ok, err := parseLabels("test-container", "test-id", test.labels, test.mounts)

				if ok != test.wantOk {
					t.Errorf("ok = %v, want %v", ok, test.wantOk)
				}
				if (err != nil) != test.wantErr {
					t.Errorf("err = %v, wantErr %v", err, test.wantErr)
				}
				if test.wantOk && !reflect.DeepEqual(got, test.wantTarget) {
					t.Errorf("target = %+v, want %+v", got, test.wantTarget)
				}
			})
		}
	})

	t.Run("postgres", func(t *testing.T) {
		tests := []struct {
			name       string
			labels     map[string]string
			wantOk     bool
			wantTarget provider.Target
		}{
			{
				name: "opted in, no path needed",
				labels: map[string]string{
					"backup.enable": "true",
					"backup.kind":   "postgres",
				},
				wantOk: true,
				wantTarget: provider.Target{
					Name:         "test-container",
					Kind:         provider.KindPostgres,
					ContainerRef: "test-id",
					Labels: map[string]string{
						"backup.enable": "true",
						"backup.kind":   "postgres",
					},
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				got, ok, err := parseLabels("test-container", "test-id", test.labels, nil)

				if ok != test.wantOk {
					t.Errorf("ok = %v, want %v", ok, test.wantOk)
				}
				if err != nil {
					t.Errorf("err = %v, want nil", err)
				}
				if test.wantOk && !reflect.DeepEqual(got, test.wantTarget) {
					t.Errorf("target = %+v, want %+v", got, test.wantTarget)
				}
			})
		}
	})
}
