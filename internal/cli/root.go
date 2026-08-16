package cli

import (
	"context"

	"github.com/spf13/cobra"
)

var confPath string

var rootCmd = &cobra.Command{
	Use:   "zyp",
	Short: "Zyp is a backup tool for various data sources",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&confPath, "config", "zyp.yaml", "Path to the configuration file")
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(discoverCmd)
}

func Execute() error {
	return rootCmd.ExecuteContext(context.Background())
}
