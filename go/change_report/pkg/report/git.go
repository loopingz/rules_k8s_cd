package report

import (
	"fmt"

	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
)

// GitAccessors returns the Input.FileAt and Input.Tracked functions backed by
// repo. A read error is reported once through warn (which may be nil) and the
// file is treated as absent; a Tracked error means untracked.
func GitAccessors(repo gitx.Repo, warn func(string)) (fileAt func(rev, path string) ([]byte, bool), tracked func(path string) bool) {
	warned := false
	fileAt = func(rev, p string) ([]byte, bool) {
		b, ok, err := repo.Show(rev, p)
		if err != nil {
			if !warned && warn != nil {
				warned = true
				warn(fmt.Sprintf("cannot read %s at %s: %v; treating as absent", p, rev, err))
			}
			return nil, false
		}
		return b, ok
	}
	tracked = func(p string) bool {
		ok, err := repo.Tracked(p)
		return err == nil && ok
	}
	return fileAt, tracked
}
