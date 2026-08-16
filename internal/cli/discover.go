package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"zyp/internal/app"
	"zyp/internal/config"
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
		providers := app.BuildProviders(ctx, cfg)
		discovered := app.DiscoverAllTargets(ctx, providers)

		data, err := json.MarshalIndent(discovered, "", "  ")
		if err != nil {
			return fmt.Errorf("format results: %w", err)
		}

		fmt.Println(string(data))

		return nil
	},
}
