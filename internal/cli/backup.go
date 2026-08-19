package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"zyp/internal/app"
	"zyp/internal/config"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Discover, collect, and back up everything configured",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load(confPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		ctx := cmd.Context()
		providers := app.BuildProviders(ctx, cfg)

		return app.Run(ctx, cfg, providers)
	},
}
