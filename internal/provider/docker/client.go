package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

type Client interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	Ping(ctx context.Context) (types.Ping, error)
}
