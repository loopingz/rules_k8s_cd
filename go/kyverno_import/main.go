package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	kyvernoPoliciesBaseURL = "https://raw.githubusercontent.com/kyverno/policies/main/"
	defaultPoliciesDir     = "kyverno/policies"
)

var (
	policiesDir string
	baseURL     = kyvernoPoliciesBaseURL
)

func process(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Usage: kyverno_import <policy-path>\n\nExample: kyverno_import best-practices/require-labels")
	}

	policyPath := args[0]

	// Determine output directory
	outDir := defaultPoliciesDir
	if policiesDir != "" {
		outDir = policiesDir
	}

	// Determine the workspace directory
	wsDir := os.Getenv("BUILD_WORKSPACE_DIRECTORY")
	if wsDir == "" {
		wsDir = "."
	}

	// Build the download URL
	// Policy path like "best-practices/require-labels" maps to
	// kyverno/policies/main/best-practices/require-labels/require-labels.yaml
	parts := strings.Split(policyPath, "/")
	policyName := parts[len(parts)-1]
	url := baseURL + policyPath + "/" + policyName + ".yaml"

	// Download the policy
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download policy %q: HTTP %d", policyPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Write to output directory
	outPath := filepath.Join(wsDir, outDir, policyPath+".yaml")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outPath, body, 0644); err != nil {
		return fmt.Errorf("failed to write policy file: %w", err)
	}

	fmt.Printf("Saved %s\n", outPath)
	return nil
}

func main() {
	if err := process(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
