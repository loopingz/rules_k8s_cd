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

var policiesDir string

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatal("Usage: kyverno_import <policy-path>\n\nExample: kyverno_import best-practices/require-labels")
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
	url := kyvernoPoliciesBaseURL + policyPath + "/" + policyName + ".yaml"

	// Download the policy
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to download policy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Failed to download policy %q: HTTP %d", policyPath, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	// Write to output directory
	outPath := filepath.Join(wsDir, outDir, policyPath+".yaml")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(outPath, body, 0644); err != nil {
		log.Fatalf("Failed to write policy file: %v", err)
	}

	fmt.Printf("Saved %s\n", outPath)
}
