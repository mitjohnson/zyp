package collector

import (
	"context"
	"zyp/internal/target"
	"zyp/internal/workdir"
)

type Dump struct {
	Target target.Target
	Path   string
}

type Collector interface {
	Collect(ctx context.Context, t target.Target, wd *workdir.WorkDir) (Dump, error)
}

var registry = map[target.Kind]Collector{}

func Register(kind target.Kind, c Collector) {
	registry[kind] = c
}

func Registered() map[target.Kind]Collector {
	return registry
}
