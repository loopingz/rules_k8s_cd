// helm_postrender reads a helm-rendered multi-doc YAML file, applies
// exclude selectors and strategic-merge / JSON6902 patches, then writes
// the result to --out. Failure on any unmatched patch is intentional.
package main

import (
	"flag"
	"fmt"
	"os"
)

type stringSlice []string

func (s *stringSlice) String() string     { return fmt.Sprintf("%v", *s) }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	var (
		in            = flag.String("in", "", "path to rendered helm YAML")
		out           = flag.String("out", "", "path to write final YAML")
		excludes      stringSlice
		patchSMs      stringSlice
		patchJSON6902 stringSlice
	)
	flag.Var(&excludes, "exclude", "exclude selector (k=v,k=v); repeatable")
	flag.Var(&patchSMs, "patch-sm", "strategic-merge patch file path; repeatable")
	flag.Var(&patchJSON6902, "patch-json6902", "JSON6902 patch file path; repeatable")
	flag.Parse()

	if *in == "" || *out == "" {
		die("helm_postrender: --in and --out are required")
	}

	inBytes, err := os.ReadFile(*in)
	if err != nil {
		die("read %s: %v", *in, err)
	}
	docs, err := SplitDocuments(inBytes)
	if err != nil {
		die("split %s: %v", *in, err)
	}

	sels := make([]Selector, 0, len(excludes))
	for _, s := range excludes {
		sel, err := ParseSelector(s)
		if err != nil {
			die("%v", err)
		}
		sels = append(sels, sel)
	}
	docs = ApplyExcludes(docs, sels)

	for _, p := range patchSMs {
		b, err := os.ReadFile(p)
		if err != nil {
			die("read %s: %v", p, err)
		}
		docs, err = ApplyStrategicMerge(docs, p, b)
		if err != nil {
			die("%v", err)
		}
	}

	for _, p := range patchJSON6902 {
		b, err := os.ReadFile(p)
		if err != nil {
			die("read %s: %v", p, err)
		}
		docs, err = ApplyJSON6902(docs, p, b)
		if err != nil {
			die("%v", err)
		}
	}

	outBytes, err := JoinDocuments(docs)
	if err != nil {
		die("join: %v", err)
	}
	if err := os.WriteFile(*out, outBytes, 0o644); err != nil {
		die("write %s: %v", *out, err)
	}
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
