package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type stringsFlag []string

func (s *stringsFlag) String() string { return strings.Join(*s, ",") }
func (s *stringsFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type resourceFilter struct {
	kinds     map[string]bool
	resources [][]string
}

func newFilter(kinds, resources []string) (*resourceFilter, error) {
	f := &resourceFilter{kinds: make(map[string]bool)}
	for _, kind := range kinds {
		if kind == "" {
			return nil, fmt.Errorf("excluded kind must not be empty")
		}
		f.kinds[kind] = true
	}
	for _, resource := range resources {
		parts := strings.Split(resource, "/")
		if (len(parts) != 2 && len(parts) != 3) || parts[0] == "" || parts[len(parts)-1] == "" {
			return nil, fmt.Errorf("invalid resource filter %q: expected Kind/name or Kind/namespace/name", resource)
		}
		for _, part := range parts {
			if _, err := path.Match(part, ""); err != nil {
				return nil, fmt.Errorf("invalid resource filter %q: %w", resource, err)
			}
		}
		f.resources = append(f.resources, parts)
	}
	return f, nil
}

func field(node *yaml.Node, key string) *yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind == yaml.AliasNode {
		return field(node.Alias, key)
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == key {
				return node.Content[i+1]
			}
		}
	}
	return &yaml.Node{}
}

func (f *resourceFilter) excluded(node *yaml.Node) bool {
	kind := field(node, "kind").Value
	if f.kinds[kind] {
		return true
	}
	metadata := field(node, "metadata")
	name, namespace := field(metadata, "name").Value, field(metadata, "namespace").Value
	for _, pattern := range f.resources {
		values := []string{kind, name}
		if len(pattern) == 3 {
			values = []string{kind, namespace, name}
		}
		matches := true
		for i, part := range pattern {
			match, _ := path.Match(part, values[i]) // Patterns are validated in newFilter.
			matches = matches && match
		}
		if matches {
			return true
		}
	}
	return false
}

// Kubernetes List wrappers can contain resources that also need filtering.
func (f *resourceFilter) keep(node *yaml.Node) bool {
	if f.excluded(node) {
		return false
	}
	if field(node, "kind").Value == "List" {
		items := field(node, "items")
		if items.Kind == yaml.SequenceNode {
			kept := make([]*yaml.Node, 0, len(items.Content))
			for _, item := range items.Content {
				if f.keep(item) {
					kept = append(kept, item)
				}
			}
			items.Content = kept
			return len(kept) > 0
		}
	}
	return true
}

func (f *resourceFilter) filter(input io.Reader, output io.Writer) error {
	decoder := yaml.NewDecoder(input)
	encoder := yaml.NewEncoder(output)
	encoder.SetIndent(2)
	defer encoder.Close()
	for {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("decode rendered YAML: %w", err)
		}
		if len(doc.Content) == 0 || doc.Content[0].Tag == "!!null" {
			continue
		}
		if f.keep(&doc) {
			if err := encoder.Encode(&doc); err != nil {
				return fmt.Errorf("encode rendered YAML: %w", err)
			}
		}
	}
}

func render(helm, output string, args []string, filter *resourceFilter, stderr io.Writer) error {
	config, err := os.MkdirTemp("", "helm-template-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(config)
	cmd := exec.Command(helm, args...)
	// Use isolated directories for every Helm action and disable user plugins.
	// Preserve execution environment variables such as PATH and TMPDIR.
	for _, entry := range os.Environ() {
		key := strings.SplitN(entry, "=", 2)[0]
		if !strings.HasPrefix(key, "HELM_") && key != "KUBECONFIG" && key != "HOME" && key != "XDG_CONFIG_HOME" && key != "XDG_CACHE_HOME" && key != "XDG_DATA_HOME" {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env,
		"HOME="+config,
		"HELM_CONFIG_HOME="+filepath.Join(config, "config"),
		"HELM_CACHE_HOME="+filepath.Join(config, "cache"),
		"HELM_DATA_HOME="+filepath.Join(config, "data"),
		"HELM_PLUGINS="+filepath.Join(config, "plugins"),
		"KUBECONFIG="+filepath.Join(config, "kubeconfig"),
	)
	var rendered bytes.Buffer
	cmd.Stdout, cmd.Stderr = &rendered, stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm template failed: %w", err)
	}
	var result bytes.Buffer
	if len(filter.kinds) == 0 && len(filter.resources) == 0 {
		result = rendered // Preserve Helm formatting when no filters were requested.
	} else if err := filter.filter(&rendered, &result); err != nil {
		return err
	}
	return os.WriteFile(output, result.Bytes(), 0644)
}

func main() {
	var helm, output string
	var kinds, resources stringsFlag
	flag.StringVar(&helm, "helm", "", "Helm executable")
	flag.StringVar(&output, "output", "", "Output YAML file")
	flag.Var(&kinds, "exclude-kind", "Kind to exclude (repeatable)")
	flag.Var(&resources, "exclude-resource", "Kind/name or Kind/namespace/name glob to exclude (repeatable)")
	flag.Parse()
	if helm == "" || output == "" || len(flag.Args()) == 0 {
		log.Fatal("--helm, --output, and Helm arguments after -- are required")
	}
	filter, err := newFilter(kinds, resources)
	if err != nil {
		log.Fatal(err)
	}
	if err := render(helm, output, flag.Args(), filter, os.Stderr); err != nil {
		log.Fatal(err)
	}
}
