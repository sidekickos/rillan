package main

import (
	"fmt"

	"github.com/sidekickos/rillan/internal/config"
	"github.com/sidekickos/rillan/internal/index"
	"github.com/spf13/cobra"
)

func newIndexCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Build or rebuild the local index",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadWithMode(configPath, config.ValidationModeIndex)
			if err != nil {
				return err
			}

			status, err := index.Rebuild(cmd.Context(), cfg, newLogger(cfg.Server.LogLevel))
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "index complete\nroot: %s\ndocuments: %d\nchunks: %d\nvectors: %d\ndb_path: %s\n", status.CommittedRootPath, status.Documents, status.Chunks, status.Vectors, status.DBPath)
			return err
		},
	}

	cmd.Flags().StringVar(&configPath, "config", config.DefaultConfigPath(), "Path to the runtime config file")

	return cmd
}
