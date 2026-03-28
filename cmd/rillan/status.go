package main

import (
	"fmt"
	"time"

	"github.com/sidekickos/rillan/internal/config"
	"github.com/sidekickos/rillan/internal/index"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show local index status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadWithMode(configPath, config.ValidationModeStatus)
			if err != nil {
				return err
			}

			status, err := index.ReadStatus(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "configured_root: %s\nlast_attempt_state: %s\nlast_attempt_root: %s\nlast_attempt_at: %s\nlast_attempt_error: %s\ncommitted_root: %s\ncommitted_last_indexed_at: %s\ndocuments: %d\nchunks: %d\nvectors: %d\ndb_path: %s\n",
				emptyFallback(status.ConfiguredRootPath, "not configured"),
				emptyFallback(status.LastAttemptState, index.RunStatusNeverIndexed),
				emptyFallback(status.LastAttemptRootPath, "none"),
				formatStatusTime(status.LastAttemptAt),
				emptyFallback(status.LastAttemptError, "none"),
				emptyFallback(status.CommittedRootPath, "none"),
				formatStatusTime(status.CommittedIndexedAt),
				status.Documents,
				status.Chunks,
				status.Vectors,
				status.DBPath,
			)
			return err
		},
	}

	cmd.Flags().StringVar(&configPath, "config", config.DefaultConfigPath(), "Path to the runtime config file")

	return cmd
}

func formatStatusTime(value time.Time) string {
	if value.IsZero() {
		return "never"
	}
	return value.Format("2006-01-02T15:04:05Z07:00")
}

func emptyFallback(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
