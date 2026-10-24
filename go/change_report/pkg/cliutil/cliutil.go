// Package cliutil holds helpers shared by the command line tools.
package cliutil

import (
	"os"
	"path/filepath"
)

// ResolvePath makes a relative path absolute against BUILD_WORKING_DIRECTORY
// (set by `bazel run`), else leaves it for the process cwd.
func ResolvePath(env func(string) string, p string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	if wd := env("BUILD_WORKING_DIRECTORY"); wd != "" {
		return filepath.Join(wd, p)
	}
	return p
}

// DefaultCacheDir returns <cache root>/rules_k8s_cd/<tool>, where the cache
// root is $XDG_CACHE_HOME, else the user cache dir. It is "" when neither is
// known (caching is then disabled).
func DefaultCacheDir(env func(string) string, tool string) string {
	if x := env("XDG_CACHE_HOME"); x != "" {
		return filepath.Join(x, "rules_k8s_cd", tool)
	}
	if uc, err := os.UserCacheDir(); err == nil {
		return filepath.Join(uc, "rules_k8s_cd", tool)
	}
	return ""
}
