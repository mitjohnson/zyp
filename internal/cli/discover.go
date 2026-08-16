package cli

import (
	"fmt"

	"github.com/spf13/cobra"

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
		providers := buildProviders(ctx, cfg)

		for _, p := range providers {
			targets, err := p.Discover(ctx)
			if err != nil {
				fmt.Printf("discover failed: %v\n", err)
				continue
			}

			for _, t := range targets {
				fmt.Printf("%+v\n", t)
			}
		}

		return nil
	},
}
