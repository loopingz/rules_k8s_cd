package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// resetGlobals resets all package-level flag variables to their zero values between tests.
func resetGlobals() {
	paths = arrayFlags{}
	images = arrayFlags{}
	vars = arrayFlags{}
	output = ""
	combine = ""
	input = ""
	repository = ""
	relativePath = ""
}

// writeFile is a test helper that writes content to a file and returns the path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile %s: %v", name, err)
	}
	return p
}

// readYAML parses a YAML file into a map for assertions.
func readYAML(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readYAML: %v", err)
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(b, &m); err != nil {
		t.Fatalf("readYAML unmarshal: %v", err)
	}
	return m
}

// TestReplaceVars checks that variable substitution works on byte content.
func TestReplaceVars(t *testing.T) {
	resetGlobals()
	vars = arrayFlags{"{{NAMESPACE}}:production", "{{TAG}}:v1.2.3"}

	input := []byte("namespace: {{NAMESPACE}}\nimage: myapp:{{TAG}}")
	got := replaceVars(input)

	if string(got) != "namespace: production\nimage: myapp:v1.2.3" {
		t.Errorf("replaceVars: got %q", string(got))
	}
}

// TestReplaceVarsNoMatch ensures content is unchanged when no vars match.
func TestReplaceVarsNoMatch(t *testing.T) {
	resetGlobals()
	vars = arrayFlags{"{{FOO}}:bar"}

	original := []byte("nothing to replace here")
	got := replaceVars(original)
	if string(got) != string(original) {
		t.Errorf("expected unchanged content, got %q", string(got))
	}
}

// TestImageInjectionOciPushInfo verifies that an oci_push_info:// image is injected correctly.
func TestImageInjectionOciPushInfo(t *testing.T) {
	dir := t.TempDir()

	// Create a fake push info file containing the digest.
	pushInfoPath := writeFile(t, dir, "push_info.txt", "sha256:abc123\n")

	inputPath := writeFile(t, dir, "kustomization.yaml", `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
`)
	outputPath := filepath.Join(dir, "out.yaml")

	resetGlobals()
	input = inputPath
	output = outputPath
	repository = "registry.example.com/"
	images = arrayFlags{"myapp:oci_push_info://" + pushInfoPath}

	process()

	m := readYAML(t, outputPath)
	rawImages, ok := m["images"]
	if !ok {
		t.Fatal("expected 'images' key in output")
	}
	imgList, ok := rawImages.([]interface{})
	if !ok || len(imgList) == 0 {
		t.Fatal("expected non-empty images list")
	}
	img, ok := imgList[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected image entry to be a map")
	}
	if img["name"] != "myapp" {
		t.Errorf("expected name=myapp, got %v", img["name"])
	}
	if img["newName"] != "registry.example.com/myapp" {
		t.Errorf("expected newName=registry.example.com/myapp, got %v", img["newName"])
	}
	if img["digest"] != "sha256:abc123" {
		t.Errorf("expected digest=sha256:abc123, got %v", img["digest"])
	}
}

// TestImageInjectionRefProtocol verifies that a ref:// image is injected correctly.
func TestImageInjectionRefProtocol(t *testing.T) {
	dir := t.TempDir()

	inputPath := writeFile(t, dir, "kustomization.yaml", `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
`)
	outputPath := filepath.Join(dir, "out.yaml")

	resetGlobals()
	input = inputPath
	output = outputPath
	images = arrayFlags{"myapp:ref://other.registry.io/myapp@sha256:deadbeef"}

	process()

	m := readYAML(t, outputPath)
	rawImages, ok := m["images"]
	if !ok {
		t.Fatal("expected 'images' key in output")
	}
	imgList := rawImages.([]interface{})
	img := imgList[0].(map[string]interface{})
	if img["name"] != "myapp" {
		t.Errorf("expected name=myapp, got %v", img["name"])
	}
	if img["newName"] != "other.registry.io/myapp" {
		t.Errorf("expected newName=other.registry.io/myapp, got %v", img["newName"])
	}
	if img["digest"] != "sha256:deadbeef" {
		t.Errorf("expected digest=sha256:deadbeef, got %v", img["digest"])
	}
}

// TestResourcePathInjection verifies that a resource path is appended to the resources list.
func TestResourcePathInjection(t *testing.T) {
	dir := t.TempDir()

	inputPath := writeFile(t, dir, "kustomization.yaml", `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources: []
`)
	outputPath := filepath.Join(dir, "out.yaml")

	resetGlobals()
	input = inputPath
	output = outputPath
	paths = arrayFlags{"resources:deployment.yaml"}

	process()

	m := readYAML(t, outputPath)
	rawResources, ok := m["resources"]
	if !ok {
		t.Fatal("expected 'resources' key in output")
	}
	resList := rawResources.([]interface{})
	if len(resList) == 0 {
		t.Fatal("expected non-empty resources list")
	}
	if resList[0] != "deployment.yaml" {
		t.Errorf("expected resources[0]=deployment.yaml, got %v", resList[0])
	}
}

// TestResourcePathRelativeStripping verifies that relativePath prefix is stripped from resource paths.
func TestResourcePathRelativeStripping(t *testing.T) {
	dir := t.TempDir()

	inputPath := writeFile(t, dir, "kustomization.yaml", `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources: []
`)
	outputPath := filepath.Join(dir, "out.yaml")

	resetGlobals()
	input = inputPath
	output = outputPath
	relativePath = "/some/prefix/"
	paths = arrayFlags{"resources:/some/prefix/deployment.yaml"}

	process()

	m := readYAML(t, outputPath)
	resList := m["resources"].([]interface{})
	if resList[0] != "deployment.yaml" {
		t.Errorf("expected stripped path deployment.yaml, got %v", resList[0])
	}
}

// TestVarSubstitutionEndToEnd verifies that vars are substituted in the final output.
func TestVarSubstitutionEndToEnd(t *testing.T) {
	dir := t.TempDir()

	inputPath := writeFile(t, dir, "kustomization.yaml", `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: "{{NAMESPACE}}"
`)
	outputPath := filepath.Join(dir, "out.yaml")

	resetGlobals()
	input = inputPath
	output = outputPath
	vars = arrayFlags{"{{NAMESPACE}}:production"}

	process()

	b, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	content := string(b)
	if !strings.Contains(content, "production") {
		t.Errorf("expected 'production' in output, got: %s", content)
	}
	if strings.Contains(content, "{{NAMESPACE}}") {
		t.Errorf("expected {{NAMESPACE}} to be replaced, but found it in: %s", content)
	}
}

