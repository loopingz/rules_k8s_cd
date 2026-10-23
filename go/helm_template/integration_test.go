package main

import (
	"os"
	"strings"
	"testing"
)

// These files are build outputs of helm_template rules, rendered by managed Helm.
func TestRenderedChart(t *testing.T) {
	for _, name := range []string{"rendered", "archived"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(name + ".yaml")
			if err != nil {
				t.Fatal(err)
			}
			output := string(data)
			for _, want := range []string{"name: example", "namespace: prod", "replicas: 7", `code: "00123"`, `message: "later values"`, "kind: CustomResourceDefinition", `kubeVersion: "v1.32.0"`} {
				if !strings.Contains(output, want) {
					t.Errorf("missing %q in:\n%s", want, output)
				}
			}
			for _, removed := range []string{"kind: Secret", "kind: Service", "kind: Pod"} {
				if strings.Contains(output, removed) {
					t.Errorf("excluded resource %q present", removed)
				}
			}
		})
	}
	data, err := os.ReadFile("unfiltered.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"kind: Secret", "kind: Service", "kind: Pod", "replicas: 1", "# Source:"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("unfiltered output missing %q", want)
		}
	}
	if strings.Contains(string(data), "kind: CustomResourceDefinition") {
		t.Error("CRDs must be opt in")
	}
}
