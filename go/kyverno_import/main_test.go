package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const fakePolicyYAML = `apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-labels
spec:
  rules:
  - name: check-for-labels
    match:
      resources:
        kinds:
        - Pod
`

// resetGlobals resets package-level variables to their defaults between tests.
func resetGlobals() {
	policiesDir = ""
	baseURL = kyvernoPoliciesBaseURL
}

func TestImportPolicy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fakePolicyYAML))
	}))
	defer ts.Close()

	tmpDir := t.TempDir()

	resetGlobals()
	baseURL = ts.URL + "/"
	t.Setenv("BUILD_WORKSPACE_DIRECTORY", tmpDir)

	err := process([]string{"best-practices/require-labels"})
	if err != nil {
		t.Fatalf("process() returned unexpected error: %v", err)
	}

	outPath := filepath.Join(tmpDir, defaultPoliciesDir, "best-practices/require-labels.yaml")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file to exist at %s: %v", outPath, err)
	}

	if string(data) != fakePolicyYAML {
		t.Errorf("file content mismatch:\ngot:  %q\nwant: %q", string(data), fakePolicyYAML)
	}
}

func TestImportPolicyCreatesDirectories(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fakePolicyYAML))
	}))
	defer ts.Close()

	tmpDir := t.TempDir()

	resetGlobals()
	baseURL = ts.URL + "/"
	t.Setenv("BUILD_WORKSPACE_DIRECTORY", tmpDir)

	err := process([]string{"pod-security/baseline/disallow-privileged-containers"})
	if err != nil {
		t.Fatalf("process() returned unexpected error: %v", err)
	}

	outPath := filepath.Join(tmpDir, defaultPoliciesDir, "pod-security/baseline/disallow-privileged-containers.yaml")
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Errorf("expected output file to exist at %s", outPath)
	}
}

func TestImportPolicyHTTP404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	tmpDir := t.TempDir()

	resetGlobals()
	baseURL = ts.URL + "/"
	t.Setenv("BUILD_WORKSPACE_DIRECTORY", tmpDir)

	err := process([]string{"best-practices/nonexistent-policy"})
	if err == nil {
		t.Fatal("process() expected error for HTTP 404, got nil")
	}
}

func TestImportPolicyNoArgs(t *testing.T) {
	resetGlobals()

	err := process([]string{})
	if err == nil {
		t.Fatal("process() expected error for empty args, got nil")
	}
}
