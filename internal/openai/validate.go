package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func ValidateChatCompletionRequest(req ChatCompletionRequest) error {
	if strings.TrimSpace(req.Model) == "" {
		return fmt.Errorf("model must not be empty")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("messages must contain at least one item")
	}

	for idx, message := range req.Messages {
		if !validRole(message.Role) {
			return fmt.Errorf("messages[%d].role must be one of system, developer, user, assistant, or tool", idx)
		}

		content, err := MessageText(message)
		if err != nil {
			return fmt.Errorf("messages[%d].content must be a string in milestone one", idx)
		}
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("messages[%d].content must not be empty", idx)
		}
	}

	if req.Retrieval != nil {
		if req.Retrieval.TopK != nil && *req.Retrieval.TopK < 1 {
			return fmt.Errorf("retrieval.top_k must be greater than zero")
		}
		if req.Retrieval.MaxContextChars != nil && *req.Retrieval.MaxContextChars < 1 {
			return fmt.Errorf("retrieval.max_context_chars must be greater than zero")
		}
	}

	return nil
}

func MessageText(message Message) (string, error) {
	var content string
	if err := json.Unmarshal(message.Content, &content); err != nil {
		return "", err
	}
	return content, nil
}

func WriteError(w http.ResponseWriter, status int, messageType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: APIError{
			Message: message,
			Type:    messageType,
		},
	})
}

func validRole(role string) bool {
	switch role {
	case "system", "developer", "user", "assistant", "tool":
		return true
	default:
		return false
	}
}
