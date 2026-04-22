package main

import (
	"bytes"
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"
)

// SplitDocuments splits a multi-doc YAML byte stream on the `---` separator and
// parses each document into a generic map. Documents that are empty or
// comment-only (producing a nil parse result) are silently skipped.
func SplitDocuments(in []byte) ([]map[string]any, error) {
	text := strings.ReplaceAll(string(in), "\r\n", "\n")

	var docs []map[string]any
	for i, chunk := range splitOnYAMLSeparator(text) {
		trim := strings.TrimSpace(chunk)
		if trim == "" {
			continue
		}
		var doc map[string]any
		if err := yaml.Unmarshal([]byte(chunk), &doc); err != nil {
			return nil, fmt.Errorf("document %d: %w", i, err)
		}
		if doc == nil {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// JoinDocuments serializes a list of documents into a single multi-doc YAML
// byte stream with `---` separators between documents. Order is preserved.
func JoinDocuments(docs []map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	for i, d := range docs {
		if i > 0 {
			buf.WriteString("---\n")
		}
		b, err := yaml.Marshal(d)
		if err != nil {
			return nil, fmt.Errorf("document %d: %w", i, err)
		}
		buf.Write(b)
	}
	return buf.Bytes(), nil
}

// splitOnYAMLSeparator splits on lines consisting only of `---` (optional
// surrounding whitespace). It intentionally does not use a naive Split on
// "---" because that would also match inside string values.
func splitOnYAMLSeparator(text string) []string {
	var chunks []string
	var current strings.Builder
	for _, line := range strings.SplitAfter(text, "\n") {
		stripped := strings.TrimSpace(line)
		if stripped == "---" {
			chunks = append(chunks, current.String())
			current.Reset()
			continue
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}
