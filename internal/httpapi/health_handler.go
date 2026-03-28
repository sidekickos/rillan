package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	internalopenai "github.com/sidekickos/rillan/internal/openai"
)

type readinessChecker interface {
	Ready(context.Context) error
}

func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func ReadyHandler(checker readinessChecker, ollamaChecker func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := checker.Ready(r.Context()); err != nil {
			internalopenai.WriteError(w, http.StatusServiceUnavailable, "service_unavailable", err.Error())
			return
		}

		resp := map[string]any{"status": "ready"}
		if ollamaChecker != nil {
			if err := ollamaChecker(r.Context()); err != nil {
				resp["local_model"] = map[string]string{"status": "unavailable", "error": err.Error()}
			} else {
				resp["local_model"] = map[string]string{"status": "available"}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
