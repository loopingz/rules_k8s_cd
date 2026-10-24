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

	if !strings.Contains(*tagFormat, "{version}") {
		return fmt.Errorf("--tag-format %q must contain {version}", *tagFormat)
	}
	if expand((*tagFormat)[:strings.Index(*tagFormat, "{version}")], map[string]string{"name": *component}) == "" {
		return fmt.Errorf("--tag-format %q has an empty literal prefix before {version}", *tagFormat)
	}
	owns := ownership(*tagFormat, *component)
	label := status["BUILD_EMBED_LABEL"]
	var ownedTags []string
	for _, t := range strings.Fields(status["STABLE_GIT_TAGS_AT_HEAD"]) {
		if owns(t) {
			ownedTags = append(ownedTags, t)
		}
	}
	ownedLabel := label != "" && owns(label)
	needed := ownedLabel || len(ownedTags) > 0

	ver, haveVer, err := readVersion(*verFile)
	if err != nil {
		if needed {
			return err
		}
		fmt.Fprintf(stderr, "stampver: warning: ignoring invalid version file: %v\n", err)
	} else if !haveVer && needed {
		return fmt.Errorf("component %q has a release tag at HEAD but no usable --version-file %q", *component, *verFile)
	}
	if haveVer {
		want := expand(*tagFormat, map[string]string{"name": *component, "version": ver.String()})
		switch {
		case ownedLabel && label != want:
			return fmt.Errorf("BUILD_EMBED_LABEL %q does not match the expected tag %q for VERSION %s", label, want, ver)
		case ownedLabel:
			in.Version, in.Tagged = ver.String(), true
		default:
			for _, t := range ownedTags {
				if t == want {
					in.Version, in.Tagged = ver.String(), true
				}
			}
			if !in.Tagged && len(ownedTags) > 0 {
				fmt.Fprintf(stderr, "stampver: warning: tags %v at HEAD do not match expected tag %q for VERSION %s; building untagged\n", ownedTags, want, ver)
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

// ownership returns a predicate telling whether a tag belongs to component:
// it has the format's literal prefix and suffix around a valid semver, so
// "api-gateway-1.0.0" is not an "api" tag for format "{name}-{version}".
func ownership(tagFormat, component string) func(string) bool {
	i := strings.Index(tagFormat, "{version}")
	vars := map[string]string{"name": component}
	prefix := expand(tagFormat[:i], vars)
	suffix := expand(tagFormat[i+len("{version}"):], vars)
	return func(tag string) bool {
		if !strings.HasPrefix(tag, prefix) || !strings.HasSuffix(tag, suffix) || len(tag) < len(prefix)+len(suffix) {
			return false
		}
		_, err := semver.Parse(tag[len(prefix) : len(tag)-len(suffix)])
		return err == nil
	}
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
