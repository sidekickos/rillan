package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesEnvOverrides(t *testing.T) {
	t.Setenv("RILLAN_OPENAI_API_KEY", "env-openai-key")
	t.Setenv("RILLAN_SERVER_PORT", "9001")
	t.Setenv("RILLAN_INDEX_ROOT", "/tmp/project")

	path := writeTempConfig(t, `server:
  host: "127.0.0.1"
provider:
  type: "openai"
  openai:
    api_key: "file-openai-key"
runtime:
  vector_store_mode: "embedded"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got, want := cfg.Provider.OpenAI.APIKey, "env-openai-key"; got != want {
		t.Fatalf("openai api key = %q, want %q", got, want)
	}

	if got, want := cfg.Server.Port, 9001; got != want {
		t.Fatalf("server port = %d, want %d", got, want)
	}

	if got, want := cfg.Index.Root, "/tmp/project"; got != want {
		t.Fatalf("index root = %q, want %q", got, want)
	}
}

func TestLoadDefaultsProviderTypeToOpenAI(t *testing.T) {
	t.Setenv("RILLAN_OPENAI_API_KEY", "test-key")

	path := writeTempConfig(t, `runtime:
  vector_store_mode: "embedded"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got, want := cfg.Provider.Type, ProviderOpenAI; got != want {
		t.Fatalf("provider.type = %q, want %q", got, want)
	}
}

func TestLoadAppliesIndexDefaults(t *testing.T) {
	t.Setenv("RILLAN_OPENAI_API_KEY", "test-key")

	path := writeTempConfig(t, `runtime:
  vector_store_mode: "embedded"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got, want := cfg.Index.ChunkSizeLines, 120; got != want {
		t.Fatalf("chunk_size_lines = %d, want %d", got, want)
	}
	if len(cfg.Index.Excludes) == 0 {
		t.Fatal("expected default excludes to be populated")
	}
}

func TestDefaultPathsAreNonEmpty(t *testing.T) {
	if DefaultConfigPath() == "" {
		t.Fatal("DefaultConfigPath returned empty string")
	}
	if DefaultDataDir() == "" {
		t.Fatal("DefaultDataDir returned empty string")
	}
	if DefaultLogDir() == "" {
		t.Fatal("DefaultLogDir returned empty string")
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	return path
}
