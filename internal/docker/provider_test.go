package docker

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"zyp/internal/provider"

	"github.com/docker/docker/api/types/container"
)

type fakeContainerLister struct {
	containers []container.Summary
	err        error
}

func (f *fakeContainerLister) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	return f.containers, f.err
}

func TestDockerProvider(t *testing.T) {
	tests := []struct {
		name        string
		containers  []container.Summary
		listErr     error
		wantTargets []provider.Target
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
					Labels: map[string]string{"backup.enable": "false", "backup.kind": "postgres"},
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
					Labels: map[string]string{"backup.enable": "true", "backup.kind": "postgres"},
				},
			},
			wantTargets: []provider.Target{
				{
					Name:         "my-container",
					ContainerRef: "1234567890",
					Kind:         provider.KindPostgres,
					Labels:       map[string]string{"backup.enable": "true", "backup.kind": "postgres"},
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
					Labels: map[string]string{"backup.enable": "true", "backup.kind": "postgres"},
				},
			},
			wantTargets: []provider.Target{
				{
					Name:         "my-container",
					ContainerRef: "1234567890",
					Kind:         provider.KindPostgres,
					Labels:       map[string]string{"backup.enable": "true", "backup.kind": "postgres"},
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
					Labels: map[string]string{"backup.enable": "true", "backup.kind": "invalid-kind"},
				},
				{
					Names:  []string{"/my-container-2"},
					ID:     "0987654321",
					Labels: map[string]string{"backup.enable": "true"},
				},
				{
					Names:  []string{"/my-container-3"},
					ID:     "1122334455",
					Labels: map[string]string{"backup.enable": "true", "backup.kind": "postgres"},
				},
			},
			wantTargets: []provider.Target{
				{
					Name:         "my-container-3",
					ContainerRef: "1122334455",
					Kind:         provider.KindPostgres,
					Labels:       map[string]string{"backup.enable": "true", "backup.kind": "postgres"},
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
					Labels: map[string]string{"backup.enable": "true", "backup.kind": "postgres"},
				},
				{
					Names:  []string{"/my-container-2"},
					ID:     "0987654321",
					Labels: map[string]string{"backup.enable": "true", "backup.kind": "sqlite", "backup.path": "/data/db.sqlite"},
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
					Labels: map[string]string{"backup.enable": "true"},
				},
			},
			wantTargets: []provider.Target{
				{
					Name:         "my-container",
					ContainerRef: "1234567890",
					Kind:         provider.KindPostgres,
					Labels:       map[string]string{"backup.enable": "true", "backup.kind": "postgres"},
				},
				{
					Name:         "my-container-2",
					ContainerRef: "0987654321",
					Kind:         provider.KindSQLite,
					Labels:       map[string]string{"backup.enable": "true", "backup.kind": "sqlite", "backup.path": "/data/db.sqlite"},
					Source:       "/mnt/data/db.sqlite",
				},
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := NewDockerProvider(&fakeContainerLister{
				containers: test.containers,
				err:        test.listErr,
			})

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
