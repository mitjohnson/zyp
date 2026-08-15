package provider

import "context"

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
	ContainerRef string
	Labels       map[string]string
}

type Provider interface {
	Discover(ctx context.Context) ([]Target, error)
}
