package collector

import (
	"context"
	"zyp/internal/provider"
)

type Dump struct {
	Target provider.Target
	Path   string
}

type Collector interface {
	Collect(ctx context.Context, t provider.Target) (Dump, error)
}
