package main

import (
	"reflect"
	"testing"
)

func TestSplitDocuments_threeDocs(t *testing.T) {
	in := []byte("apiVersion: v1\nkind: A\nmetadata:\n  name: a\n---\napiVersion: v1\nkind: B\nmetadata:\n  name: b\n---\napiVersion: v1\nkind: C\nmetadata:\n  name: c\n")
	docs, err := SplitDocuments(in)
	if err != nil {
		t.Fatalf("SplitDocuments: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("want 3 docs, got %d", len(docs))
	}
	want := []string{"A", "B", "C"}
	for i, d := range docs {
		if k, _ := d["kind"].(string); k != want[i] {
			t.Errorf("doc %d kind = %q, want %q", i, k, want[i])
		}
	}
}

func TestSplitDocuments_skipsEmptyAndCommentOnly(t *testing.T) {
	in := []byte("# just a comment\n---\napiVersion: v1\nkind: A\nmetadata:\n  name: a\n---\n# another\n---\n\n---\napiVersion: v1\nkind: B\nmetadata:\n  name: b\n")
	docs, err := SplitDocuments(in)
	if err != nil {
		t.Fatalf("SplitDocuments: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("want 2 docs (empty/comment skipped), got %d", len(docs))
	}
}

func TestJoinDocuments_preservesOrder(t *testing.T) {
	docs := []map[string]any{
		{"apiVersion": "v1", "kind": "A", "metadata": map[string]any{"name": "a"}},
		{"apiVersion": "v1", "kind": "B", "metadata": map[string]any{"name": "b"}},
	}
	out, err := JoinDocuments(docs)
	if err != nil {
		t.Fatalf("JoinDocuments: %v", err)
	}
	roundTrip, err := SplitDocuments(out)
	if err != nil {
		t.Fatalf("round-trip SplitDocuments: %v", err)
	}
	if !reflect.DeepEqual(docs, roundTrip) {
		t.Errorf("round-trip mismatch: got %v, want %v", roundTrip, docs)
	}
}
