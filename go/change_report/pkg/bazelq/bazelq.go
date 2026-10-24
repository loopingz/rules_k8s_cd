// Package bazelq runs and parses `bazel query` output.
package bazelq

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
)

// Runner runs the bazel binary with args and returns its stdout.
type Runner interface {
	Run(args ...string) ([]byte, error)
}

// ExecRunner runs Bazel in Dir.
type ExecRunner struct {
	Bazel, Dir string
}

// Run executes `Bazel args...` in Dir, returning stdout. stderr is included in the error.
func (e ExecRunner) Run(args ...string) ([]byte, error) {
	bin := e.Bazel
	if bin == "" {
		bin = "bazel"
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = e.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", bin, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// Node is a target in the query result.
type Node struct {
	Label          string
	Kind           string // "rule", "source" or "generated"
	RuleClass      string
	Inputs         []string
	GeneratingRule string
}

type entry struct {
	Type string `json:"type"`
	Rule *struct {
		Name      string   `json:"name"`
		RuleClass string   `json:"ruleClass"`
		RuleInput []string `json:"ruleInput"`
	} `json:"rule"`
	SourceFile *struct {
		Name string `json:"name"`
	} `json:"sourceFile"`
	GeneratedFile *struct {
		Name           string `json:"name"`
		GeneratingRule string `json:"generatingRule"`
	} `json:"generatedFile"`
}

// Parse reads streamed_jsonproto output (one JSON object per line).
// Unknown entry types are skipped.
func Parse(r io.Reader) ([]Node, error) {
	dec := json.NewDecoder(bufio.NewReaderSize(r, 1<<20))
	var nodes []Node
	for {
		var e entry
		if err := dec.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("parsing bazel query output: %w", err)
		}
		switch {
		case e.Type == "RULE" && e.Rule != nil:
			nodes = append(nodes, Node{Label: e.Rule.Name, Kind: "rule", RuleClass: e.Rule.RuleClass, Inputs: e.Rule.RuleInput})
		case e.Type == "SOURCE_FILE" && e.SourceFile != nil:
			nodes = append(nodes, Node{Label: e.SourceFile.Name, Kind: "source"})
		case e.Type == "GENERATED_FILE" && e.GeneratedFile != nil:
			nodes = append(nodes, Node{Label: e.GeneratedFile.Name, Kind: "generated", GeneratingRule: e.GeneratedFile.GeneratingRule})
		}
	}
	return nodes, nil
}

// Query runs a bazel query and parses the result. Any failure, including
// exit code 3 (partial results), is an error: a partial graph could hide changes.
func Query(r Runner, expr string) ([]Node, error) {
	out, err := r.Run("query", "--consistent_labels", "--output=streamed_jsonproto", "--keep_going", expr)
	if err != nil {
		return nil, err
	}
	return Parse(bytes.NewReader(out))
}

// Labels runs a bazel query with --output=label and returns the sorted labels.
func Labels(r Runner, expr string) ([]string, error) {
	out, err := r.Run("query", "--consistent_labels", "--output=label", expr)
	if err != nil {
		return nil, err
	}
	var labels []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			labels = append(labels, l)
		}
	}
	sort.Strings(labels)
	return labels, nil
}

// Version returns the "Build label:" value of `bazel version`.
func Version(r Runner) (string, error) {
	out, err := r.Run("version")
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(l), "Build label:"); ok {
			return strings.TrimSpace(v), nil
		}
	}
	return "", errors.New("bazel version: no \"Build label:\" line in output")
}
