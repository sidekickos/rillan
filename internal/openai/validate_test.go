package openai

import (
	"encoding/json"
	"testing"
)

func TestValidateChatCompletionRequestAcceptsMinimalRequest(t *testing.T) {
	req := ChatCompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{{
			Role:    "user",
			Content: mustMarshalString(t, "hello"),
		}},
	}

	if err := ValidateChatCompletionRequest(req); err != nil {
		t.Fatalf("ValidateChatCompletionRequest returned error: %v", err)
	}
}

func TestValidateChatCompletionRequestRejectsMissingModel(t *testing.T) {
	req := ChatCompletionRequest{
		Messages: []Message{{Role: "user", Content: mustMarshalString(t, "hello")}},
	}

	if err := ValidateChatCompletionRequest(req); err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestValidateChatCompletionRequestRejectsNonStringContent(t *testing.T) {
	req := ChatCompletionRequest{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: "user", Content: json.RawMessage(`[{"type":"text","text":"hi"}]`)}},
	}

	if err := ValidateChatCompletionRequest(req); err == nil {
		t.Fatal("expected error for non-string content")
	}
}

func TestValidateChatCompletionRequestRejectsInvalidRetrievalOverride(t *testing.T) {
	zero := 0
	req := ChatCompletionRequest{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: "user", Content: mustMarshalString(t, "hello")}},
		Retrieval: &RetrievalOptions{
			TopK: &zero,
		},
	}

	if err := ValidateChatCompletionRequest(req); err == nil {
		t.Fatal("expected error for invalid retrieval override")
	}
}

func mustMarshalString(t *testing.T, value string) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	return data
}
