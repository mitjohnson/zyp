package docker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"zyp/internal/provider"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v3"
)

type DockerProvider struct {
	cli ContainerLister
}

func init() {
	provider.Register("docker", NewProvider)
}

type dockerConfig struct {
	Enabled bool   `yaml:"enabled"`
	Socket  string `yaml:"socket"`
}

func NewProvider(ctx context.Context, raw yaml.Node) (provider.Provider, bool, error) {
	var cfg dockerConfig
	if err := raw.Decode(&cfg); err != nil {
		return nil, false, fmt.Errorf("decode docker config: %w", err)
	}

	if !cfg.Enabled {
		return nil, false, nil
	}

	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if cfg.Socket != "" {
		opts = append(opts, client.WithHost(cfg.Socket))
	} else {
		opts = append(opts, client.FromEnv)
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, true, fmt.Errorf("create docker client: %w", err)
	}

	p := &DockerProvider{cli: cli}

	if err := p.HealthCheck(ctx); err != nil {
		return nil, true, fmt.Errorf("docker health check failed: %w", err)
	}

	return p, true, nil
}

func (p *DockerProvider) Discover(ctx context.Context) ([]provider.Target, error) {
	containers, err := p.cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var targets []provider.Target
	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		target, ok, err := parseLabels(name, c.ID, c.Labels, c.Mounts)

		if err != nil {
			slog.Warn("skipping containers with invalid labels", "container", name, "error", err)
			continue
		}

		if ok {
			targets = append(targets, target)
		}
	}

	return targets, nil
}

func (p *DockerProvider) HealthCheck(ctx context.Context) error {
	_, err := p.Discover(ctx)
	return err
}
