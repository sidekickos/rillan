package app

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/sidekickos/rillan/internal/classify"
	"github.com/sidekickos/rillan/internal/config"
	"github.com/sidekickos/rillan/internal/httpapi"
	"github.com/sidekickos/rillan/internal/index"
	"github.com/sidekickos/rillan/internal/modules"
	"github.com/sidekickos/rillan/internal/ollama"
	"github.com/sidekickos/rillan/internal/providers"
	"github.com/sidekickos/rillan/internal/retrieval"
	"github.com/sidekickos/rillan/internal/routing"
)

type runtimeSnapshotBuilder struct {
	configPath       string
	systemConfigPath string
	auditLedgerPath  string
}

func (b runtimeSnapshotBuilder) buildFromLoaded(ctx context.Context, cfg config.Config, project config.ProjectConfig, system *config.SystemConfig, projectConfigPath string) (*runtimeState, error) {
	snapshot, err := buildRuntimeSnapshot(ctx, cfg, project, system, b.auditLedgerPath, projectConfigPath)
	if err != nil {
		return nil, err
	}
	return &runtimeState{
		snapshot:          snapshot,
		projectConfigPath: projectConfigPath,
		systemConfigPath:  b.systemConfigPath,
	}, nil
}

func (b runtimeSnapshotBuilder) buildFromDisk(ctx context.Context) (*runtimeState, error) {
	cfg, err := config.Load(b.configPath)
	if err != nil {
		return nil, err
	}

	projectConfigPath := config.ResolveProjectConfigPath(cfg.Index.Root)
	projectCfg, err := config.LoadProject(projectConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			projectCfg = config.DefaultProjectConfig()
		} else {
			return nil, err
		}
	}

	var systemCfg *config.SystemConfig
	loadedSystemCfg, err := config.LoadSystem(b.systemConfigPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	} else {
		systemCfg = &loadedSystemCfg
	}

	return b.buildFromLoaded(ctx, cfg, projectCfg, systemCfg, projectConfigPath)
}

func buildRuntimeSnapshot(ctx context.Context, cfg config.Config, project config.ProjectConfig, system *config.SystemConfig, auditLedgerPath string, projectConfigPath string) (httpapi.RuntimeSnapshot, error) {
	discoveredModules, err := modules.LoadProjectCatalog(projectConfigPath)
	if err != nil {
		return httpapi.RuntimeSnapshot{}, err
	}
	moduleCatalog, err := modules.FilterEnabled(discoveredModules, project.Modules.Enabled)
	if err != nil {
		return httpapi.RuntimeSnapshot{}, err
	}

	providerHostCfg, err := config.ResolveRuntimeProviderHostConfig(cfg, project)
	if err != nil {
		return httpapi.RuntimeSnapshot{}, err
	}

	httpClient := &http.Client{}
	providerHost, err := providers.NewHost(providerHostCfg, httpClient)
	if err != nil {
		return httpapi.RuntimeSnapshot{}, err
	}

	provider, err := providerHost.DefaultProvider()
	if err != nil {
		return httpapi.RuntimeSnapshot{}, err
	}

	routeCatalog := routing.BuildCatalog(cfg, project)
	snapshot := httpapi.RuntimeSnapshot{
		Provider:      provider,
		ProviderHost:  providerHost,
		ProjectConfig: project,
		SystemConfig:  system,
		Modules:       moduleCatalog,
		RouteCatalog:  routeCatalog,
		RouteStatus: routing.BuildStatusCatalog(ctx, routing.StatusInput{
			Catalog:    routeCatalog,
			Config:     cfg,
			HTTPClient: httpClient,
		}),
		ReadinessInfo: httpapi.ReadinessInfo{
			RetrievalMode:      "disabled",
			SystemConfigLoaded: system != nil,
			AuditLedgerPath:    auditLedgerPath,
			LocalModelRequired: cfg.LocalModel.Enabled,
			ModulesDiscovered:  len(discoveredModules.Modules),
			ModulesEnabled:     len(moduleCatalog.Modules),
		},
	}
	if cfg.Retrieval.Enabled {
		snapshot.ReadinessInfo.RetrievalMode = "targeted_remote"
	}

	pipelineOpts := make([]retrieval.PipelineOption, 0, 2)
	if cfg.LocalModel.Enabled {
		ollamaClient := ollama.New(cfg.LocalModel.BaseURL, &http.Client{})
		snapshot.OllamaChecker = ollamaClient.Ping
		snapshot.Classifier = classify.NewSafeClassifier(classify.NewOllamaClassifier(ollamaClient, cfg.LocalModel.QueryRewrite.Model))

		pipelineOpts = append(pipelineOpts,
			retrieval.WithQueryEmbedder(
				retrieval.NewFallbackEmbedder(
					retrieval.NewOllamaEmbedder(ollamaClient, cfg.LocalModel.EmbedModel),
					retrieval.PlaceholderEmbedder{},
				),
			),
		)

		if cfg.LocalModel.QueryRewrite.Enabled {
			pipelineOpts = append(pipelineOpts,
				retrieval.WithQueryRewriter(retrieval.NewOllamaQueryRewriter(ollamaClient, cfg.LocalModel.QueryRewrite.Model)),
			)
		}
	}

	snapshot.Pipeline = retrieval.NewPipeline(cfg.Retrieval, index.DefaultDBPath(), pipelineOpts...)
	return snapshot, nil
}
