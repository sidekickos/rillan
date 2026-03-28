package index

import (
	"context"
	"errors"
	"testing"
)

func TestNewVectorStoreDefaultsToEmbedded(t *testing.T) {
	store, err := NewVectorStore("")
	if err != nil {
		t.Fatalf("NewVectorStore returned error: %v", err)
	}
	if got, want := store.Mode(), VectorStoreModeEmbedded; got != want {
		t.Fatalf("mode = %q, want %q", got, want)
	}
}

func TestEmbeddedVectorStoreBuildRecordsDeterministically(t *testing.T) {
	store := EmbeddedVectorStore{}
	chunks := []ChunkRecord{{ID: "chunk-1", Content: "hello"}}

	first, err := store.BuildRecords(context.Background(), chunks)
	if err != nil {
		t.Fatalf("BuildRecords returned error: %v", err)
	}
	second, err := store.BuildRecords(context.Background(), chunks)
	if err != nil {
		t.Fatalf("BuildRecords returned error: %v", err)
	}

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected vector count: %d %d", len(first), len(second))
	}
	if string(first[0].Embedding) != string(second[0].Embedding) {
		t.Fatal("expected embedded vector records to be deterministic")
	}
}

func TestEmbeddedVectorStoreHonorsContextCancellation(t *testing.T) {
	store := EmbeddedVectorStore{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.BuildRecords(ctx, []ChunkRecord{{ID: "chunk-1", Content: "hello"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
