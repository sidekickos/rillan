package httpapi

import (
	"context"

	"github.com/sidekickos/rillan/internal/classify"
	"github.com/sidekickos/rillan/internal/config"
	"github.com/sidekickos/rillan/internal/providers"
	"github.com/sidekickos/rillan/internal/retrieval"
	"github.com/sidekickos/rillan/internal/routing"
)

type RuntimeSnapshot struct {
	Provider      providers.Provider
	ProviderHost  providerHost
	Pipeline      *retrieval.Pipeline
	ProjectConfig config.ProjectConfig
	SystemConfig  *config.SystemConfig
	Classifier    classify.Classifier
	RouteCatalog  routing.Catalog
	RouteStatus   routing.StatusCatalog
	ReadinessInfo ReadinessInfo
	OllamaChecker func(context.Context) error
}

type RuntimeSnapshotFunc func() RuntimeSnapshot
