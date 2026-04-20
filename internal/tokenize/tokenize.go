// SPDX-FileCopyrightText: 2026 Rillan AI
// SPDX-License-Identifier: Apache-2.0

// Package tokenize provides a deterministic outbound tokenizer seam used by
// the request-preparation path to budget remote-bound payloads.
//
// Token accounting is pinned to an Encoding rather than a Model alias so that
// counts stay stable even when upstream model names drift (see ADR-012).
// Model names are mapped to encodings by EncodingForModel. Unknown models and
// encodings that the backend cannot load degrade to an approximate count and
// emit a one-shot warning on the default slog logger.
//
// This first slice introduces the abstraction and a tiktoken-go-backed exact
// counter for the bundled OpenAI-compatible presets. Budget enforcement,
// source-span mapping, and minimization are layered on by later M06T tasks.
package tokenize

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	tiktoken "github.com/tiktoken-go/tokenizer"
)

// Encoding names a deterministic wire tokenization scheme. It maps 1:1 to a
// concrete codec in the backend.
type Encoding string

const (
	// EncodingCL100KBase is the encoding used by gpt-4 / gpt-4-turbo / gpt-3.5.
	EncodingCL100KBase Encoding = "cl100k_base"
	// EncodingO200KBase is the encoding used by gpt-4o, gpt-4.1, gpt-5, and
	// the o-series reasoning models.
	EncodingO200KBase Encoding = "o200k_base"
)

// CountResult is the outcome of counting tokens for a single string.
type CountResult struct {
	// Tokens is the token count. When Approximate is true this is a
	// conservative (round-up) heuristic rather than an exact codec count.
	Tokens int
	// Approximate is true when the backend could not provide an exact count
	// and the heuristic fallback was used.
	Approximate bool
	// Encoding is the encoding the exact count was produced with. Empty when
	// Approximate is true.
	Encoding Encoding
}

// Counter reports the token cost of a text for a given model. Implementations
// must be safe for concurrent use.
type Counter interface {
	Count(model, text string) (CountResult, error)
}

// EncodingForModel returns the deterministic encoding for a bundled
// OpenAI-compatible model name. The second return is false when the name does
// not match a known family; callers should fall back to an approximate count.
//
// Matching is prefix-based on the lowercased model name so that dated or
// versioned aliases (e.g. gpt-4o-2024-08-06) resolve to the same encoding as
// their family.
func EncodingForModel(model string) (Encoding, bool) {
	name := strings.ToLower(strings.TrimSpace(model))
	if name == "" {
		return "", false
	}
	switch {
	case strings.HasPrefix(name, "gpt-4o"),
		strings.HasPrefix(name, "gpt-4.1"),
		strings.HasPrefix(name, "gpt-5"),
		strings.HasPrefix(name, "o1"),
		strings.HasPrefix(name, "o3"),
		strings.HasPrefix(name, "o4"):
		return EncodingO200KBase, true
	case strings.HasPrefix(name, "gpt-4"),
		strings.HasPrefix(name, "gpt-3.5"):
		return EncodingCL100KBase, true
	}
	return "", false
}

// NewCounter returns the default tiktoken-go-backed Counter. It lazily loads
// codecs on first use and caches them for the process lifetime. Warnings for
// unknown models or unsupported encodings are emitted once per key via the
// default slog logger.
func NewCounter() Counter {
	return &tiktokenCounter{}
}

type tiktokenCounter struct {
	codecs sync.Map // Encoding -> tiktoken.Codec
	warned sync.Map // string -> struct{}
}

func (c *tiktokenCounter) Count(model, text string) (CountResult, error) {
	if text == "" {
		return CountResult{}, nil
	}
	encoding, ok := EncodingForModel(model)
	if !ok {
		c.warnOnce("model:"+strings.ToLower(strings.TrimSpace(model)),
			"tokenize: unknown model, using approximate token count",
			"model", model,
		)
		return CountResult{Tokens: approximateTokens(text), Approximate: true}, nil
	}
	codec, err := c.loadCodec(encoding)
	if err != nil {
		if errors.Is(err, tiktoken.ErrEncodingNotSupported) {
			c.warnOnce("encoding:"+string(encoding),
				"tokenize: encoding not supported by backend, using approximate token count",
				"encoding", string(encoding),
			)
			return CountResult{Tokens: approximateTokens(text), Approximate: true}, nil
		}
		return CountResult{}, fmt.Errorf("tokenize: load codec %s: %w", encoding, err)
	}
	tokens, err := codec.Count(text)
	if err != nil {
		return CountResult{}, fmt.Errorf("tokenize: count with %s: %w", encoding, err)
	}
	return CountResult{Tokens: tokens, Encoding: encoding}, nil
}

func (c *tiktokenCounter) loadCodec(encoding Encoding) (tiktoken.Codec, error) {
	if cached, ok := c.codecs.Load(encoding); ok {
		return cached.(tiktoken.Codec), nil
	}
	codec, err := tiktoken.Get(tiktoken.Encoding(encoding))
	if err != nil {
		return nil, err
	}
	actual, _ := c.codecs.LoadOrStore(encoding, codec)
	return actual.(tiktoken.Codec), nil
}

func (c *tiktokenCounter) warnOnce(key, msg string, args ...any) {
	if _, already := c.warned.LoadOrStore(key, struct{}{}); already {
		return
	}
	slog.Warn(msg, args...)
}

// CountStrings sums the token cost of multiple texts measured against the same
// model. It short-circuits on the first error. The returned CountResult is
// Approximate if any individual count fell back, and carries the exact
// Encoding used when at least one exact count was produced.
func CountStrings(c Counter, model string, texts ...string) (CountResult, error) {
	var total CountResult
	for _, text := range texts {
		result, err := c.Count(model, text)
		if err != nil {
			return CountResult{}, err
		}
		total.Tokens += result.Tokens
		if result.Approximate {
			total.Approximate = true
		} else if result.Encoding != "" {
			total.Encoding = result.Encoding
		}
	}
	return total, nil
}

// approximateTokens is the last-resort heuristic used when no exact codec is
// available. It rounds up so callers budgeting against it never under-count.
// The 4 bytes/token ratio is a conservative OpenAI rule-of-thumb for English
// text; non-Latin scripts will typically produce higher exact counts, which is
// acceptable for a pre-dispatch budget floor.
func approximateTokens(text string) int {
	if text == "" {
		return 0
	}
	return (len(text) + 3) / 4
}
