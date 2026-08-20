package provider

import (
	"context"

	"zyp/internal/target"

	"gopkg.in/yaml.v3"
)

type Provider interface {
	Name() string
	Discover(ctx context.Context) ([]target.Target, error)
	HealthCheck(ctx context.Context) error
}

type Constructor func(ctx context.Context, cfg yaml.Node) (p Provider, enabled bool, err error)

var registry = map[string]Constructor{}

func Register(name string, constructor Constructor) {
	registry[name] = constructor
}

func Registered() map[string]Constructor {
	return registry
}
