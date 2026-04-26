package index

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rillanai/rillan/internal/config"
)

func TestRebuildIncludesGraphifyContentWhenEnabled(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	graphRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(graphRoot, "graph.json"), []byte(`{"nodes":[{"id":"AuthHandler","label":"AuthHandler","type":"class"}],"edges":[{"source":"AuthHandler","target":"PaymentFlow","type":"calls"}]}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(graphRoot, "wiki"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(graphRoot, "wiki", "AuthHandler.md"), []byte("# AuthHandler\nCalls PaymentFlow.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cfg := config.DefaultConfig()
	cfg.Index.Root = root
	cfg.KnowledgeGraph.Enabled = true
	cfg.KnowledgeGraph.Path = graphRoot

	if _, err := Rebuild(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}

	store, err := OpenStore(DefaultDBPath())
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	defer store.Close()

	results, err := store.SearchChunksKeyword(context.Background(), "PaymentFlow", 10)
	if err != nil {
		t.Fatalf("SearchChunksKeyword returned error: %v", err)
	}
	found := false
	for _, result := range results {
		if strings.HasPrefix(result.DocumentPath, graphifyPrefix) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Graphify chunk among results: %#v", results)
	}
}

func TestDiscoverGraphifyFilesSkipsMalformedGraphJSON(t *testing.T) {
	graphRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(graphRoot, "graph.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(graphRoot, "notes.md"), []byte("# Notes\nKeep me.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cfg := config.KnowledgeGraphConfig{Enabled: true, Path: graphRoot, MaxNodes: 100}
	files, err := DiscoverGraphifyFiles(cfg)
	if err != nil {
		t.Fatalf("DiscoverGraphifyFiles should degrade gracefully on malformed graph.json, got error: %v", err)
	}

	var hasGraph, hasNotes bool
	for _, f := range files {
		if f.RelativePath == graphifyPrefix+"graph.json" {
			hasGraph = true
		}
		if f.RelativePath == graphifyPrefix+"notes.md" {
			hasNotes = true
		}
	}
	if hasGraph {
		t.Fatalf("malformed graph.json should not produce a source file; got %#v", files)
	}
	if !hasNotes {
		t.Fatalf("markdown file should still be discovered alongside malformed graph.json; got %#v", files)
	}
}

func TestDiscoverGraphifyFilesSkipsPerFileReadErrors(t *testing.T) {
	graphRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(graphRoot, "graph.json"), []byte(`{"nodes":[{"id":"a"}],"edges":[]}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(graphRoot, "good.md"), []byte("# Good\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	brokenPath := filepath.Join(graphRoot, "broken.md")
	if err := os.Symlink(filepath.Join(graphRoot, "missing-target.md"), brokenPath); err != nil {
		t.Skipf("symlink unsupported on this platform: %v", err)
	}

	cfg := config.KnowledgeGraphConfig{Enabled: true, Path: graphRoot, MaxNodes: 100}
	files, err := DiscoverGraphifyFiles(cfg)
	if err != nil {
		t.Fatalf("DiscoverGraphifyFiles should skip a broken symlink, got error: %v", err)
	}

	var hasGraph, hasGood, hasBroken bool
	for _, f := range files {
		switch f.RelativePath {
		case graphifyPrefix + "graph.json":
			hasGraph = true
		case graphifyPrefix + "good.md":
			hasGood = true
		case graphifyPrefix + "broken.md":
			hasBroken = true
		}
	}
	if !hasGraph {
		t.Errorf("expected graph.json summary to be present; got %#v", files)
	}
	if !hasGood {
		t.Errorf("expected good.md to be discovered; got %#v", files)
	}
	if hasBroken {
		t.Errorf("broken symlink should not produce a source file; got %#v", files)
	}
}

func TestRebuildSucceedsWithMalformedGraphJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	graphRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(graphRoot, "graph.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))

	cfg := config.DefaultConfig()
	cfg.Index.Root = root
	cfg.KnowledgeGraph.Enabled = true
	cfg.KnowledgeGraph.Path = graphRoot

	if _, err := Rebuild(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("Rebuild should succeed with malformed graph.json; got error: %v", err)
	}
}

func TestReadGraphifyStatus(t *testing.T) {
	graphRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(graphRoot, "graph.json"), []byte(`{"nodes":[{"id":"n1"},{"id":"n2"}],"edges":[{"source":"n1","target":"n2"}]}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	status, err := ReadGraphifyStatus(config.KnowledgeGraphConfig{Enabled: true, Path: graphRoot})
	if err != nil {
		t.Fatalf("ReadGraphifyStatus returned error: %v", err)
	}
	if !status.Present || status.Nodes != 2 || status.Edges != 1 || status.SHA256 == "" {
		t.Fatalf("unexpected status: %#v", status)
	}
}
