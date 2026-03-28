package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/sidekickos/rillan/internal/config"
	"github.com/sidekickos/rillan/internal/index"
	"github.com/sidekickos/rillan/internal/providers"
	"github.com/sidekickos/rillan/internal/retrieval"
)

// RouterOptions configures the HTTP router.
type RouterOptions struct {
	OllamaChecker  func(context.Context) error
	PipelineOpts   []retrieval.PipelineOption
}

func NewRouter(logger *slog.Logger, provider providers.Provider, cfg config.Config, opts RouterOptions) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", HealthHandler)
	mux.HandleFunc("GET /readyz", ReadyHandler(provider, opts.OllamaChecker))
	mux.Handle("/v1/chat/completions", NewChatCompletionsHandler(logger, provider, retrieval.NewPipeline(cfg.Retrieval, index.DefaultDBPath(), opts.PipelineOpts...)))

	return WrapWithMiddleware(logger, mux)
}
