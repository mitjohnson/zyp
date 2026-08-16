package app

import (
	"context"
	"log/slog"

	"zyp/internal/collector"
	"zyp/internal/config"
	"zyp/internal/engine"
	"zyp/internal/provider"
)

func discoverTargets(ctx context.Context, providers []provider.Provider) []provider.Target {
	var targets []provider.Target
	for _, p := range providers {
		found, err := p.Discover(ctx)
		if err != nil {
			slog.Warn("provider discovery failed", "error", err)
			continue
		}
		targets = append(targets, found...)
	}
	return targets
}

func collectDumps(ctx context.Context, targets []provider.Target) []collector.Dump {
	var dumps []collector.Dump
	for _, t := range targets {
		coll, ok := collector.Registered()[t.Kind]
		if !ok {
			slog.Warn("no collector for target kind, skipping", "target", t.Name, "kind", t.Kind)
			continue
		}

		dump, err := coll.Collect(ctx, t)
		if err != nil {
			slog.Warn("failed to collect target", "target", t.Name, "error", err)
			continue
		}

		dumps = append(dumps, dump)
	}
	return dumps
}

func groupByRepository(dumps []collector.Dump, cfg config.Config) map[string][]collector.Dump {
	groups := map[string][]collector.Dump{}
	for _, dump := range dumps {
		name := dump.Target.Repository
		if name == "" {
			name = cfg.DefaultRepository
		}
		groups[name] = append(groups[name], dump)
	}
	return groups
}

func backupGroups(ctx context.Context, groups map[string][]collector.Dump, cfg config.Config) {
	for name, dumps := range groups {
		repo, ok := cfg.Repositories[name]
		if !ok {
			slog.Warn("unknown repository, skipping", "repository", name)
			continue
		}

		constructor, ok := engine.Registered()[repo.Engine]
		if !ok {
			slog.Warn("unsupported engine, skipping", "repository", name, "engine", repo.Engine)
			continue
		}

		if err := constructor(repo).Backup(ctx, dumps); err != nil {
			slog.Warn("backup failed", "repository", name, "error", err)
		}
	}
}

func Run(ctx context.Context, cfg config.Config, providers []provider.Provider) error {
	targets := discoverTargets(ctx, providers)
	dumps := collectDumps(ctx, targets)

	if len(dumps) == 0 {
		slog.Info("nothing to back up")
		return nil
	}

	backupGroups(ctx, groupByRepository(dumps, cfg), cfg)

	return nil
}
