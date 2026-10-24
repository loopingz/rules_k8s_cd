package cliutil

import (
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	env := func(k string) string {
		if k == "BUILD_WORKING_DIRECTORY" {
			return "/wd"
		}
		return ""
	}
	if got := ResolvePath(env, "a/b"); got != "/wd/a/b" {
		t.Errorf("relative: %q", got)
	}
	if got := ResolvePath(env, "/abs"); got != "/abs" {
		t.Errorf("absolute: %q", got)
	}
	if got := ResolvePath(func(string) string { return "" }, "a"); got != "a" {
		t.Errorf("no wd: %q", got)
	}
}

func TestDefaultCacheDir(t *testing.T) {
	env := func(k string) string {
		if k == "XDG_CACHE_HOME" {
			return "/xdg"
		}
		return ""
	}
	if got, want := DefaultCacheDir(env, "release"), filepath.Join("/xdg", "rules_k8s_cd", "release"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := DefaultCacheDir(func(string) string { return "" }, "release"); got != "" && filepath.Base(got) != "release" {
		t.Errorf("fallback: %q", got)
	}
}
