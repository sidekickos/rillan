// SPDX-FileCopyrightText: 2026 Rillan AI
// SPDX-License-Identifier: Apache-2.0

package tokenize

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestEncodingForModel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		model   string
		want    Encoding
		matched bool
	}{
		{"gpt-4o", "gpt-4o", EncodingO200KBase, true},
		{"gpt-4o dated alias", "gpt-4o-2024-08-06", EncodingO200KBase, true},
		{"gpt-4o-mini", "gpt-4o-mini", EncodingO200KBase, true},
		{"gpt-4.1", "gpt-4.1", EncodingO200KBase, true},
		{"gpt-5", "gpt-5-mini", EncodingO200KBase, true},
		{"o1", "o1-preview", EncodingO200KBase, true},
		{"o3-mini", "o3-mini", EncodingO200KBase, true},
		{"o4-mini", "o4-mini", EncodingO200KBase, true},
		{"gpt-4 turbo", "gpt-4-turbo", EncodingCL100KBase, true},
		{"gpt-4", "gpt-4", EncodingCL100KBase, true},
		{"gpt-3.5", "gpt-3.5-turbo", EncodingCL100KBase, true},
		{"uppercase and whitespace", "  GPT-4O  ", EncodingO200KBase, true},
		{"empty", "", "", false},
		{"unknown", "claude-3-opus", "", false},
		{"close but not a bundled family", "gpt2", "", false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := EncodingForModel(tc.model)
			if ok != tc.matched {
				t.Fatalf("matched = %v, want %v (got encoding %q)", ok, tc.matched, got)
			}
			if got != tc.want {
				t.Fatalf("encoding = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCounterExactForBundledEncodings(t *testing.T) {
	t.Parallel()
	counter := NewCounter()
	text := "Hello, Rillan tokenizer!"

	tests := []struct {
		model    string
		encoding Encoding
	}{
		{"gpt-4o", EncodingO200KBase},
		{"gpt-4-turbo", EncodingCL100KBase},
		{"gpt-3.5-turbo", EncodingCL100KBase},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.model, func(t *testing.T) {
			t.Parallel()
			got, err := counter.Count(tc.model, text)
			if err != nil {
				t.Fatalf("Count: %v", err)
			}
			if got.Approximate {
				t.Fatalf("expected exact count for %s, got approximate", tc.model)
			}
			if got.Encoding != tc.encoding {
				t.Fatalf("Encoding = %q, want %q", got.Encoding, tc.encoding)
			}
			if got.Tokens <= 0 {
				t.Fatalf("expected positive token count, got %d", got.Tokens)
			}
			if got.Tokens > len(text) {
				t.Fatalf("exact token count %d exceeds byte length %d, which should not happen for ASCII", got.Tokens, len(text))
			}
		})
	}
}

func TestCounterExactCountsDifferBetweenEncodings(t *testing.T) {
	t.Parallel()
	counter := NewCounter()
	text := "func main() { fmt.Println(\"hello\") }"

	gpt4o, err := counter.Count("gpt-4o", text)
	if err != nil || gpt4o.Approximate {
		t.Fatalf("gpt-4o count failed: err=%v approx=%v", err, gpt4o.Approximate)
	}
	gpt4, err := counter.Count("gpt-4-turbo", text)
	if err != nil || gpt4.Approximate {
		t.Fatalf("gpt-4-turbo count failed: err=%v approx=%v", err, gpt4.Approximate)
	}
	if gpt4o.Encoding == gpt4.Encoding {
		t.Fatalf("expected distinct encodings, both returned %q", gpt4o.Encoding)
	}
}

func TestCounterApproximateFallbackForUnknownModel(t *testing.T) {
	t.Parallel()
	buf := captureSlogWarnings(t)

	counter := NewCounter()
	text := "some untokenized text input"
	got, err := counter.Count("claude-3-opus", text)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if !got.Approximate {
		t.Fatalf("expected approximate count for unknown model, got exact %+v", got)
	}
	if got.Encoding != "" {
		t.Fatalf("expected empty encoding on approximate path, got %q", got.Encoding)
	}
	expected := (len(text) + 3) / 4
	if got.Tokens != expected {
		t.Fatalf("tokens = %d, want %d", got.Tokens, expected)
	}

	// Second call with the same unknown model must not emit another warning
	// (one-shot dedupe).
	if _, err := counter.Count("claude-3-opus", text); err != nil {
		t.Fatalf("second Count: %v", err)
	}
	if occurrences := strings.Count(buf.String(), "unknown model"); occurrences != 1 {
		t.Fatalf("expected exactly 1 warning for repeated unknown model, got %d\n%s", occurrences, buf.String())
	}
}

func TestCountStringsAggregates(t *testing.T) {
	t.Parallel()
	counter := NewCounter()
	one, err := counter.Count("gpt-4o", "hello")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	two, err := counter.Count("gpt-4o", "world")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}

	sum, err := CountStrings(counter, "gpt-4o", "hello", "world")
	if err != nil {
		t.Fatalf("CountStrings: %v", err)
	}
	if sum.Tokens != one.Tokens+two.Tokens {
		t.Fatalf("sum tokens = %d, want %d", sum.Tokens, one.Tokens+two.Tokens)
	}
	if sum.Approximate {
		t.Fatalf("expected exact aggregate, got approximate")
	}
	if sum.Encoding != EncodingO200KBase {
		t.Fatalf("encoding = %q, want %q", sum.Encoding, EncodingO200KBase)
	}
}

func TestCountStringsMarksApproximateWhenAnyFellBack(t *testing.T) {
	t.Parallel()
	_ = captureSlogWarnings(t)
	counter := NewCounter()
	sum, err := CountStrings(counter, "claude-3-opus", "hello", "world")
	if err != nil {
		t.Fatalf("CountStrings: %v", err)
	}
	if !sum.Approximate {
		t.Fatalf("expected approximate aggregate, got %+v", sum)
	}
	if sum.Encoding != "" {
		t.Fatalf("encoding should be empty on fully-approximate path, got %q", sum.Encoding)
	}
	if sum.Tokens == 0 {
		t.Fatalf("expected non-zero token count")
	}
}

func TestCounterEmptyText(t *testing.T) {
	t.Parallel()
	counter := NewCounter()
	got, err := counter.Count("gpt-4o", "")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if got.Tokens != 0 || got.Approximate {
		t.Fatalf("expected zero exact tokens for empty text, got %+v", got)
	}
}

// captureSlogWarnings swaps the default slog logger for the duration of the
// test and returns a buffer containing everything logged during it.
func captureSlogWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return buf
}
