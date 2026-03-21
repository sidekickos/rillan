package config

import (
	"fmt"
	"strings"
)

func Validate(cfg Config) error {
	return ValidateForMode(cfg, ValidationModeServe)
}

func ValidateForMode(cfg Config, mode ValidationMode) error {
	if cfg.Server.Host == "" {
		return fmt.Errorf("server.host must not be empty")
	}
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	switch normalizeString(cfg.Server.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("server.log_level must be one of debug, info, warn, or error")
	}

	if normalizeString(cfg.Runtime.VectorStoreMode) != "embedded" {
		return fmt.Errorf("runtime.vector_store_mode must be %q in milestone two", "embedded")
	}

	if cfg.Index.ChunkSizeLines < 1 {
		return fmt.Errorf("index.chunk_size_lines must be greater than zero")
	}
	for _, pattern := range cfg.Index.Includes {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("index.includes must not contain empty patterns")
		}
	}
	for _, pattern := range cfg.Index.Excludes {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("index.excludes must not contain empty patterns")
		}
	}

	switch mode {
	case ValidationModeServe:
		switch normalizeString(cfg.Provider.Type) {
		case ProviderOpenAI:
			if cfg.Provider.OpenAI.APIKey == "" {
				return fmt.Errorf("provider.openai.api_key is required for the openai provider")
			}
		case ProviderAnthropic:
			if !cfg.Provider.Anthropic.Enabled {
				return fmt.Errorf("anthropic is never implicit; set provider.anthropic.enabled=true to opt in")
			}
			if cfg.Provider.Anthropic.APIKey == "" {
				return fmt.Errorf("provider.anthropic.api_key is required when anthropic is selected")
			}
		default:
			return fmt.Errorf("provider.type must be one of %q or %q", ProviderOpenAI, ProviderAnthropic)
		}
	case ValidationModeIndex:
		if strings.TrimSpace(cfg.Index.Root) == "" {
			return fmt.Errorf("index.root is required for the index command")
		}
	case ValidationModeStatus:
		return nil
	default:
		return fmt.Errorf("unsupported validation mode %q", mode)
	}

	return nil
}
