package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const exampleConfig = `server:
  host: "127.0.0.1"
  port: 8420
  log_level: "info"

provider:
  type: "openai"
  openai:
    base_url: "https://api.openai.com/v1"
    api_key: ""
  anthropic:
    enabled: false
    base_url: "https://api.anthropic.com"
    api_key: ""
  local:
    base_url: "http://127.0.0.1:11434"

index:
  root: ""
  includes: []
  excludes:
    - ".git"
    - "node_modules"
    - ".direnv"
    - ".idea"
  chunk_size_lines: 120

runtime:
  vector_store_mode: "embedded"
  local_model_base_url: "http://127.0.0.1:11434"
`

func WriteExampleConfig(path string, overwrite bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("refusing to overwrite existing file at %s; pass --force to replace it", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check config path: %w", err)
		}
	}

	if err := os.WriteFile(path, []byte(exampleConfig), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
