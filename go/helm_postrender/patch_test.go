package main

import "testing"

func TestApplyStrategicMerge_replacesScalarField(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": "web"},
		"spec":       map[string]any{"replicas": float64(1)},
	}
	patch := []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 3
`)
	out, err := ApplyStrategicMerge([]map[string]any{doc}, "fake.yaml", patch)
	if err != nil {
		t.Fatalf("ApplyStrategicMerge: %v", err)
	}
	got := out[0]["spec"].(map[string]any)["replicas"]
	if f, ok := got.(float64); !ok || f != 3 {
		t.Errorf("replicas = %v, want 3", got)
	}
}

func TestApplyStrategicMerge_unmatchedTargetErrors(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": "web"},
	}
	patch := []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3
`)
	_, err := ApplyStrategicMerge([]map[string]any{doc}, "fake.yaml", patch)
	if err == nil {
		t.Fatal("expected error for unmatched patch target")
	}
}

func TestApplyStrategicMerge_mergesContainerEnvByName(t *testing.T) {
	// Uses Pod directly, whose PodSpec schema enables strategic-merge to
	// merge containers by name (a core/v1 merge-key behavior).
	doc := map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata":   map[string]any{"name": "web"},
		"spec": map[string]any{
			"containers": []any{
				map[string]any{
					"name":  "app",
					"image": "nginx",
					"env": []any{
						map[string]any{"name": "LOG", "value": "info"},
					},
				},
			},
		},
	}
	patch := []byte(`apiVersion: v1
kind: Pod
metadata:
  name: web
spec:
  containers:
  - name: app
    env:
    - name: EXTRA
      value: "1"
`)
	out, err := ApplyStrategicMerge([]map[string]any{doc}, "fake.yaml", patch)
	if err != nil {
		t.Fatalf("ApplyStrategicMerge: %v", err)
	}
	cs := out[0]["spec"].(map[string]any)["containers"].([]any)
	env := cs[0].(map[string]any)["env"].([]any)
	if len(env) != 2 {
		t.Fatalf("want 2 env vars (merged by name), got %d: %v", len(env), env)
	}
	if img := cs[0].(map[string]any)["image"]; img != "nginx" {
		t.Errorf("image = %v, want nginx (should be preserved)", img)
	}
}

func TestApplyStrategicMerge_fallsBackForUnknownGVK(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "example.com/v1",
		"kind":       "Widget",
		"metadata":   map[string]any{"name": "w"},
		"spec":       map[string]any{"color": "red"},
	}
	patch := []byte(`apiVersion: example.com/v1
kind: Widget
metadata:
  name: w
spec:
  color: blue
`)
	out, err := ApplyStrategicMerge([]map[string]any{doc}, "fake.yaml", patch)
	if err != nil {
		t.Fatalf("ApplyStrategicMerge: %v", err)
	}
	if c := out[0]["spec"].(map[string]any)["color"]; c != "blue" {
		t.Errorf("color = %v, want blue", c)
	}
}

func TestApplyJSON6902_replacesField(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": "web"},
		"spec":       map[string]any{"replicas": float64(1)},
	}
	patch := []byte(`target:
  apiVersion: apps/v1
  kind: Deployment
  name: web
patch:
  - op: replace
    path: /spec/replicas
    value: 5
`)
	out, err := ApplyJSON6902([]map[string]any{doc}, "p.yaml", patch)
	if err != nil {
		t.Fatalf("ApplyJSON6902: %v", err)
	}
	got := out[0]["spec"].(map[string]any)["replicas"]
	if f, ok := got.(float64); !ok || f != 5 {
		t.Errorf("replicas = %v, want 5", got)
	}
}

func TestApplyJSON6902_addOp(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata":   map[string]any{"name": "s", "labels": map[string]any{"a": "1"}},
	}
	patch := []byte(`target:
  apiVersion: v1
  kind: Service
  name: s
patch:
  - op: add
    path: /metadata/labels/b
    value: "2"
`)
	out, err := ApplyJSON6902([]map[string]any{doc}, "p.yaml", patch)
	if err != nil {
		t.Fatalf("ApplyJSON6902: %v", err)
	}
	got := out[0]["metadata"].(map[string]any)["labels"].(map[string]any)["b"]
	if got != "2" {
		t.Errorf("label b = %v, want 2", got)
	}
}

func TestApplyJSON6902_unmatchedTargetErrors(t *testing.T) {
	doc := map[string]any{
		"apiVersion": "v1", "kind": "Service",
		"metadata": map[string]any{"name": "s"},
	}
	patch := []byte(`target:
  apiVersion: v1
  kind: Service
  name: missing
patch:
  - op: replace
    path: /metadata/name
    value: x
`)
	if _, err := ApplyJSON6902([]map[string]any{doc}, "p.yaml", patch); err == nil {
		t.Error("expected error for unmatched target")
	}
}
