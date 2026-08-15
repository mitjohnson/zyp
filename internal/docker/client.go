package docker

import (
	"context"
	"github.com/docker/docker/api/types/container"
)

type ContainerLister interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
}
