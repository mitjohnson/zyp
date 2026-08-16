package cli

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"zyp/internal/config"
	"zyp/internal/provider"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Show what targets would be backed up, without collecting or backing up",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(confPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		ctx := cmd.Context()
		providers := buildProviders(ctx, cfg)

		results := map[string][]provider.Target{}
		for _, p := range providers {
			targets, err := p.Discover(ctx)
			if err != nil {
				slog.Warn("discover failed", "provider", p.Name(), "error", err)
				continue
			}

			if targets == nil {
				targets = []provider.Target{}
			}
			results[p.Name()] = targets
		}

		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("format results: %w", err)
		}

		fmt.Println(string(data))

		return nil
	},
}
