package config

import "log/slog"

const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Provider ProviderConfig `yaml:"provider"`
	Index    IndexConfig    `yaml:"index"`
	Runtime  RuntimeConfig  `yaml:"runtime"`
}

type ServerConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"log_level"`
}

type ProviderConfig struct {
	Type      string             `yaml:"type"`
	OpenAI    OpenAIConfig       `yaml:"openai"`
	Anthropic AnthropicConfig    `yaml:"anthropic"`
	Local     LocalModelProvider `yaml:"local"`
}

type OpenAIConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

type AnthropicConfig struct {
	Enabled bool   `yaml:"enabled"`
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

type LocalModelProvider struct {
	BaseURL string `yaml:"base_url"`
}

type IndexConfig struct {
	Root           string   `yaml:"root"`
	Includes       []string `yaml:"includes"`
	Excludes       []string `yaml:"excludes"`
	ChunkSizeLines int      `yaml:"chunk_size_lines"`
}

type RuntimeConfig struct {
	VectorStoreMode   string `yaml:"vector_store_mode"`
	LocalModelBaseURL string `yaml:"local_model_base_url"`
}

func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Host:     "127.0.0.1",
			Port:     8420,
			LogLevel: "info",
		},
		Provider: ProviderConfig{
			Type: ProviderOpenAI,
			OpenAI: OpenAIConfig{
				BaseURL: "https://api.openai.com/v1",
			},
			Anthropic: AnthropicConfig{
				Enabled: false,
				BaseURL: "https://api.anthropic.com",
			},
			Local: LocalModelProvider{
				BaseURL: "http://127.0.0.1:11434",
			},
		},
		Index: IndexConfig{
			Excludes:       []string{".git", "node_modules", ".direnv", ".idea"},
			ChunkSizeLines: 120,
		},
		Runtime: RuntimeConfig{
			VectorStoreMode:   "embedded",
			LocalModelBaseURL: "http://127.0.0.1:11434",
		},
	}
}

func ParseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
