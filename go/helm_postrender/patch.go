package main

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/yaml"
)

// ApplyStrategicMerge applies a single strategic-merge patch file to the
// matching document in `docs`. The patch must contain apiVersion, kind, and
// metadata.name (optionally metadata.namespace) to identify its target.
// Returns a new slice where the target doc has been replaced with the merged
// version. Errors if no document matches.
func ApplyStrategicMerge(docs []map[string]any, patchPath string, patchBytes []byte) ([]map[string]any, error) {
	patchDoc := map[string]any{}
	if err := yaml.Unmarshal(patchBytes, &patchDoc); err != nil {
		return nil, fmt.Errorf("parse patch %s: %w", patchPath, err)
	}
	target, err := readTarget(patchDoc, patchPath)
	if err != nil {
		return nil, err
	}

	idx := findDoc(docs, target)
	if idx < 0 {
		return nil, fmt.Errorf("patch %s: target %s/%s name=%q namespace=%q does not match any rendered resource",
			patchPath, target["apiVersion"], target["kind"], target["name"], target["namespace"])
	}

	origJSON, err := json.Marshal(docs[idx])
	if err != nil {
		return nil, fmt.Errorf("patch %s: marshal original: %w", patchPath, err)
	}
	patchJSON, err := json.Marshal(patchDoc)
	if err != nil {
		return nil, fmt.Errorf("patch %s: marshal patch: %w", patchPath, err)
	}

	mergedJSON, err := strategicMergeWithFallback(origJSON, patchJSON, target["kind"])
	if err != nil {
		return nil, fmt.Errorf("patch %s: %w", patchPath, err)
	}

	var merged map[string]any
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		return nil, fmt.Errorf("patch %s: unmarshal merged: %w", patchPath, err)
	}

	out := make([]map[string]any, len(docs))
	copy(out, docs)
	out[idx] = merged
	return out, nil
}

// strategicMergeWithFallback tries strategic-merge against a known GVK schema;
// on failure (unknown type), falls back to RFC7396 JSON merge patch.
func strategicMergeWithFallback(orig, patch []byte, kind string) ([]byte, error) {
	schemaObj, ok := strategicMergeSchemaFor(kind)
	if ok {
		merged, err := strategicpatch.StrategicMergePatch(orig, patch, schemaObj)
		if err == nil {
			return merged, nil
		}
		// Fall through to merge-patch on schema error.
	}
	return jsonpatch.MergePatch(orig, patch)
}

// strategicMergeSchemaFor returns the runtime.Object used to derive the
// strategic-merge schema for well-known kinds. Returning ok=false triggers
// the JSON-merge-patch fallback.
func strategicMergeSchemaFor(kind string) (any, bool) {
	switch kind {
	case "Pod":
		return &corev1.Pod{}, true
	case "Service":
		return &corev1.Service{}, true
	case "ConfigMap":
		return &corev1.ConfigMap{}, true
	case "Secret":
		return &corev1.Secret{}, true
	case "PersistentVolumeClaim":
		return &corev1.PersistentVolumeClaim{}, true
	}
	return nil, false
}

// readTarget extracts the identity fields from a patch document.
func readTarget(patchDoc map[string]any, patchPath string) (map[string]string, error) {
	t := map[string]string{}
	t["apiVersion"], _ = patchDoc["apiVersion"].(string)
	t["kind"], _ = patchDoc["kind"].(string)
	if md, ok := patchDoc["metadata"].(map[string]any); ok {
		t["name"], _ = md["name"].(string)
		t["namespace"], _ = md["namespace"].(string)
	}
	if t["apiVersion"] == "" || t["kind"] == "" || t["name"] == "" {
		return nil, fmt.Errorf("patch %s: missing apiVersion/kind/metadata.name", patchPath)
	}
	return t, nil
}

// findDoc returns the index of the first document matching target identity.
// Namespace is only required if target specifies one.
func findDoc(docs []map[string]any, target map[string]string) int {
	for i, d := range docs {
		av, _ := d["apiVersion"].(string)
		kd, _ := d["kind"].(string)
		if av != target["apiVersion"] || kd != target["kind"] {
			continue
		}
		md, _ := d["metadata"].(map[string]any)
		nm, _ := md["name"].(string)
		if nm != target["name"] {
			continue
		}
		if ns := target["namespace"]; ns != "" {
			docNs, _ := md["namespace"].(string)
			if docNs != ns {
				continue
			}
		}
		return i
	}
	return -1
}

type json6902File struct {
	Target map[string]string `json:"target"`
	Patch  []any             `json:"patch"`
}

// ApplyJSON6902 applies a single JSON6902 patch file (YAML with top-level
// `target:` and `patch:` keys) to the matching document.
func ApplyJSON6902(docs []map[string]any, patchPath string, patchBytes []byte) ([]map[string]any, error) {
	var f json6902File
	if err := yaml.Unmarshal(patchBytes, &f); err != nil {
		return nil, fmt.Errorf("parse patch %s: %w", patchPath, err)
	}
	if f.Target["apiVersion"] == "" || f.Target["kind"] == "" || f.Target["name"] == "" {
		return nil, fmt.Errorf("patch %s: target missing apiVersion/kind/name", patchPath)
	}
	idx := findDoc(docs, f.Target)
	if idx < 0 {
		return nil, fmt.Errorf("patch %s: target %s/%s name=%q namespace=%q does not match any rendered resource",
			patchPath, f.Target["apiVersion"], f.Target["kind"], f.Target["name"], f.Target["namespace"])
	}

	opsJSON, err := json.Marshal(f.Patch)
	if err != nil {
		return nil, fmt.Errorf("patch %s: marshal ops: %w", patchPath, err)
	}
	patch, err := jsonpatch.DecodePatch(opsJSON)
	if err != nil {
		return nil, fmt.Errorf("patch %s: decode: %w", patchPath, err)
	}
	origJSON, err := json.Marshal(docs[idx])
	if err != nil {
		return nil, fmt.Errorf("patch %s: marshal original: %w", patchPath, err)
	}
	modifiedJSON, err := patch.Apply(origJSON)
	if err != nil {
		return nil, fmt.Errorf("patch %s: apply: %w", patchPath, err)
	}
	var merged map[string]any
	if err := json.Unmarshal(modifiedJSON, &merged); err != nil {
		return nil, fmt.Errorf("patch %s: unmarshal merged: %w", patchPath, err)
	}

	out := make([]map[string]any, len(docs))
	copy(out, docs)
	out[idx] = merged
	return out, nil
}
