package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sidekickos/rillan/internal/index"
)

func TestStatusCommandShowsCommittedAndFailedAttemptSeparately(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	configPath := writeStatusTestConfig(t)
	store, err := index.OpenStore(index.DefaultDBPath())
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	runID, err := store.RecordRunStart(ctx, "/committed-root")
	if err != nil {
		t.Fatalf("RecordRunStart returned error: %v", err)
	}
	if err := store.ReplaceAll(ctx,
		[]index.DocumentRecord{{Path: "a.go", ContentHash: "hash", SizeBytes: 10}},
		[]index.ChunkRecord{{ID: "chunk", DocumentPath: "a.go", Ordinal: 0, StartLine: 1, EndLine: 1, Content: "package main", ContentHash: "hash2"}},
		[]index.VectorRecord{{ChunkID: "chunk", Dimensions: 8, Embedding: []byte{1, 2, 3, 4}}},
	); err != nil {
		t.Fatalf("ReplaceAll returned error: %v", err)
	}
	if err := store.RecordRunCompletion(ctx, runID, index.RunStatusSucceeded, 1, 1, 1, ""); err != nil {
		t.Fatalf("RecordRunCompletion returned error: %v", err)
	}

	failedRunID, err := store.RecordRunStart(ctx, "/failed-root")
	if err != nil {
		t.Fatalf("RecordRunStart returned error: %v", err)
	}
	if err := store.RecordRunCompletion(ctx, failedRunID, index.RunStatusFailed, 0, 0, 0, "boom"); err != nil {
		t.Fatalf("RecordRunCompletion returned error: %v", err)
	}

	cmd := newStatusCommand()
	cmd.SetArgs([]string{"--config", configPath})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"configured_root: /configured-root",
		"last_attempt_state: failed",
		"last_attempt_root: /failed-root",
		"last_attempt_error: boom",
		"committed_root: /committed-root",
		"documents: 1",
		"chunks: 1",
		"vectors: 1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func writeStatusTestConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `server:
  host: "127.0.0.1"
  port: 8420
  log_level: "info"

index:
  root: "/configured-root"

runtime:
  vector_store_mode: "embedded"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	return path
}
