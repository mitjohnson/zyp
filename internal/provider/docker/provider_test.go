package docker

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"zyp/internal/target"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

type fakeContainerLister struct {
	containers []container.Summary
	err        error
}

func (f *fakeContainerLister) ContainerList(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
	return f.containers, f.err
}

func (f *fakeContainerLister) Ping(_ context.Context) (types.Ping, error) {
	if f.err != nil {
		return types.Ping{}, fmt.Errorf("Ping() = %v, want empty string", f.err)
	}
	return types.Ping{}, nil
}

func TestDockerProvider(t *testing.T) {
	tests := []struct {
		name        string
		containers  []container.Summary
		listErr     error
		wantTargets []target.Target
		wantErr     bool
	}{
		{
			name:       "no containers",
			containers: []container.Summary{},
			listErr:    nil,
			wantErr:    false,
		},
		{
			name: "container not opted in for backup",
			containers: []container.Summary{
				{
					Names:  []string{"/my-container"},
					ID:     "1234567890",
					Labels: map[string]string{"zyp.enable": "false", "zyp.kind": "postgres"},
				},
			},
			wantErr: false,
		},
		{
			name:    "Docker API Error",
			listErr: errors.New("BOOM"),
			wantErr: true,
		},
		{
			name: "prefixed container name",
			containers: []container.Summary{
				{
					Names:  []string{"/my-container"},
					ID:     "1234567890",
					Labels: map[string]string{"zyp.enable": "true", "zyp.kind": "postgres"},
				},
			},
			wantTargets: []target.Target{
				{
					Name: "my-container",
					Kind: target.KindPostgres,
				},
			},
			wantErr: false,
		},
		{
			name: "non-prefixed container name",
			containers: []container.Summary{
				{
					Names:  []string{"my-container"},
					ID:     "1234567890",
					Labels: map[string]string{"zyp.enable": "true", "zyp.kind": "postgres"},
				},
			},
			wantTargets: []target.Target{
				{
					Name: "my-container",
					Kind: target.KindPostgres,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid labels",
			containers: []container.Summary{
				{
					Names:  []string{"/my-container"},
					ID:     "1234567890",
					Labels: map[string]string{"zyp.enable": "true", "zyp.kind": "invalid-kind"},
				},
				{
					Names:  []string{"/my-container-2"},
					ID:     "0987654321",
					Labels: map[string]string{"zyp.enable": "true"},
				},
				{
					Names:  []string{"/my-container-3"},
					ID:     "1122334455",
					Labels: map[string]string{"zyp.enable": "true", "zyp.kind": "postgres"},
				},
			},
			wantTargets: []target.Target{
				{
					Name: "my-container-3",
					Kind: target.KindPostgres,
				},
			},
			wantErr: false,
		},
		{
			name: "multiple valid kinds",
			containers: []container.Summary{
				{
					Names:  []string{"/my-container"},
					ID:     "1234567890",
					Labels: map[string]string{"zyp.enable": "true", "zyp.kind": "postgres"},
				},
				{
					Names:  []string{"/my-container-2"},
					ID:     "0987654321",
					Labels: map[string]string{"zyp.enable": "true", "zyp.kind": "sqlite", "zyp.file-path": "/data/db.sqlite"},
					Mounts: []container.MountPoint{
						{
							Destination: "/data",
							Source:      "/mnt/data",
						},
					},
				},
				{
					Names:  []string{"/my-container-3"},
					ID:     "1122334455",
					Labels: map[string]string{"zyp.enable": "true"},
				},
			},
			wantTargets: []target.Target{
				{
					Name: "my-container",
					Kind: target.KindPostgres,
				},
				{
					Name:   "my-container-2",
					Kind:   target.KindSQLite,
					Source: "/mnt/data/db.sqlite",
				},
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &Provider{
				cli: &fakeContainerLister{
					containers: test.containers,
					err:        test.listErr,
				},
			}

			got, err := p.Discover(context.Background())

			if (err != nil) != test.wantErr {
				t.Errorf("Discover() returned error: %v", err)
			}

			if !reflect.DeepEqual(got, test.wantTargets) {
				t.Errorf("Discover() = %v, want %v", got, test.wantTargets)
			}
		})
	}
}
