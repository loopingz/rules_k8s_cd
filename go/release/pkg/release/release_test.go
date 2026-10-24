package release

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"loopingz.com/rules_k8s_cd/change_report/pkg/cache"
	"loopingz.com/rules_k8s_cd/change_report/pkg/gitx"
	"loopingz.com/rules_k8s_cd/change_report/pkg/report"
	"loopingz.com/rules_k8s_cd/release/pkg/attrib"
	"loopingz.com/rules_k8s_cd/release/pkg/component"
	"loopingz.com/rules_k8s_cd/release/pkg/forge"
	"loopingz.com/rules_k8s_cd/release/pkg/semver"
)

// --- git helpers -----------------------------------------------------------

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, dir, path, content string) {
	t.Helper()
	p := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type fixture struct {
	t            *testing.T
	dir, origin  string
	forge        *fakeForge
	out, warns   []string
	runs         []targetRun
	failTarget   string
	retryTargets bool
	dryRun       bool
	noForge      bool
}

type targetRun struct {
	Target string
	Env    []string
	Dir    string
	// Version is the content of a/VERSION in Dir when the target ran.
	Version string
}

// setup creates a repository with components a and b released at 1.0.0
// (tagged), a bare origin with main pushed, and refs/remotes/origin/main.
func setup(t *testing.T) *fixture {
	t.Helper()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_AUTHOR_NAME", "T")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "T")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	f := &fixture{t: t, dir: t.TempDir(), origin: t.TempDir(), forge: newFakeForge()}
	git(t, f.origin, "init", "-q", "--bare")
	git(t, f.dir, "init", "-q")
	git(t, f.dir, "remote", "add", "origin", f.origin)
	write(t, f.dir, "a/VERSION", "1.0.0\n")
	write(t, f.dir, "a/CHANGELOG.md", "# Changelog\n\n## 1.0.0 (2026-01-01)\n\n- old\n")
	write(t, f.dir, "a/version.go", "package a\n\nconst Version = \"1.0.0\"\n")
	write(t, f.dir, "b/VERSION", "1.0.0\n")
	write(t, f.dir, "b/b.txt", "b\n")
	write(t, f.dir, "lib/l.txt", "l\n")
	git(t, f.dir, "add", "-A")
	git(t, f.dir, "commit", "-q", "-m", "init")
	git(t, f.dir, "tag", "a@v1.0.0")
	git(t, f.dir, "tag", "b@v1.0.0")
	f.push()
	git(t, f.dir, "push", "-q", "origin", "--tags")
	return f
}

// commit writes files and commits them on main, then pushes main.
func (f *fixture) commit(msg string, files map[string]string) string {
	f.t.Helper()
	for p, c := range files {
		write(f.t, f.dir, p, c)
	}
	git(f.t, f.dir, "add", "-A")
	git(f.t, f.dir, "commit", "-q", "-m", msg)
	f.push()
	return git(f.t, f.dir, "rev-parse", "HEAD")
}

func (f *fixture) push() {
	f.t.Helper()
	git(f.t, f.dir, "push", "-q", "origin", "HEAD:refs/heads/main")
	git(f.t, f.dir, "fetch", "-q", "origin")
}

func comps() []component.Component {
	return []component.Component{
		{
			Label: "//b:b", Name: "b", Package: "b", VersionFile: "b/VERSION",
			TagFormat: "{name}@v{version}", Initial: semver.Version{Minor: 1},
		},
		{
			Label: "//a:a", Name: "a", Package: "a", VersionFile: "a/VERSION", Changelog: "a/CHANGELOG.md",
			TagFormat: "{name}@v{version}", Initial: semver.Version{Minor: 1},
			BumpFiles:      map[string]string{"a/version.go": `Version = "(\d+\.\d+\.\d+)"`},
			ReleaseTargets: []string{"//a:push", "//a:publish"},
		},
	}
}

// fakeAttribute attributes to a component every non-release commit since
// its baseline touching its package or lib/ (except its version file and
// changelog).
func fakeAttribute(o cache.Options, repo gitx.Repo, head string, cs []component.Component, states map[string]component.State, _ []string) (map[string]attrib.Result, []string, error) {
	out := map[string]attrib.Result{}
	for _, c := range cs {
		from := states[c.Label].BaselineSHA
		if from == report.RootBaseline {
			from = ""
		}
		log, err := repo.Log(from, head)
		if err != nil {
			return nil, nil, err
		}
		r := attrib.Result{Component: c, Files: map[string][]string{}}
		for _, cm := range log {
			if attrib.IsReleaseCommit(cm) {
				continue
			}
			var files []string
			for _, fc := range cm.Files {
				if fc.Path == c.VersionFile || fc.Path == c.Changelog {
					continue
				}
				if strings.HasPrefix(fc.Path, c.Package+"/") || strings.HasPrefix(fc.Path, "lib/") {
					files = append(files, fc.Path)
				}
			}
			if len(files) > 0 {
				sort.Strings(files)
				r.Commits = append(r.Commits, cm)
				r.Files[cm.SHA] = files
			}
		}
		out[c.Label] = r
	}
	return out, []string{"attribution warning"}, nil
}

func (f *fixture) env() Env {
	e := Env{
		Repo:         gitx.Repo{Dir: f.dir},
		Remote:       "origin",
		MainBranch:   "main",
		CommitURL:    "https://example.com/c/{sha}",
		Cache:        cache.Options{CacheDir: filepath.Join(f.dir, ".cache-not-used")},
		DryRun:       f.dryRun,
		RetryTargets: f.retryTargets,
		Now:          func() time.Time { return time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC) },
		Out:          func(s string) { f.out = append(f.out, s) },
		Warn:         func(s string) { f.warns = append(f.warns, s) },
		Attribute:    fakeAttribute,
		RunTarget: func(target string, env []string, dir string) error {
			v, _ := os.ReadFile(filepath.Join(dir, "a/VERSION"))
			f.runs = append(f.runs, targetRun{target, env, dir, string(v)})
			if target == f.failTarget {
				return fmt.Errorf("boom")
			}
			return nil
		},
	}
	e.Cache.CacheDir = f.t.TempDir()
	if !f.noForge {
		e.Forge = f.forge
	}
	return e
}

func (f *fixture) remoteRef(ref string) string {
	f.t.Helper()
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", ref)
	cmd.Dir = f.origin
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func (f *fixture) remoteFile(rev, path string) string {
	f.t.Helper()
	return git(f.t, f.origin, "show", rev+":"+path)
}

// --- fake forge ------------------------------------------------------------

type fakeForge struct {
	prs   map[string]*forge.PR // open PRs by head
	next  int
	calls []string
	// failRelease makes CreateRelease fail.
	failRelease bool
	releases    map[string]string // tag -> notes
}

func newFakeForge() *fakeForge {
	return &fakeForge{prs: map[string]*forge.PR{}, next: 1, releases: map[string]string{}}
}

func (f *fakeForge) FindOpenPR(head string) (*forge.PR, error) {
	f.calls = append(f.calls, "find "+head)
	return f.prs[head], nil
}

func (f *fakeForge) CreatePR(head, base, title, body string) (*forge.PR, error) {
	f.calls = append(f.calls, fmt.Sprintf("create %s->%s %q", head, base, title))
	if f.prs[head] != nil {
		return nil, fmt.Errorf("a pull request already exists for %s", head)
	}
	pr := &forge.PR{Number: f.next, Head: head, Base: base, Title: title, Body: body}
	f.next++
	f.prs[head] = pr
	return pr, nil
}

func (f *fakeForge) UpdatePR(number int, title, body string) error {
	f.calls = append(f.calls, fmt.Sprintf("update #%d %q", number, title))
	for _, pr := range f.prs {
		if pr.Number == number {
			pr.Title, pr.Body = title, body
		}
	}
	return nil
}

func (f *fakeForge) ClosePR(number int, comment string) error {
	f.calls = append(f.calls, fmt.Sprintf("close #%d %q", number, comment))
	for h, pr := range f.prs {
		if pr.Number == number {
			delete(f.prs, h)
		}
	}
	return nil
}

func (f *fakeForge) ReleaseExists(tag string) (bool, error) {
	f.calls = append(f.calls, "exists "+tag)
	_, ok := f.releases[tag]
	return ok, nil
}

func (f *fakeForge) CreateRelease(tag, name, notes string) error {
	if f.failRelease {
		return fmt.Errorf("forge down")
	}
	f.calls = append(f.calls, fmt.Sprintf("release %s %s", tag, name))
	f.releases[tag] = notes
	return nil
}

// writes returns the recorded calls that change forge state.
func (f *fakeForge) writes() []string {
	var out []string
	for _, c := range f.calls {
		if !strings.HasPrefix(c, "find ") && !strings.HasPrefix(c, "exists ") {
			out = append(out, c)
		}
	}
	return out
}

func contains(xs []string, sub string) bool {
	for _, x := range xs {
		if strings.Contains(x, sub) {
			return true
		}
	}
	return false
}

// --- Status ----------------------------------------------------------------

func TestStatus(t *testing.T) {
	f := setup(t)
	feat := f.commit("feat(api): add thing", map[string]string{"a/x.go": "x\n"})
	libc := f.commit("fix: shared lib", map[string]string{"lib/l.txt": "l2\n"})
	f.commit("chore: tidy b", map[string]string{"b/b.txt": "b2\n"})

	plans, err := Status(f.env(), comps())
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 2 || plans[0].Component.Name != "a" || plans[1].Component.Name != "b" {
		t.Fatalf("plans not in name order: %+v", plans)
	}
	a, b := plans[0], plans[1]
	if !a.Release || a.Next != (semver.Version{Major: 1, Minor: 1}) {
		t.Errorf("a: release=%v next=%v", a.Release, a.Next)
	}
	if len(a.Entries) != 2 || a.Entries[0].Commit.SHA != libc || a.Entries[1].Commit.SHA != feat {
		t.Fatalf("a entries: %+v", a.Entries)
	}
	if !reflect.DeepEqual(a.Entries[0].Via, []string{"lib/l.txt"}) || a.Entries[1].Via != nil {
		t.Errorf("via: %v / %v", a.Entries[0].Via, a.Entries[1].Via)
	}
	for _, s := range []string{"## 1.1.0 (2026-10-24)", "### Features", "**api:** add thing", "via `lib/l.txt`", "https://example.com/c/" + feat} {
		if !strings.Contains(a.Section, s) {
			t.Errorf("section lacks %q:\n%s", s, a.Section)
		}
	}
	if !b.Release || b.Next != (semver.Version{Major: 1, Patch: 1}) {
		t.Errorf("b: release=%v next=%v (lib fix is a patch)", b.Release, b.Next)
	}
	if !contains(f.warns, "attribution warning") {
		t.Errorf("attribution warnings not forwarded: %v", f.warns)
	}
}

// --- PR --------------------------------------------------------------------

func TestPRCreatesBranchAndPR(t *testing.T) {
	f := setup(t)
	head := f.commit("feat: add thing", map[string]string{"a/x.go": "x\n"})
	statusBefore := git(t, f.dir, "status", "--porcelain")

	if err := PR(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	want := []string{`create release/a->main "chore(release): a 1.1.0"`}
	if got := f.forge.writes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("forge writes = %v, want %v", got, want)
	}
	pr := f.forge.prs["release/a"]
	if !strings.HasPrefix(pr.Body, "<!-- rules_k8s_cd:release a -->\n\n## 1.1.0 (2026-10-24)") {
		t.Errorf("body:\n%s", pr.Body)
	}
	br := "refs/heads/release/a"
	if f.remoteRef(br) == "" {
		t.Fatal("release/a not pushed")
	}
	if got := git(t, f.origin, "log", "-1", "--format=%s%n%P", br); got != "chore(release): a 1.1.0\n"+head {
		t.Errorf("release commit subject/parent = %q", got)
	}
	if got := f.remoteFile(br, "a/VERSION"); got != "1.1.0" {
		t.Errorf("VERSION = %q", got)
	}
	if got := f.remoteFile(br, "a/version.go"); !strings.Contains(got, `const Version = "1.1.0"`) {
		t.Errorf("bump file = %q", got)
	}
	cl := f.remoteFile(br, "a/CHANGELOG.md")
	if !strings.HasPrefix(cl, "# Changelog\n\n## 1.1.0 (2026-10-24)") || !strings.Contains(cl, "## 1.0.0") {
		t.Errorf("changelog:\n%s", cl)
	}
	if got := git(t, f.dir, "status", "--porcelain"); got != statusBefore {
		t.Errorf("working tree changed: %q", got)
	}
	if git(t, f.dir, "rev-parse", "HEAD") != head {
		t.Error("HEAD moved")
	}

	// Second run updates the same PR.
	f.commit("fix: another", map[string]string{"a/y.go": "y\n"})
	if err := PR(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	want = append(want, `update #1 "chore(release): a 1.1.0"`)
	if got := f.forge.writes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("forge writes = %v, want %v", got, want)
	}
	if !strings.Contains(f.forge.prs["release/a"].Body, "another") {
		t.Error("PR body not updated")
	}
	if !strings.Contains(f.remoteFile(br, "a/CHANGELOG.md"), "another") {
		t.Error("branch not updated")
	}
}

func TestPRReleaseBaseTrailer(t *testing.T) {
	f := setup(t)
	head := f.commit("feat: add thing", map[string]string{"a/x.go": "x\n"})
	if err := PR(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	if got, want := git(t, f.origin, "log", "-1", "--format=%B", "refs/heads/release/a"), "chore(release): a 1.1.0\n\nRelease-Base: "+head; got != want {
		t.Errorf("release commit message %q, want %q", got, want)
	}
	if body := f.forge.prs["release/a"].Body; !strings.HasSuffix(body, "\n\nRelease-Base: "+head+"\n") {
		t.Errorf("PR body lacks the trailer:\n%s", body)
	}
}

func TestPRNoChangesClosesPR(t *testing.T) {
	f := setup(t)
	f.forge.prs["release/a"] = &forge.PR{Number: 7, Head: "release/a", Body: "<!-- rules_k8s_cd:release a -->\n\n## 1.0.1"}
	f.commit("docs: readme", map[string]string{"a/README.md": "r\n"})
	if err := PR(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	want := []string{`close #7 "No releasable changes for a anymore."`}
	if got := f.forge.writes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("forge writes = %v, want %v", got, want)
	}
	if f.remoteRef("refs/heads/release/a") != "" || f.remoteRef("refs/heads/release/b") != "" {
		t.Error("branch pushed without changes")
	}
}

// A PR on release/<name> without the marker is not the tool's: it is never
// updated or closed, and creating the release PR then fails loudly.
func TestPRLeavesForeignPRUntouched(t *testing.T) {
	f := setup(t)
	foreign := &forge.PR{Number: 9, Head: "release/a", Title: "mine", Body: "hand-made"}
	f.forge.prs["release/a"] = foreign
	f.commit("feat: a", map[string]string{"a/x.go": "x\n"})
	err := PR(f.env(), comps()[1:])
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v, want the CreatePR error", err)
	}
	if foreign.Title != "mine" || foreign.Body != "hand-made" || contains(f.forge.writes(), "update") {
		t.Errorf("foreign PR updated: %+v (writes %v)", foreign, f.forge.writes())
	}
	if !contains(f.warns, "PR #9 on release/a lacks the release marker") {
		t.Errorf("warns = %v", f.warns)
	}

	// Nothing to release: the foreign PR is not closed either.
	f2 := setup(t)
	f2.forge.prs["release/a"] = &forge.PR{Number: 9, Head: "release/a", Body: "hand-made"}
	if err := PR(f2.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	if w := f2.forge.writes(); len(w) != 0 || f2.forge.prs["release/a"] == nil {
		t.Errorf("foreign PR closed: writes %v", w)
	}
	if !contains(f2.warns, "PR #9 on release/a lacks the release marker") {
		t.Errorf("warns = %v", f2.warns)
	}
}

func TestPRBumpFileNoMatch(t *testing.T) {
	f := setup(t)
	f.commit("fix: a and b", map[string]string{"a/x.go": "x\n", "b/b.txt": "b2\n"})
	cs := comps()
	cs[1].BumpFiles = map[string]string{"a/version.go": `Nope = "(\d+\.\d+\.\d+)"`}
	err := PR(f.env(), cs)
	if err == nil || !strings.Contains(err.Error(), "a/version.go") {
		t.Fatalf("err = %v", err)
	}
	if f.remoteRef("refs/heads/release/a") != "" {
		t.Error("release/a pushed despite error")
	}
	if f.forge.prs["release/b"] == nil {
		t.Error("b blocked by a's error")
	}
}

func TestPRBumpFileNeedsOneGroup(t *testing.T) {
	f := setup(t)
	f.commit("fix: a", map[string]string{"a/x.go": "x\n"})
	cs := comps()[1:]
	cs[0].BumpFiles = map[string]string{"a/version.go": `Version = "\d+\.\d+\.\d+"`}
	if err := PR(f.env(), cs); err == nil || !strings.Contains(err.Error(), "capturing group") {
		t.Fatalf("err = %v", err)
	}
}

func TestPRRequiresMainTip(t *testing.T) {
	f := setup(t)
	f.commit("fix: a", map[string]string{"a/x.go": "x\n"})
	write(t, f.dir, "a/z.go", "z\n")
	git(t, f.dir, "add", "-A")
	git(t, f.dir, "commit", "-q", "-m", "fix: local only")
	if err := PR(f.env(), comps()); err == nil || !strings.Contains(err.Error(), "origin/main") {
		t.Fatalf("err = %v", err)
	}
	if len(f.forge.calls) != 0 {
		t.Errorf("forge called: %v", f.forge.calls)
	}
}

func TestPRSkipsWhenMainMoved(t *testing.T) {
	f := setup(t)
	f.commit("fix: a", map[string]string{"a/x.go": "x\n"})
	// Someone pushes to main after this checkout.
	other := t.TempDir()
	git(t, other, "clone", "-q", f.origin, ".")
	write(t, other, "a/z.go", "z\n")
	git(t, other, "add", "-A")
	git(t, other, "commit", "-q", "-m", "fix: later")
	git(t, other, "push", "-q", "origin", "HEAD:main")
	git(t, f.dir, "fetch", "-q", "origin")

	if err := PR(f.env(), comps()); err != nil {
		t.Fatalf("err = %v, want pr skipped", err)
	}
	if !contains(f.warns, "main moved since checkout; skipping pr (the next run will update PRs)") {
		t.Errorf("warns = %v", f.warns)
	}
	if len(f.forge.calls) != 0 || f.remoteRef("refs/heads/release/a") != "" {
		t.Errorf("pr ran on a stale checkout: calls %v", f.forge.calls)
	}
}

func TestPRDryRun(t *testing.T) {
	f := setup(t)
	f.forge.prs["release/b"] = &forge.PR{Number: 3, Head: "release/b", Body: "<!-- rules_k8s_cd:release b -->"}
	f.commit("feat: a", map[string]string{"a/x.go": "x\n"})
	f.dryRun = true
	for _, noForge := range []bool{false, true} {
		f.noForge = noForge
		f.out = nil
		if err := PR(f.env(), comps()); err != nil {
			t.Fatal(err)
		}
		if w := f.forge.writes(); len(w) != 0 {
			t.Errorf("forge writes in dry-run: %v", w)
		}
		if f.remoteRef("refs/heads/release/a") != "" {
			t.Error("pushed in dry-run")
		}
		if !contains(f.out, "chore(release): a 1.1.0") {
			t.Errorf("plan not printed: %v", f.out)
		}
	}
}

func TestPRSkipsComponentAwaitingTag(t *testing.T) {
	f := setup(t)
	f.commit("feat: a", map[string]string{"a/x.go": "x\n"})
	f.commit("chore(release): a 1.1.0 (#4)", map[string]string{"a/VERSION": "1.1.0\n"})
	f.commit("fix: after release", map[string]string{"a/y.go": "y\n"})
	if err := PR(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	if w := f.forge.writes(); len(w) != 0 {
		t.Errorf("forge writes while a 1.1.0 awaits its tag: %v", w)
	}
	plans, err := Status(f.env(), comps()[1:])
	if err != nil {
		t.Fatal(err)
	}
	p := plans[0]
	if !p.TagPending || len(p.Entries) != 1 || p.Entries[0].Commit.Subject != "fix: after release" || p.Next.String() != "1.1.1" {
		t.Errorf("plan = pending %v next %v entries %+v", p.TagPending, p.Next, p.Entries)
	}
	if contains(f.warns, "has no tag") {
		t.Errorf("spurious baseline warning: %v", f.warns)
	}
}

// --- Tag -------------------------------------------------------------------

// release simulates merging a's release PR (squash) on main.
func (f *fixture) release() string {
	f.commit("feat: a", map[string]string{"a/x.go": "x\n"})
	return f.commit("chore(release): a 1.1.0 (#4)", map[string]string{
		"a/VERSION":      "1.1.0\n",
		"a/CHANGELOG.md": "# Changelog\n\n## 1.1.0 (2026-10-24)\n\n### Features\n\n- a\n\n## 1.0.0 (2026-01-01)\n\n- old\n",
	})
}

func TestTagReleaseCommit(t *testing.T) {
	f := setup(t)
	rc := f.release()
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if got := f.remoteRef("refs/tags/a@v1.1.0^{commit}"); got != rc {
		t.Errorf("remote tag at %q, want %s", got, rc)
	}
	if got := git(t, f.dir, "rev-parse", "a@v1.1.0^{commit}"); got != rc {
		t.Errorf("local tag at %q", got)
	}
	want := []string{"release a@v1.1.0 a@v1.1.0"}
	if got := f.forge.writes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("forge writes = %v, want %v", got, want)
	}
	if got := f.forge.releases["a@v1.1.0"]; got != "### Features\n\n- a" {
		t.Errorf("notes = %q", got)
	}
	env := []string{"RELEASE_COMPONENT=a", "RELEASE_VERSION=1.1.0", "RELEASE_TAG=a@v1.1.0"}
	wantRuns := []targetRun{{"//a:push", env, f.dir, "1.1.0\n"}, {"//a:publish", env, f.dir, "1.1.0\n"}}
	if !reflect.DeepEqual(f.runs, wantRuns) {
		t.Fatalf("runs = %+v, want %+v", f.runs, wantRuns)
	}

	// Second run: nothing.
	f.runs = nil
	n := len(f.forge.writes())
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if len(f.forge.writes()) != n || len(f.runs) != 0 {
		t.Errorf("second run: writes %v runs %v", f.forge.writes(), f.runs)
	}

	// --retry-targets: targets only.
	f.retryTargets = true
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if len(f.forge.writes()) != n || !reflect.DeepEqual(f.runs, wantRuns) {
		t.Errorf("retry: writes %v runs %+v", f.forge.writes(), f.runs)
	}
}

func TestTagRetryAfterHeadMoved(t *testing.T) {
	f := setup(t)
	rc := f.release()
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	f.commit("fix: later", map[string]string{"b/b.txt": "b3\n"})
	f.runs = nil
	n := len(f.forge.writes())
	f.retryTargets = true
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if len(f.runs) != 2 || f.runs[0].Dir == f.dir || f.runs[0].Version != "1.1.0\n" {
		t.Errorf("retry runs = %+v", f.runs)
	}
	if len(f.forge.writes()) != n {
		t.Errorf("retry wrote to the forge: %v", f.forge.writes())
	}
	if got := f.remoteRef("refs/tags/a@v1.1.0^{commit}"); got != rc {
		t.Errorf("tag moved to %q", got)
	}
	// b's VERSION was not set by a release commit: no retry for it.
	for _, r := range f.runs {
		if strings.Contains(strings.Join(r.Env, " "), "RELEASE_COMPONENT=b") {
			t.Errorf("retried b: %+v", r)
		}
	}
}

// squashBranch merges release/a like a forge squash merge keeping the
// release commit's message: subject " (#4)" plus its body.
func (f *fixture) squashBranch() string {
	f.t.Helper()
	git(f.t, f.dir, "fetch", "-q", "origin")
	msg := git(f.t, f.dir, "log", "-1", "--format=%B", "origin/release/a")
	subject, body, _ := strings.Cut(msg, "\n")
	git(f.t, f.dir, "checkout", "origin/release/a", "--", "a")
	git(f.t, f.dir, "commit", "-q", "-m", subject+" (#4)"+"\n"+body)
	f.push()
	return git(f.t, f.dir, "rev-parse", "HEAD")
}

func TestTagWarnsCommitsMissingFromChangelog(t *testing.T) {
	f := setup(t)
	f.commit("feat: a", map[string]string{"a/x.go": "x\n"})
	if err := PR(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	late := f.commit("fix: late", map[string]string{"a/z.go": "z\n"})
	f.commit("chore(release): b 1.0.1", map[string]string{"b/VERSION": "1.0.1\n"})
	f.squashBranch()
	if err := Tag(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	want := "a 1.1.0: 1 commit(s) landed after the release PR was last updated and are not in its changelog: " + git(t, f.dir, "rev-parse", "--short", late)
	if !contains(f.warns, want) {
		t.Errorf("warns = %v, want %q", f.warns, want)
	}

	// Merged right after pr ran: nothing is missing.
	g := setup(t)
	g.commit("feat: a", map[string]string{"a/x.go": "x\n"})
	if err := PR(g.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	g.squashBranch()
	if err := Tag(g.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	if contains(g.warns, "landed after") {
		t.Errorf("spurious warning: %v", g.warns)
	}
	if g.remoteRef("refs/tags/a@v1.1.0") == "" {
		t.Error("not tagged")
	}
}

func TestTagHandEditedVersion(t *testing.T) {
	f := setup(t)
	f.commit("fix: bump by hand", map[string]string{"a/VERSION": "1.1.0\n"})
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if !contains(f.warns, "VERSION of a changed outside a release commit; not releasing") {
		t.Errorf("warns = %v", f.warns)
	}
	if f.remoteRef("refs/tags/a@v1.1.0") != "" || len(f.forge.writes()) != 0 || len(f.runs) != 0 {
		t.Error("released a hand-edited VERSION")
	}
}

func TestTagOtherComponentsVersion(t *testing.T) {
	f := setup(t)
	f.commit("chore(release): b 1.1.0", map[string]string{"a/VERSION": "1.1.0\n"})
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if f.remoteRef("refs/tags/a@v1.1.0") != "" {
		t.Error("tagged a from b's release commit")
	}
}

func TestTagWorktreeWhenHeadMoved(t *testing.T) {
	f := setup(t)
	rc := f.release()
	f.commit("docs: after", map[string]string{"b/b.txt": "b9\n"})
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if got := f.remoteRef("refs/tags/a@v1.1.0^{commit}"); got != rc {
		t.Fatalf("tag at %q, want %s", got, rc)
	}
	if len(f.runs) != 2 {
		t.Fatalf("runs = %+v", f.runs)
	}
	d := f.runs[0].Dir
	if d == f.dir || f.runs[1].Dir != d || f.runs[0].Version != "1.1.0\n" {
		t.Errorf("runs = %+v", f.runs)
	}
	if _, err := os.Stat(d); !os.IsNotExist(err) {
		t.Errorf("worktree %s not removed: %v", d, err)
	}
	if out := git(t, f.dir, "worktree", "list", "--porcelain"); strings.Count(out, "worktree ") != 1 {
		t.Errorf("worktree left registered:\n%s", out)
	}
}

func TestTagTargetFailureCleansWorktree(t *testing.T) {
	f := setup(t)
	f.release()
	f.commit("docs: after", map[string]string{"b/b.txt": "b9\n"})
	f.failTarget = "//a:push"
	err := Tag(f.env(), comps())
	if err == nil || !strings.Contains(err.Error(), "//a:push") {
		t.Fatalf("err = %v", err)
	}
	if len(f.runs) != 1 {
		t.Errorf("ran past the failing target: %+v", f.runs)
	}
	if !contains(f.warns, "a: a@v1.1.0 was tagged but target //a:push failed; rerun with --retry-targets") {
		t.Errorf("warns = %v", f.warns)
	}
	if _, err := os.Stat(f.runs[0].Dir); !os.IsNotExist(err) {
		t.Errorf("worktree not removed after failure")
	}
}

func TestTagPushesLocalOnlyTag(t *testing.T) {
	f := setup(t)
	rc := f.release()
	git(t, f.dir, "tag", "-a", "a@v1.1.0", "-m", "x", rc)
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if f.remoteRef("refs/tags/a@v1.1.0^{commit}") != rc || len(f.runs) != 2 {
		t.Errorf("local-only tag not published: runs %+v", f.runs)
	}
}

func TestTagExistingTagWithoutRelease(t *testing.T) {
	f := setup(t)
	f.release()
	// Tag pushed by an earlier run (or by hand) without a forge release.
	git(t, f.dir, "tag", "-a", "a@v1.1.0", "-m", "x")
	git(t, f.dir, "push", "-q", "origin", "a@v1.1.0")
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if got := f.forge.writes(); len(got) != 0 || len(f.runs) != 0 {
		t.Fatalf("plain rerun published: writes %v runs %+v", got, f.runs)
	}
	f.retryTargets = true
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if got := f.forge.writes(); !reflect.DeepEqual(got, []string{"release a@v1.1.0 a@v1.1.0"}) {
		t.Errorf("writes = %v", got)
	}
	if len(f.runs) != 2 {
		t.Errorf("targets not run on retry: %+v", f.runs)
	}
}

func TestTagReleaseFailureThenRetry(t *testing.T) {
	f := setup(t)
	rc := f.release()
	f.forge.failRelease = true
	err := Tag(f.env(), comps())
	if err == nil || !strings.Contains(err.Error(), "forge down") {
		t.Fatalf("err = %v", err)
	}
	if !contains(f.warns, "a: a@v1.1.0 was tagged but release failed; rerun with --retry-targets") {
		t.Errorf("warns = %v", f.warns)
	}
	if f.remoteRef("refs/tags/a@v1.1.0^{commit}") != rc || len(f.runs) != 0 {
		t.Fatalf("tag missing or targets ran: %+v", f.runs)
	}
	// A plain rerun does nothing; the retry finishes the release.
	f.forge.failRelease = false
	if err := Tag(f.env(), comps()); err != nil || len(f.forge.writes()) != 0 {
		t.Fatalf("plain rerun: err %v writes %v", err, f.forge.writes())
	}
	f.retryTargets = true
	if err := Tag(f.env(), comps()); err != nil {
		t.Fatal(err)
	}
	if got := f.forge.writes(); !reflect.DeepEqual(got, []string{"release a@v1.1.0 a@v1.1.0"}) || len(f.runs) != 2 {
		t.Errorf("retry: writes %v runs %+v", got, f.runs)
	}
	if n := strings.Count(strings.Join(f.out, "\n"), "tagged a@v1.1.0"); n != 1 {
		t.Errorf("retagged: %v", f.out)
	}
}

func TestStatusLocalOnlyTagIsPending(t *testing.T) {
	f := setup(t)
	rc := f.release()
	git(t, f.dir, "tag", "-a", "a@v1.1.0", "-m", "x", rc)
	f.commit("fix: after release", map[string]string{"a/y.go": "y\n"})
	plans, err := Status(f.env(), comps()[1:])
	if err != nil {
		t.Fatal(err)
	}
	if !plans[0].TagPending {
		t.Errorf("unpublished tag not pending: %+v", plans[0])
	}
	if err := PR(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	if w := f.forge.writes(); len(w) != 0 || f.remoteRef("refs/heads/release/a") != "" {
		t.Errorf("pr proposed over an unpublished tag: %v", w)
	}
}

// A clone without tags (or a CI checkout fetching none): the released
// version's tag is only on the remote.
func TestStatusRemoteOnlyTag(t *testing.T) {
	f := setup(t)
	rc := f.release()
	if err := Tag(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	f.commit("fix: next", map[string]string{"a/y.go": "y\n"})
	// After the commit: its fetch would bring the tag back.
	git(t, f.dir, "tag", "-d", "a@v1.1.0")
	f.warns = nil
	plans, err := Status(f.env(), comps()[1:])
	if err != nil {
		t.Fatal(err)
	}
	p := plans[0]
	if p.State.BaselineSHA != rc || p.TagPending || len(p.Entries) != 1 || p.Entries[0].Commit.Subject != "fix: next" {
		t.Errorf("plan: baseline %s (want %s) pending %v entries %+v", p.State.BaselineSHA, rc, p.TagPending, p.Entries)
	}
	if contains(f.warns, "has no tag") {
		t.Errorf("spurious baseline warning: %v", f.warns)
	}
	if err := PR(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	pr := f.forge.prs["release/a"]
	if pr == nil || pr.Title != "chore(release): a 1.1.1" || strings.Contains(pr.Body, "feat") {
		t.Errorf("PR re-proposes released commits: %+v", pr)
	}
}

func TestTagThenPR(t *testing.T) {
	f := setup(t)
	f.release()
	if err := Tag(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	if err := PR(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	if got := f.forge.writes(); !reflect.DeepEqual(got, []string{"release a@v1.1.0 a@v1.1.0"}) {
		t.Fatalf("PR right after the release proposed something: %v", got)
	}
	f.commit("fix: next", map[string]string{"a/y.go": "y\n"})
	if err := PR(f.env(), comps()[1:]); err != nil {
		t.Fatal(err)
	}
	pr := f.forge.prs["release/a"]
	if pr == nil || pr.Title != "chore(release): a 1.1.1" {
		t.Fatalf("pr = %+v", pr)
	}
	if strings.Contains(pr.Body, "feat") || strings.Contains(pr.Body, "- a (") || !strings.Contains(pr.Body, "next") {
		t.Errorf("body re-proposes released commits:\n%s", pr.Body)
	}
}

func TestTagDryRun(t *testing.T) {
	f := setup(t)
	f.release()
	f.dryRun = true
	for _, noForge := range []bool{false, true} {
		f.noForge = noForge
		f.out = nil
		if err := Tag(f.env(), comps()); err != nil {
			t.Fatal(err)
		}
		if f.remoteRef("refs/tags/a@v1.1.0") != "" || git(t, f.dir, "tag", "--list", "a@v1.1.0") != "" {
			t.Error("tagged in dry-run")
		}
		if len(f.forge.writes()) != 0 || len(f.runs) != 0 {
			t.Errorf("dry-run wrote: %v %v", f.forge.writes(), f.runs)
		}
		if !contains(f.out, "a@v1.1.0") || !contains(f.out, "//a:publish") {
			t.Errorf("plan not printed: %v", f.out)
		}
	}
}

func TestBazelRunTarget(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	bazel := filepath.Join(dir, "bazel")
	write(t, dir, "bazel", "#!/bin/sh\necho \"$PWD|$RELEASE_TAG|$*\" > "+log+"\n")
	if err := os.Chmod(bazel, 0o755); err != nil {
		t.Fatal(err)
	}
	ws := t.TempDir()
	run := BazelRunTarget(bazel)
	if err := run("//a:push", []string{"RELEASE_TAG=a@v1.1.0"}, ws); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(log)
	got := strings.TrimSpace(string(b))
	real, _ := filepath.EvalSymlinks(ws)
	if got != real+"|a@v1.1.0|run --stamp --embed_label=a@v1.1.0 //a:push" && got != ws+"|a@v1.1.0|run --stamp --embed_label=a@v1.1.0 //a:push" {
		t.Errorf("bazel invoked as %q", got)
	}
}

func TestTagRetryRefusesMisplacedTag(t *testing.T) {
	f := setup(t)
	rc := f.release()
	parent := git(t, f.dir, "rev-parse", rc+"^")
	git(t, f.dir, "tag", "-a", "a@v1.1.0", "-m", "x", parent)
	git(t, f.dir, "push", "-q", "origin", "a@v1.1.0")
	f.retryTargets = true
	err := Tag(f.env(), comps())
	if err == nil || !strings.Contains(err.Error(), "a: a@v1.1.0 points at "+parent+", not release commit "+rc+"; refusing to retry") {
		t.Fatalf("err = %v", err)
	}
	if len(f.runs) != 0 || len(f.forge.writes()) != 0 {
		t.Errorf("retried a misplaced tag: writes %v runs %+v", f.forge.writes(), f.runs)
	}
}
