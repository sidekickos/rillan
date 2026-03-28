package openai

import "encoding/json"

type ChatCompletionRequest struct {
	Model     string            `json:"model"`
	Messages  []Message         `json:"messages"`
	Stream    bool              `json:"stream,omitempty"`
	Retrieval *RetrievalOptions `json:"retrieval,omitempty"`
	Metadata  []interface{}     `json:"-"`
}

type RetrievalOptions struct {
	Enabled         *bool `json:"enabled,omitempty"`
	TopK            *int  `json:"top_k,omitempty"`
	MaxContextChars *int  `json:"max_context_chars,omitempty"`
}

type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}
