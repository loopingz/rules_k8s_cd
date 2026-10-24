// Command stampver resolves a component's version from Bazel stamp files and
// renders version-file templates (version.go/ts/py/json) from the result.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

const unknown = "unknown"

// info is the JSON document written by `version` and read by `render`.
type info struct {
	Version    string `json:"version"`
	Commit     string `json:"commit"`
	CommitFull string `json:"commit_full"`
	Date       string `json:"date"`
	DateISO    string `json:"date_iso"`
	Branch     string `json:"branch"`
	Component  string `json:"component"`
	Tagged     bool   `json:"tagged"`
}

// vars returns the placeholder values available to templates.
func (i info) vars() map[string]string {
	return map[string]string{
		"version":     i.Version,
		"commit":      i.Commit,
		"commit_full": i.CommitFull,
		"date":        i.Date,
		"date_iso":    i.DateISO,
		"branch":      i.Branch,
		"component":   i.Component,
	}
}

type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: stampver version|render [flags]")
		return 2
	}
	var err error
	switch args[0] {
	case "version":
		err = runVersion(args[1:], stderr)
	case "render":
		err = runRender(args[1:], stderr)
	default:
		fmt.Fprintf(stderr, "stampver: unknown command %q (want version|render)\n", args[0])
		return 2
	}
	if err == flag.ErrHelp {
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "stampver: %v\n", err)
		return 1
	}
	return 0
}

// parseStatus reads Bazel status lines ("KEY value...") into m. A missing
// file is not an error: builds without --stamp have no status.
func parseStatus(path string, m map[string]string) error {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		k, v := line, ""
		if i := strings.IndexByte(line, ' '); i >= 0 {
			k, v = line[:i], line[i+1:]
		}
		m[k] = v
	}
	return nil
}

// expand replaces known {placeholders} in one pass; unknown ones are kept.
func expand(tmpl string, vars map[string]string) string {
	pairs := make([]string, 0, 2*len(vars))
	for k, v := range vars {
		pairs = append(pairs, "{"+k+"}", v)
	}
	return strings.NewReplacer(pairs...).Replace(tmpl)
}

func orUnknown(s string) string {
	if s == "" {
		return unknown
	}
	return s
}

func runVersion(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("stampver version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		component = fs.String("component", "", "component name (required)")
		verFile   = fs.String("version-file", "", "VERSION file (X.Y.Z)")
		tagFormat = fs.String("tag-format", "{name}@v{version}", "tag format with {name} and {version}")
		untagged  = fs.String("untagged", "", "untagged version template (required)")
		stable    = fs.String("stable-status", "", "Bazel stable-status file")
		volatile  = fs.String("volatile-status", "", "Bazel volatile-status file")
		out       = fs.String("out", "", "output file for the version (required)")
		outJSON   = fs.String("out-json", "", "output file for the JSON info (required)")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *component == "" || *untagged == "" || *out == "" || *outJSON == "" {
		return fmt.Errorf("--component, --untagged, --out and --out-json are required")
	}

	status := map[string]string{}
	for _, p := range []string{*stable, *volatile} {
		if err := parseStatus(p, status); err != nil {
			return err
		}
	}

	in := info{
		Commit:     orUnknown(status["STABLE_GIT_COMMIT"]),
		CommitFull: orUnknown(status["STABLE_GIT_COMMIT_FULL"]),
		Date:       unknown,
		DateISO:    orUnknown(status["STABLE_GIT_COMMIT_DATE"]),
		Branch:     orUnknown(status["STABLE_GIT_BRANCH"]),
		Component:  *component,
	}
	if t, err := time.Parse(time.RFC3339, status["STABLE_GIT_COMMIT_DATE"]); err == nil {
		in.Date = t.Format("20060102")
	}

	// Candidate tags: the stamped label and every tag at HEAD.
	candidates := strings.Fields(status["STABLE_GIT_TAGS_AT_HEAD"])
	if l := status["BUILD_EMBED_LABEL"]; l != "" {
		candidates = append([]string{l}, candidates...)
	}

	ver, haveVer, err := readVersion(*verFile)
	if err != nil && ownsAny(*tagFormat, *component, candidates) {
		return err
	}
	if !haveVer && ownsAny(*tagFormat, *component, candidates) {
		return fmt.Errorf("component %q has a release tag at HEAD but no usable --version-file %q", *component, *verFile)
	}
	if haveVer {
		want := expand(*tagFormat, map[string]string{"name": *component, "version": ver.String()})
		for _, c := range candidates {
			if c == want {
				in.Version, in.Tagged = ver.String(), true
				break
			}
		}
	}
	if !in.Tagged {
		in.Version = expand(*untagged, in.vars())
	}

	if err := os.WriteFile(*out, []byte(in.Version+"\n"), 0o644); err != nil {
		return err
	}
	b, err := json.MarshalIndent(in, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(*outJSON, append(b, '\n'), 0o644)
}

// readVersion parses the VERSION file; a missing file reports ok=false with a
// nil error, an unreadable or invalid one returns the error.
func readVersion(path string) (v semver.Version, ok bool, err error) {
	if path == "" {
		return v, false, nil
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return v, false, nil
	}
	if err != nil {
		return v, false, err
	}
	v, err = semver.Parse(strings.TrimSpace(string(b)))
	if err != nil {
		return v, false, fmt.Errorf("%s: %w", path, err)
	}
	return v, true, nil
}

// ownsAny reports whether any candidate looks like a tag of this component,
// i.e. starts with the tag format up to {version}. Only then is VERSION needed.
func ownsAny(tagFormat, component string, candidates []string) bool {
	prefix := expand(tagFormat, map[string]string{"name": component})
	if i := strings.Index(prefix, "{version}"); i >= 0 {
		prefix = prefix[:i]
	}
	for _, c := range candidates {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func runRender(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("stampver render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var subs stringSlice
	var (
		infoPath = fs.String("info", "", "JSON info file from `stampver version` (required)")
		tmplPath = fs.String("template", "", "template file (required)")
		out      = fs.String("out", "", "output file (required)")
	)
	fs.Var(&subs, "sub", "extra KEY=VALUE substitution (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *infoPath == "" || *tmplPath == "" || *out == "" {
		return fmt.Errorf("--info, --template and --out are required")
	}
	vars := map[string]string{}
	for _, s := range subs {
		i := strings.IndexByte(s, '=')
		if i <= 0 {
			return fmt.Errorf("invalid --sub %q: want KEY=VALUE", s)
		}
		vars[s[:i]] = s[i+1:]
	}
	b, err := os.ReadFile(*infoPath)
	if err != nil {
		return err
	}
	var in info
	if err := json.Unmarshal(b, &in); err != nil {
		return fmt.Errorf("%s: %w", *infoPath, err)
	}
	// --sub keys override info keys.
	merged := in.vars()
	for k, v := range vars {
		merged[k] = v
	}
	tmpl, err := os.ReadFile(*tmplPath)
	if err != nil {
		return err
	}
	return os.WriteFile(*out, []byte(expand(string(tmpl), merged)), 0o644)
}
