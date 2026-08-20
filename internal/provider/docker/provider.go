package docker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"zyp/internal/provider"
	"zyp/internal/target"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v3"
)

type Provider struct {
	cli DockerClient
	// Target name -> Container ID
	containers map[string]string
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

	p := &Provider{cli: cli, containers: map[string]string{}}

	if err := p.HealthCheck(ctx); err != nil {
		return nil, true, fmt.Errorf("docker health check failed: %w", err)
	}

	return p, true, nil
}

func (p *Provider) Discover(ctx context.Context) ([]target.Target, error) {
	containers, err := p.cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	if p.containers == nil {
		p.containers = map[string]string{}
	}

	var targets []target.Target
	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		target, ok, err := parseLabels(name, c.Labels, c.Mounts)


		if err != nil {
			slog.Warn("skipping containers with invalid labels", "container", name, "error", err)
			continue
		}

		if ok {
			p.containers[target.Name] = c.ID
			targets = append(targets, target)
		}
	}

	return targets, nil
}

func (p *Provider) HealthCheck(ctx context.Context) error {
	if _, err := p.cli.Ping(ctx); err != nil {
		return fmt.Errorf("ping docker daemon: %w", err)
	}
	return nil
}

func (p *Provider) Name() string {
	return "docker"
}
