package main

import "testing"

func mkDoc(apiVersion, kind, name, namespace string) map[string]any {
	md := map[string]any{"name": name}
	if namespace != "" {
		md["namespace"] = namespace
	}
	return map[string]any{"apiVersion": apiVersion, "kind": kind, "metadata": md}
}

func TestParseSelector(t *testing.T) {
	s, err := ParseSelector("kind=NetworkPolicy,name=foo")
	if err != nil {
		t.Fatalf("ParseSelector: %v", err)
	}
	if s["kind"] != "NetworkPolicy" || s["name"] != "foo" {
		t.Errorf("unexpected: %v", s)
	}
}

func TestParseSelector_rejectsUnknownKey(t *testing.T) {
	if _, err := ParseSelector("weird=1"); err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestApplyExcludes_dropsByKind(t *testing.T) {
	docs := []map[string]any{
		mkDoc("v1", "Service", "s", ""),
		mkDoc("networking.k8s.io/v1", "NetworkPolicy", "np", ""),
		mkDoc("apps/v1", "Deployment", "d", ""),
	}
	sel, _ := ParseSelector("kind=NetworkPolicy")
	got := ApplyExcludes(docs, []Selector{sel})
	if len(got) != 2 {
		t.Fatalf("want 2 docs, got %d", len(got))
	}
	for _, d := range got {
		if d["kind"] == "NetworkPolicy" {
			t.Errorf("NetworkPolicy not filtered: %v", d)
		}
	}
}

func TestApplyExcludes_dictKeysAreAnded(t *testing.T) {
	docs := []map[string]any{
		mkDoc("v1", "Service", "keep", ""),
		mkDoc("v1", "Service", "drop", ""),
	}
	sel, _ := ParseSelector("kind=Service,name=drop")
	got := ApplyExcludes(docs, []Selector{sel})
	if len(got) != 1 || got[0]["metadata"].(map[string]any)["name"] != "keep" {
		t.Fatalf("expected only 'keep' to remain, got %v", got)
	}
}

func TestApplyExcludes_multipleSelectorsAreOred(t *testing.T) {
	docs := []map[string]any{
		mkDoc("v1", "Service", "s", ""),
		mkDoc("networking.k8s.io/v1", "NetworkPolicy", "np", ""),
		mkDoc("apps/v1", "Deployment", "d", ""),
	}
	sel1, _ := ParseSelector("kind=Service")
	sel2, _ := ParseSelector("kind=NetworkPolicy")
	got := ApplyExcludes(docs, []Selector{sel1, sel2})
	if len(got) != 1 || got[0]["kind"] != "Deployment" {
		t.Fatalf("expected only Deployment to remain, got %v", got)
	}
}

func TestApplyExcludes_matchesNamespace(t *testing.T) {
	docs := []map[string]any{
		mkDoc("v1", "ConfigMap", "c", "ns-a"),
		mkDoc("v1", "ConfigMap", "c", "ns-b"),
	}
	sel, _ := ParseSelector("kind=ConfigMap,namespace=ns-a")
	got := ApplyExcludes(docs, []Selector{sel})
	if len(got) != 1 || got[0]["metadata"].(map[string]any)["namespace"] != "ns-b" {
		t.Fatalf("wrong doc remained: %v", got)
	}
}
