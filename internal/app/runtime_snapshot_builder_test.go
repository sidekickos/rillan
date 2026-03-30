package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sidekickos/rillan/internal/config"
)

func TestBuildRuntimeSnapshotLoadsProjectModules(t *testing.T) {
	projectRoot := t.TempDir()
	projectPath := filepath.Join(projectRoot, ".rillan", "project.yaml")
	manifestPath := filepath.Join(projectRoot, ".rillan", "modules", "demo", "module.yaml")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte(`id: "demo"
version: "0.1.0"
entrypoint: ["./bin/module"]
llm_adapters:
  - id: "demo-http"
    backend: "openai_compatible"
    transport: "http"
    endpoint: "https://example.com/v1"
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.SchemaVersion = config.SchemaVersionV1
	cfg.LLMs = config.LLMRegistryConfig{}
	cfg.Provider.Type = config.ProviderOpenAI
	cfg.Provider.OpenAI.APIKey = "test-key"
	project := config.DefaultProjectConfig()
	project.Name = "demo"
	project.Modules.Enabled = []string{"demo"}

	snapshot, err := buildRuntimeSnapshot(context.Background(), cfg, project, nil, filepath.Join(t.TempDir(), "audit.jsonl"), projectPath)
	if err != nil {
		t.Fatalf("buildRuntimeSnapshot returned error: %v", err)
	}
	if got, want := len(snapshot.Modules.Modules), 1; got != want {
		t.Fatalf("len(snapshot.Modules.Modules) = %d, want %d", got, want)
	}
	if got, want := snapshot.Modules.Modules[0].ID, "demo"; got != want {
		t.Fatalf("snapshot.Modules.Modules[0].ID = %q, want %q", got, want)
	}
	if got, want := snapshot.Modules.Modules[0].LLMAdapters[0].ID, "demo-http"; got != want {
		t.Fatalf("snapshot.Modules.Modules[0].LLMAdapters[0].ID = %q, want %q", got, want)
	}
	if got, want := snapshot.ReadinessInfo.ModulesDiscovered, 1; got != want {
		t.Fatalf("snapshot.ReadinessInfo.ModulesDiscovered = %d, want %d", got, want)
	}
	if got, want := snapshot.ReadinessInfo.ModulesEnabled, 1; got != want {
		t.Fatalf("snapshot.ReadinessInfo.ModulesEnabled = %d, want %d", got, want)
	}
}

func TestBuildRuntimeSnapshotLeavesModulesInactiveWhenProjectHasNoEnabledModules(t *testing.T) {
	projectRoot := t.TempDir()
	projectPath := filepath.Join(projectRoot, ".rillan", "project.yaml")
	manifestPath := filepath.Join(projectRoot, ".rillan", "modules", "demo", "module.yaml")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("id: \"demo\"\nversion: \"0.1.0\"\nentrypoint: [\"./bin/module\"]\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.SchemaVersion = config.SchemaVersionV1
	cfg.LLMs = config.LLMRegistryConfig{}
	cfg.Provider.Type = config.ProviderOpenAI
	cfg.Provider.OpenAI.APIKey = "test-key"
	project := config.DefaultProjectConfig()
	project.Name = "demo"

	snapshot, err := buildRuntimeSnapshot(context.Background(), cfg, project, nil, filepath.Join(t.TempDir(), "audit.jsonl"), projectPath)
	if err != nil {
		t.Fatalf("buildRuntimeSnapshot returned error: %v", err)
	}
	if got := len(snapshot.Modules.Modules); got != 0 {
		t.Fatalf("len(snapshot.Modules.Modules) = %d, want 0", got)
	}
	if got, want := snapshot.ReadinessInfo.ModulesDiscovered, 1; got != want {
		t.Fatalf("snapshot.ReadinessInfo.ModulesDiscovered = %d, want %d", got, want)
	}
	if got, want := snapshot.ReadinessInfo.ModulesEnabled, 0; got != want {
		t.Fatalf("snapshot.ReadinessInfo.ModulesEnabled = %d, want %d", got, want)
	}
}

func TestBuildRuntimeSnapshotRejectsUnknownEnabledModule(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SchemaVersion = config.SchemaVersionV1
	cfg.LLMs = config.LLMRegistryConfig{}
	cfg.Provider.Type = config.ProviderOpenAI
	cfg.Provider.OpenAI.APIKey = "test-key"
	project := config.DefaultProjectConfig()
	project.Name = "demo"
	project.Modules.Enabled = []string{"missing"}

	if _, err := buildRuntimeSnapshot(context.Background(), cfg, project, nil, filepath.Join(t.TempDir(), "audit.jsonl"), filepath.Join(t.TempDir(), ".rillan", "project.yaml")); err == nil {
		t.Fatal("expected unknown enabled module to fail")
	}
}
