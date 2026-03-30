package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sidekickos/rillan/internal/audit"
	"github.com/sidekickos/rillan/internal/config"
	"github.com/sidekickos/rillan/internal/httpapi"
	"github.com/sidekickos/rillan/internal/policy"
	"github.com/sidekickos/rillan/internal/providers"
)

type App struct {
	addr       string
	configPath string
	logger     *slog.Logger
	runtime    *runtimeManager
	server     *http.Server
}

func New(cfg config.Config, project config.ProjectConfig, system *config.SystemConfig, configPath string, projectConfigPath string, systemConfigPath string, logger *slog.Logger) (*App, error) {
	if logger == nil {
		logger = slog.Default()
	}
	auditStore, err := audit.NewStore(audit.DefaultLedgerPath())
	if err != nil {
		return nil, err
	}
	builder := runtimeSnapshotBuilder{
		configPath:       configPath,
		systemConfigPath: systemConfigPath,
		auditLedgerPath:  auditStore.Path(),
	}
	initialState, err := builder.buildFromLoaded(context.Background(), cfg, project, system, projectConfigPath)
	if err != nil {
		return nil, err
	}
	runtime := newRuntimeManager(initialState, builder.buildFromDisk, logger)
	providerHost, _ := initialState.snapshot.ProviderHost.(*providers.Host)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr: addr,
		Handler: httpapi.NewRouter(logger, initialState.snapshot.Provider, cfg, httpapi.RouterOptions{
			ProjectConfig:      initialState.snapshot.ProjectConfig,
			SystemConfig:       initialState.snapshot.SystemConfig,
			SystemConfigLoaded: initialState.snapshot.ReadinessInfo.SystemConfigLoaded,
			AuditLedgerPath:    auditStore.Path(),
			AuditRecorder:      auditStore,
			PolicyEvaluator:    policy.NewEvaluator(),
			PolicyScanner:      policy.DefaultScanner(),
			Classifier:         initialState.snapshot.Classifier,
			ProviderHost:       providerHost,
			RouteCatalog:       initialState.snapshot.RouteCatalog,
			RouteStatus:        initialState.snapshot.RouteStatus,
			RetrievalMode:      initialState.snapshot.ReadinessInfo.RetrievalMode,
			LocalModelRequired: initialState.snapshot.ReadinessInfo.LocalModelRequired,
			OllamaChecker:      initialState.snapshot.OllamaChecker,
			RuntimeSnapshot:    runtime.CurrentSnapshot,
			RefreshRuntime:     runtime.Refresh,
		}),
	}

	return &App{
		addr:       addr,
		configPath: configPath,
		logger:     logger,
		runtime:    runtime,
		server:     server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	state := a.runtime.CurrentState()
	a.logger.Info("starting rillan server",
		"addr", a.addr,
		"config_path", a.configPath,
		"project_config_path", state.projectConfigPath,
		"system_config_path", state.systemConfigPath,
		"system_config_loaded", state.snapshot.ReadinessInfo.SystemConfigLoaded,
		"provider", state.snapshot.Provider.Name(),
	)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5e9)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("server shutdown failed", "error", err.Error())
		}
	}()

	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}
