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

var registry = map[provider.Kind]Collector{}

func Register(kind provider.Kind, c Collector) {
	registry[kind] = c
}

func Registered() map[provider.Kind]Collector {
	return registry
}
