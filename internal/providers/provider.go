// SPDX-FileCopyrightText: 2026 Rillan AI
// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/rillanai/rillan/internal/chat"
	"github.com/rillanai/rillan/internal/config"
	provideranthropic "github.com/rillanai/rillan/internal/providers/anthropic"
	providerollama "github.com/rillanai/rillan/internal/providers/ollama"
	provideropenai "github.com/rillanai/rillan/internal/providers/openai"
	providerstdio "github.com/rillanai/rillan/internal/providers/stdio"
)

type Provider interface {
	Name() string
	Ready(context.Context) error
	ChatCompletions(context.Context, chat.ProviderRequest) (*http.Response, error)
}

func newAdapter(cfg config.RuntimeProviderAdapterConfig, client *http.Client) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Transport)) {
	case "", config.LLMTransportHTTP:
	case config.LLMTransportSTDIO:
		return providerstdio.New(cfg.Command), nil
	default:
		return nil, fmt.Errorf("unsupported provider transport %q", cfg.Transport)
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Type)) {
	case config.ProviderOpenAI:
		return provideropenai.New(cfg.OpenAI, client), nil
	case config.ProviderOpenAICompatible:
		return provideropenai.New(cfg.OpenAI, client), nil
	case config.ProviderAnthropic:
		return provideranthropic.New(cfg.Anthropic, client), nil
	case config.ProviderOllama:
		return providerollama.New(cfg.LocalModel, client), nil
	default:
		return nil, fmt.Errorf("unsupported provider type %q", cfg.Type)
	}
}
