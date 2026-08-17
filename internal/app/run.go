package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"zyp/internal/collector"
	"zyp/internal/config"
	"zyp/internal/engine"
	"zyp/internal/provider"
	"zyp/internal/workdir"

	_ "zyp/internal/rclone"
	_ "zyp/internal/restic"
)

func DiscoverAllTargets(ctx context.Context, providers []provider.Provider) map[string][]provider.Target {
	results := map[string][]provider.Target{}
	for _, p := range providers {
		targets, err := p.Discover(ctx)
		if err != nil {
			slog.Warn("provider discovery failed", "provider", p)
			continue
		}
		results[p.Name()] = targets
	}
	return results
}

func collectDumps(ctx context.Context, targets []provider.Target, wd *workdir.WorkDir) []collector.Dump {
	var dumps []collector.Dump
	for _, t := range targets {
		coll, ok := collector.Registered()[t.Kind]
		if !ok {
			slog.Warn("no collector for target kind, skipping", "target", t.Name, "kind", t.Kind)
			continue
		}

		dump, err := coll.Collect(ctx, t, wd)
		if err != nil {
			slog.Warn("failed to collect target", "target", t.Name, "error", err)
			continue
		}

		dumps = append(dumps, dump)
	}
	return dumps
}

func defaultWorkDirPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "zyp")
	}
	return filepath.Join(home, ".cache", "zyp")
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
	wd, err := workdir.Open(defaultWorkDirPath())
	if err != nil {
		return fmt.Errorf("acquire work dir: %w", err)
	}
	defer wd.Close()

	var targets []provider.Target
	for _, discovered := range DiscoverAllTargets(ctx, providers) {
		targets = append(targets, discovered...)
	}

	dumps := collectDumps(ctx, targets, wd)

	if len(dumps) == 0 {
		slog.Info("nothing to back up")
		return nil
	}

	backupGroups(ctx, groupByRepository(dumps, cfg), cfg)

	return nil
}
