package docker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"zyp/internal/provider"

	"github.com/docker/docker/api/types/container"
)

type DockerProvider struct {
	cli ContainerLister
}

func NewDockerProvider(cli ContainerLister) *DockerProvider {
	return &DockerProvider{cli: cli}
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
		}

		if ok {
			targets = append(targets, target)
		}
	}

	return targets, nil
}
