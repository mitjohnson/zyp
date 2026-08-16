package engine

import (
	"context"

	"zyp/internal/collector"
	"zyp/internal/config"
)

type Engine interface {
	Backup(ctx context.Context, dumps []collector.Dump) error
}

type Constructor func(repo config.Repository) Engine

var registry = map[string]Constructor{}

func Register(name string, constructor Constructor) {
	registry[name] = constructor
}

func Registered() map[string]Constructor {
	return registry
}
