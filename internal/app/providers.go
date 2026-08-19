package app

import (
	"context"
	"log/slog"

	// Registers docker provider
	_ "zyp/internal/docker"

	"zyp/internal/config"
	"zyp/internal/provider"
)

func BuildProviders(ctx context.Context, cfg config.Config) []provider.Provider {
	var providers []provider.Provider

	for name, rawCfg := range cfg.Providers {
		constructor, ok := provider.Registered()[name]
		if !ok {
			slog.Warn("unknown provider in config, skipping", "provider", name)
			continue
		}

		p, enabled, err := constructor(ctx, rawCfg)
		if !enabled {
			slog.Info("provider disabled, skipping", "provider", name)
			continue
		}
		if err != nil {
			slog.Warn("provider unavailable, skipping", "provider", name, "error", err)
			continue
		}

		slog.Info("provider ready", "provider", name)
		providers = append(providers, p)
	}

	return providers
}
