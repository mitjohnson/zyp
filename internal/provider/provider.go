package provider

import (
	"context"

	"gopkg.in/yaml.v3"
)

type Kind string

const (
	KindFile     Kind = "file"
	KindSQLite   Kind = "sqlite"
	KindPostgres Kind = "postgres"
)

type Target struct {
	Name         string
	Kind         Kind
	Source       string
	Repository   string
	Compress     bool
	ContainerRef string
	Labels       map[string]string
}

type Provider interface {
	Name() string
	Discover(ctx context.Context) ([]Target, error)
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
