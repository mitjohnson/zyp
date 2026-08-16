package engine

import (
	"context"

	"zyp/internal/collector"
)

type Engine interface {
	Backup(ctx context.Context, dumps []collector.Dump) error
}
