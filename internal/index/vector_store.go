package index

import (
	"context"
	"fmt"
)

const VectorStoreModeEmbedded = "embedded"

type VectorStore interface {
	BuildRecords(ctx context.Context, chunks []ChunkRecord) ([]VectorRecord, error)
	Mode() string
}

type EmbeddedVectorStore struct{}

func NewVectorStore(mode string) (VectorStore, error) {
	switch mode {
	case "", VectorStoreModeEmbedded:
		return EmbeddedVectorStore{}, nil
	default:
		return nil, fmt.Errorf("unsupported vector store mode %q", mode)
	}
}

func (EmbeddedVectorStore) Mode() string {
	return VectorStoreModeEmbedded
}

func (EmbeddedVectorStore) BuildRecords(ctx context.Context, chunks []ChunkRecord) ([]VectorRecord, error) {
	vectors := make([]VectorRecord, 0, len(chunks))
	for _, chunk := range chunks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		embedding := PlaceholderEmbedding(chunk.Content)
		vectors = append(vectors, VectorRecord{
			ChunkID:    chunk.ID,
			Dimensions: len(embedding),
			Embedding:  EncodeEmbedding(embedding),
		})
	}

	return vectors, nil
}
