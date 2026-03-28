package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/sidekickos/rillan/internal/config"
	"github.com/sidekickos/rillan/internal/index"
	"github.com/sidekickos/rillan/internal/providers"
	"github.com/sidekickos/rillan/internal/retrieval"
)

func NewRouter(logger *slog.Logger, provider providers.Provider, cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", HealthHandler)
	mux.HandleFunc("GET /readyz", ReadyHandler(provider))
	mux.Handle("/v1/chat/completions", NewChatCompletionsHandler(logger, provider, retrieval.NewPipeline(cfg.Retrieval, index.DefaultDBPath())))

	return WrapWithMiddleware(logger, mux)
}
