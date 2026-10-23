package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const manifests = `# a leading empty document
---
apiVersion: v1
kind: Secret
metadata:
  name: credentials
---
apiVersion: v1
kind: Service
metadata:
  name: web
  namespace: prod
---
apiVersion: v1
kind: Service
metadata:
  name: web
  namespace: dev
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 3
  template:
    metadata:
      annotations:
        version: "123"
---
# trailing empty document
`

func resourceIDs(t *testing.T, input string) []string {
	t.Helper()
	var ids []string
	decoder := yaml.NewDecoder(strings.NewReader(input))
	for {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		metadata := field(&doc, "metadata")
		ids = append(ids, field(&doc, "kind").Value+"/"+field(metadata, "namespace").Value+"/"+field(metadata, "name").Value)
	}
	return ids
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name      string
		kinds     []string
		resources []string
		want      []string
	}{
		{"kind", []string{"Secret"}, nil, []string{"Service/prod/web", "Service/dev/web", "Deployment//web"}},
		{"all namespaces", nil, []string{"Service/web"}, []string{"Secret//credentials", "Deployment//web"}},
		{"namespace", nil, []string{"Service/prod/web"}, []string{"Secret//credentials", "Service/dev/web", "Deployment//web"}},
		{"glob", nil, []string{"S*/[pd]*/w?b"}, []string{"Secret//credentials", "Deployment//web"}},
		{"omitted namespace", nil, []string{"Deployment//web"}, []string{"Secret//credentials", "Service/prod/web", "Service/dev/web"}},
		{"combined", []string{"Secret"}, []string{"Service/web"}, []string{"Deployment//web"}},
		{"all removed", nil, []string{"*/*"}, nil},
		{"no matches", []string{"Job"}, []string{"Service/missing"}, []string{"Secret//credentials", "Service/prod/web", "Service/dev/web", "Deployment//web"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := newFilter(tt.kinds, tt.resources)
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := filter.filter(strings.NewReader(manifests), &output); err != nil {
				t.Fatal(err)
			}
			if got := resourceIDs(t, output.String()); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v\n%s", got, tt.want, output.String())
			}
			if strings.Contains(output.String(), "kind: Deployment") && !strings.Contains(output.String(), `version: "123"`) {
				t.Fatal("filter changed a string value's type")
			}
		})
	}
}

func TestInvalidFilters(t *testing.T) {
	for _, pattern := range []string{"Secret", "/name", "Kind/", "a/b/c/d", "Secret/[", "[bad/name"} {
		if _, err := newFilter(nil, []string{pattern}); err == nil {
			t.Errorf("accepted invalid pattern %q", pattern)
		}
	}
	if _, err := newFilter([]string{""}, nil); err == nil {
		t.Error("accepted empty kind")
	}
}

func TestListFiltering(t *testing.T) {
	input := `apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: Secret
  metadata:
    name: hidden
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: kept
`
	for _, kinds := range [][]string{{"Secret"}, {"Secret", "ConfigMap"}, {"List"}} {
		filter, _ := newFilter(kinds, nil)
		var output bytes.Buffer
		if err := filter.filter(strings.NewReader(input), &output); err != nil {
			t.Fatal(err)
		}
		if len(kinds) == 1 && kinds[0] == "Secret" {
			if strings.Contains(output.String(), "hidden") || !strings.Contains(output.String(), "kept") {
				t.Fatalf("incorrect List filtering: %s", output.String())
			}
		} else if output.Len() != 0 {
			t.Fatalf("expected empty output, got %s", output.String())
		}
	}
}

func TestInvalidYAML(t *testing.T) {
	filter, _ := newFilter([]string{"Secret"}, nil)
	if err := filter.filter(strings.NewReader("kind: ["), io.Discard); err == nil {
		t.Fatal("accepted invalid YAML")
	}
}

func TestHelmFailureDoesNotWriteOutput(t *testing.T) {
	dir := t.TempDir()
	helm := filepath.Join(dir, "helm")
	if err := os.WriteFile(helm, []byte("#!/bin/sh\necho 'partial output'\necho 'chart error' >&2\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	filter, _ := newFilter(nil, nil)
	output := filepath.Join(dir, "output.yaml")
	var stderr bytes.Buffer
	if err := render(helm, output, []string{"template", "test"}, filter, &stderr); err == nil {
		t.Fatal("Helm error was not propagated")
	}
	if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed render created output: %v", err)
	}
	if !strings.Contains(stderr.String(), "chart error") {
		t.Fatal("Helm stderr was lost")
	}
}

func TestRenderIsolationAndLiteralArguments(t *testing.T) {
	t.Setenv("HELM_NAMESPACE", "inherited-namespace")
	t.Setenv("HELM_PLUGINS", "/user/plugins")
	t.Setenv("KUBECONFIG", "/user/kubeconfig")
	dir := t.TempDir()
	helm := filepath.Join(dir, "helm with spaces")
	script := `#!/bin/sh
test -z "$HELM_NAMESPACE" || exit 1
test "$HELM_PLUGINS" = "$HOME/plugins" || exit 1
test "$KUBECONFIG" = "$HOME/kubeconfig" || exit 1
test "$HELM_CONFIG_HOME" = "$HOME/config" || exit 1
printf '%s\n' "$@"
`
	if err := os.WriteFile(helm, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	filter, _ := newFilter(nil, nil)
	output := filepath.Join(dir, "output with spaces.yaml")
	args := []string{"template", "example", "--set-string", "message=it's $HOME; $(false)"}
	if err := render(helm, output, args, filter, os.Stderr); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Join(args, "\n") + "\n"; string(data) != want {
		t.Fatalf("arguments or unfiltered output were changed: got %q, want %q", data, want)
	}
}
