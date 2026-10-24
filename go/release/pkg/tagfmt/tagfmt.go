// Package tagfmt formats and recognises release tags built from a tag format
// such as "{name}@v{version}". It is shared by the release tool and stampver
// so that both agree on which tags belong to a component.
package tagfmt

import (
	"fmt"
	"strings"

	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

const versionVar = "{version}"

// Validate checks that format contains {version} and a non-empty literal
// prefix before it (once {name} is expanded to name).
func Validate(format, name string) error {
	i := strings.Index(format, versionVar)
	if i < 0 {
		return fmt.Errorf("tag format %q must contain {version}", format)
	}
	if strings.ReplaceAll(format[:i], "{name}", name) == "" {
		return fmt.Errorf("tag format %q has an empty literal prefix before {version}", format)
	}
	return nil
}

// Format returns the tag of version v for the component name.
func Format(format, name string, v semver.Version) string {
	return strings.NewReplacer("{name}", name, versionVar, v.String()).Replace(format)
}

// Parser returns a function that extracts the version from a tag owned by
// the component: the tag has the format's literal prefix and suffix around a
// valid semver, so "api-gateway-1.0.0" is not an "api" tag for format
// "{name}-{version}". format must satisfy Validate.
func Parser(format, name string) func(tag string) (semver.Version, bool) {
	i := strings.Index(format, versionVar)
	r := strings.NewReplacer("{name}", name)
	prefix := r.Replace(format[:i])
	suffix := r.Replace(format[i+len(versionVar):])
	return func(tag string) (semver.Version, bool) {
		if len(tag) < len(prefix)+len(suffix) || !strings.HasPrefix(tag, prefix) || !strings.HasSuffix(tag, suffix) {
			return semver.Version{}, false
		}
		v, err := semver.Parse(tag[len(prefix) : len(tag)-len(suffix)])
		return v, err == nil
	}
}
