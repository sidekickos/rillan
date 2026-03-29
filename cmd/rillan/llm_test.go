package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sidekickos/rillan/internal/config"
	"github.com/sidekickos/rillan/internal/secretstore"
)

func TestLLMAddCreatesProviderEntry(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	cmd := newLLMCommand()
	cmd.SetArgs([]string{"--config", configPath, "add", "work-gpt", "--type", "openai", "--endpoint", "https://api.openai.com/v1", "--default-model", "gpt-5", "--capability", "chat", "--capability", "reasoning"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	cfg, err := config.LoadForEdit(configPath)
	if err != nil {
		t.Fatalf("LoadForEdit returned error: %v", err)
	}
	if got, want := cfg.LLMs.Default, "work-gpt"; got != want {
		t.Fatalf("llms.default = %q, want %q", got, want)
	}
	if got, want := len(cfg.LLMs.Providers), 1; got != want {
		t.Fatalf("len(llms.providers) = %d, want %d", got, want)
	}
	provider := cfg.LLMs.Providers[0]
	if got, want := provider.AuthStrategy, config.AuthStrategyBrowserOIDC; got != want {
		t.Fatalf("auth_strategy = %q, want %q", got, want)
	}
	if got, want := strings.Join(provider.Capabilities, ","), "chat,reasoning"; got != want {
		t.Fatalf("capabilities = %q, want %q", got, want)
	}
}

func TestLLMUseSwitchesDefaultProvider(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := config.DefaultConfig()
	cfg.LLMs.Providers = []config.LLMProviderConfig{
		{ID: "work-gpt", Type: config.ProviderOpenAI, Endpoint: "https://api.openai.com/v1", AuthStrategy: config.AuthStrategyBrowserOIDC},
		{ID: "kimi-prod", Type: config.ProviderKimi, Endpoint: "https://api.moonshot.ai/v1", AuthStrategy: config.AuthStrategyAPIKey},
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	cmd := newLLMCommand()
	cmd.SetArgs([]string{"--config", configPath, "use", "kimi-prod"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	reloaded, err := config.LoadForEdit(configPath)
	if err != nil {
		t.Fatalf("LoadForEdit returned error: %v", err)
	}
	if got, want := reloaded.LLMs.Default, "kimi-prod"; got != want {
		t.Fatalf("llms.default = %q, want %q", got, want)
	}
}

func TestLLMRemoveDeletesProviderEntry(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := config.DefaultConfig()
	cfg.LLMs.Default = "work-gpt"
	cfg.LLMs.Providers = []config.LLMProviderConfig{{ID: "work-gpt", Type: config.ProviderOpenAI, Endpoint: "https://api.openai.com/v1", AuthStrategy: config.AuthStrategyBrowserOIDC}}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	cmd := newLLMCommand()
	cmd.SetArgs([]string{"--config", configPath, "remove", "work-gpt"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	reloaded, err := config.LoadForEdit(configPath)
	if err != nil {
		t.Fatalf("LoadForEdit returned error: %v", err)
	}
	if got := len(reloaded.LLMs.Providers); got != 0 {
		t.Fatalf("len(llms.providers) = %d, want 0", got)
	}
	if reloaded.LLMs.Default != "" {
		t.Fatalf("llms.default = %q, want empty", reloaded.LLMs.Default)
	}
}

func TestLLMListPrintsSortedProviders(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := config.DefaultConfig()
	cfg.LLMs.Default = "work-gpt"
	cfg.LLMs.Providers = []config.LLMProviderConfig{
		{ID: "z-local", Type: config.ProviderLocal, Endpoint: "http://127.0.0.1:11434", AuthStrategy: config.AuthStrategyAPIKey},
		{ID: "work-gpt", Type: config.ProviderOpenAI, Endpoint: "https://api.openai.com/v1", AuthStrategy: config.AuthStrategyBrowserOIDC},
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	cmd := newLLMCommand()
	cmd.SetArgs([]string{"--config", configPath, "list"})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "default: work-gpt") {
		t.Fatalf("output missing default:\n%s", output)
	}
	if strings.Index(output, "- id: work-gpt") > strings.Index(output, "- id: z-local") {
		t.Fatalf("providers not sorted by id:\n%s", output)
	}
}

func TestLLMLoginAndLogoutStoreCredentialsSecurely(t *testing.T) {
	store := map[string]string{}
	secretstoreTestHooks(t, store)
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := config.DefaultConfig()
	cfg.LLMs.Providers = []config.LLMProviderConfig{{ID: "work-gpt", Type: config.ProviderOpenAI, Endpoint: "https://api.openai.com/v1", AuthStrategy: config.AuthStrategyBrowserOIDC, CredentialRef: credentialRefForLLM("work-gpt")}}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	login := newLLMCommand()
	login.SetArgs([]string{"--config", configPath, "login", "work-gpt", "--access-token", "token-1", "--issuer", "issuer-a"})
	if err := login.Execute(); err != nil {
		t.Fatalf("login Execute returned error: %v", err)
	}
	if _, ok := store[fmt.Sprintf("rillan/llm/%s", "work-gpt")]; !ok {
		t.Fatal("expected keyring entry to be created")
	}
	if !secretstore.Exists(credentialRefForLLM("work-gpt")) {
		t.Fatal("expected credential ref to resolve")
	}

	logout := newLLMCommand()
	logout.SetArgs([]string{"--config", configPath, "logout", "work-gpt"})
	if err := logout.Execute(); err != nil {
		t.Fatalf("logout Execute returned error: %v", err)
	}
	if secretstore.Exists(credentialRefForLLM("work-gpt")) {
		t.Fatal("expected credential ref to be cleared")
	}
}
