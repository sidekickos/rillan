package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sidekickos/rillan/internal/config"
	"github.com/sidekickos/rillan/internal/index"
	"github.com/sidekickos/rillan/internal/ollama"
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
			if err != nil {
				return err
			}

			// Local model status
			if cfg.LocalModel.Enabled {
				client := ollama.New(cfg.LocalModel.BaseURL, &http.Client{})
				reachable := client.Ping(cmd.Context()) == nil
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "local_model_enabled: true\nlocal_model_url: %s\nlocal_model_reachable: %t\nlocal_model_embed_model: %s\nlocal_model_query_rewrite: %t\n",
					cfg.LocalModel.BaseURL,
					reachable,
					cfg.LocalModel.EmbedModel,
					cfg.LocalModel.QueryRewrite.Enabled,
				)
			} else {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "local_model_enabled: false\n")
			}
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
