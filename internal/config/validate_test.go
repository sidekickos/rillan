package config

import "testing"

func TestValidateRejectsImplicitAnthropic(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider.Type = ProviderAnthropic
	cfg.Provider.Anthropic.APIKey = "anthropic-key"
	cfg.Provider.OpenAI.APIKey = ""

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate returned nil error for implicit anthropic")
	}
}

func TestValidateAcceptsExplicitAnthropicOptIn(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider.Type = ProviderAnthropic
	cfg.Provider.Anthropic.Enabled = true
	cfg.Provider.Anthropic.APIKey = "anthropic-key"
	cfg.Provider.OpenAI.APIKey = ""

	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateRequiresOpenAIKeyForDefaultProvider(t *testing.T) {
	cfg := DefaultConfig()

	err := Validate(cfg)
	if err == nil {
		t.Fatal("Validate returned nil error without an OpenAI key")
	}
}

func TestValidateForIndexRequiresRootOnly(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Index.Root = "/tmp/project"

	if err := ValidateForMode(cfg, ValidationModeIndex); err != nil {
		t.Fatalf("ValidateForMode returned error: %v", err)
	}
}

func TestValidateForServeRequiresProviderKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Index.Root = "/tmp/project"

	if err := ValidateForMode(cfg, ValidationModeServe); err == nil {
		t.Fatal("expected serve validation to require a provider key")
	}
}

func TestValidateForStatusDoesNotRequireRoot(t *testing.T) {
	cfg := DefaultConfig()

	if err := ValidateForMode(cfg, ValidationModeStatus); err != nil {
		t.Fatalf("ValidateForMode returned error: %v", err)
	}
}
