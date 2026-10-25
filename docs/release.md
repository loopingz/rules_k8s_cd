# release

`release` is a Go CLI and a set of Starlark rules shipped with rules_k8s_cd. It replaces tools such as release-please for Bazel repositories: it keeps a release pull request up to date from the conventional commits that landed on the main branch, and when that pull request is merged it tags the release, publishes it on the forge and runs your release targets.

It works with GitHub and Forgejo (and Gitea), runs in any CI, and only needs `git` and `bazel` on the `PATH`. It needs Bazel 8 or later and builds on the dependency graph of [`change_report`](change_report.md): a component is released only for the commits that touch its own targets.

```bash
bazel run @rules_k8s_cd//go/release -- status
```

Inside the rules_k8s_cd repository itself use `//go/release` instead of `@rules_k8s_cd//go/release`.

## Concepts

- **Component**: a releasable unit of the repository, declared with `release_component`. A repository can have several (a library, a service, a CLI), each with its own version, tags, changelog and release PR.
- **VERSION file**: the version of a component, a single line `X.Y.Z` followed by a newline. It is only written by release commits. Only `MAJOR.MINOR.PATCH` is supported (no pre-release suffix).
- **Tag**: `tag_format` applied to the version, `{name}@v{version}` by default (for example `api@v1.4.0`), or `v{version}` for a single-component repository such as this one. The tag is the baseline: the commits since the tag of the current version are what the next release is computed from.
- **Release PR**: one pull request per component, from the branch `release/<name>` to the main branch, titled `chore(release): <name> <version>`. It contains one commit with the new VERSION, the changelog section and the bumped files. The tool force-pushes that branch, so never commit to it by hand. The PR body carries the marker `<!-- rules_k8s_cd:release <name> -->`: the tool only updates or closes an open PR on `release/<name>` whose body carries it. Another PR on that branch (opened by hand, or a fork's branch of the same name on Forgejo) is left untouched with a warning; if the forge then refuses to open the release PR next to it, the error is reported. The release commit and the PR body also end with a `Release-Base: <sha>` trailer, the main commit the PR was built on (see [Commits merged after the last update](#commits-merged-after-the-last-update)).
- **Release commit**: the squash-merged release PR, or with a merge commit the release PR's own commit (it keeps its subject). Its subject `chore(release): <name> <version>` (a forge ` (#N)` suffix is fine) is how the tool recognises it. Release commits are never attributed to any component and VERSION and changelog files never count as changes, so a release can never trigger another release.

### Flow

1. A push to the main branch runs `release run`. It first runs `tag`, then `pr`.
2. `tag` looks at the VERSION of every component at `HEAD`. If it was set by a release commit and its tag is not on the remote yet, the tool tags that commit, pushes the tag, creates the forge release (notes: the changelog section) and runs the component's `release_targets`.
3. `pr` computes, for every component, the commits since its tag that affect it, derives the next version, and opens or updates the release PR. A component with nothing to release gets its open release PR closed.
4. Merging the release PR (squash, or a merge commit) pushes to main, and step 1 cuts the release.

Every command is safe to re-run; a failed run is finished by running it again.

### Version bumps

The next version comes from the [conventional commit](https://www.conventionalcommits.org/) subjects of the attributed commits, with the highest bump winning.

| Commit | Version >= 1.0.0 | Version < 1.0.0 |
|--------|------------------|-----------------|
| breaking (`!` or a `BREAKING CHANGE` footer) | major | minor |
| `feat` | minor | patch |
| `fix`, `perf`, `revert` | patch | patch |
| any other type, or a non-conventional subject (for example `wip:` or `Update stuff`) | patch | patch |
| only `chore`, `docs`, `style`, `test`, `ci`, `build`, `refactor` | none | none |

Unknown conventional types bump patch like non-conventional subjects, so a release is never missed. When nothing releasable remains, no PR is kept open. A `Release-As: X.Y.Z` footer in a commit message forces that version, as long as it is greater than the computed one.

A release commit is detected by its subject `chore(release): <name> <version>`, so the subject must be kept as is when the PR is merged.

### First release

A component without a tag needs at least one releasable commit from the root of the history. The version is `initial_version` (`0.1.0` by default) and the changelog is `Initial release` followed by the 50 most recent commits and `... and N earlier commits`.

### Baseline and VERSION

The current version is the content of the VERSION file and the baseline is the commit of its tag. When the VERSION file is ahead of any tag (a merged release PR that is not tagged yet), `pr` leaves the component alone and `tag` cuts the tag. When VERSION is below the highest tag of the component, the tag wins and the tool prints a warning. When the tag of the VERSION is missing but older tags exist, the latest tag is the baseline, with a warning. When no baseline can be resolved (for example after a force-push), all the history is attributed: the tool prefers an unnecessary release over a missed one.

## Rules

Load them from `@rules_k8s_cd//lib:release.bzl`.

### `release_component`

```python
load("@rules_k8s_cd//lib:release.bzl", "release_component")

release_component(
    name = "api",
    targets = ["//services/api:image", "//lib/shared:go_default_library"],
    changelog = "CHANGELOG.md",
    version_file = "VERSION",
    bump_files = {"charts/api/Chart.yaml": "(?m)^version: (.+)$"},
    release_targets = ["//services/api:publish"],
)
```

It declares the component and a `<name>.version` target (a [`release_version`](#release_version)) in the same package. The component is called `name`, or `component` when set.

| Attribute | Default | Description |
|-----------|---------|-------------|
| `name` | (required) | Component name, used in tags, the release branch (`release/<name>`), the commit subject and the PR. |
| `targets` | `[]` | Targets whose changes release the component. A commit releases the component when it changes one of these targets or anything they depend on. Exactly one of `targets` and `targets_query` must be set. |
| `targets_query` | `""` | A Bazel query selecting the targets, for example `//services/api/...`. |
| `version_file` | `"VERSION"` | VERSION file, relative to the package. It may not exist yet (never-released component). |
| `changelog` | `"CHANGELOG.md"` | Changelog file, relative to the package. A new section is prepended on each release. |
| `tag_format` | `"{name}@v{version}"` | Tag format. It must contain `{version}` and a non-empty literal prefix. |
| `initial_version` | `"0.1.0"` | Version of the first release. |
| `untagged_version` | `"main-{commit}"` | Version of untagged builds (see [Build-time version](#build-time-version)). |
| `ignore` | `[]` | Paths (relative to the package) whose changes never release the component. |
| `bump_files` | `{}` | Files updated on release: path relative to the package (not to the workspace: `tauri/Cargo.toml`, not `app/tauri/Cargo.toml`) mapped to a Go regular expression with exactly one capturing group; the text of the group is replaced with the new version. A file that does not exist or a pattern that matches nothing is an error. |
| `release_targets` | `[]` | Targets run, in order, after the component is tagged (see below). |
| `component` | `name` | Component name, used in tags, the release branch, the commit subject and the PR, when it differs from the target name: set it when another target of the package already has the component's name (for example the binary it releases, `go_binary(name = "tool")`). The generated version target is still `<name>.version`. |
| `visibility` | | Visibility of the generated targets. |

Targets must exist and be in the query universe, and a component whose `targets_query` matches nothing is an error.

Release targets run with `bazel run --stamp --embed_label=<tag> <target>` and the environment variables `RELEASE_COMPONENT`, `RELEASE_VERSION` and `RELEASE_TAG`. Like any `bazel run`, they start in their runfiles tree, not in the checkout: use `$BUILD_WORKSPACE_DIRECTORY` to reach the sources (for example `cd "$BUILD_WORKSPACE_DIRECTORY"` before running `gh`, which needs a git checkout or `GH_REPO`). In a release worktree it points at the worktree, that is at the release commit. Release targets inherit the environment of the job, so they need their own credentials (a registry login, `GH_TOKEN` for `gh`, ...): `FORGE_TOKEN` is not enough for tools that read other variables. Use them to push images, publish packages or edit the release notes. When `HEAD` is not the release commit (a later commit already landed on main, or the PR was merged with a merge commit), the targets run in a git worktree of the release commit, so they always build the released sources. The worktree lives at a stable path per repository, `rules_k8s_cd-release-worktree-<hash>` under `--cache-dir` (the temp dir when the cache is disabled): its Bazel output base is reused across runs, and the tool runs `bazel shutdown` in it before removing the worktree, so no Bazel server outlives the run. Only one release may run per repository at a time (the workflow below uses a concurrency group). If a target fails, the tag stays and the tool prints a warning to re-run with `--retry-targets`.

`bump_files` are **not** ignored when computing changes. Only the release commit is. A hand edit of a bump file, for example a `bazel_dep` change in `MODULE.bazel` bumped by the module version regex, counts for the owning component.

### `release_version`

Resolves the version of a component at build time. `release_component` already creates `<name>.version`; declare your own only for another view of the same component.

```python
load("@rules_k8s_cd//lib:release.bzl", "release_version")

release_version(
    name = "version",
    component = "api",
    version_file = "//:VERSION",
)
```

| Attribute | Default | Description |
|-----------|---------|-------------|
| `component` | (required) | Component name, as in its tags. |
| `version_file` | | The VERSION file. May be omitted when it does not exist yet. |
| `version_path` | | Path reported for a missing VERSION file. |
| `tag_format` | `"{name}@v{version}"` | Same as the component. |
| `untagged_version` | `"main-{commit}"` | Same as the component. |

Outputs `<name>.txt` (the version and a newline, the default output) and `<name>.json` (`version`, `commit`, `commit_full`, `date`, `date_iso`, `branch`, `component`, `tagged`), and provides `ReleaseVersionInfo`.

### `release_version_file`

Renders a template with the version of a `release_version` target, to compile the version into a binary.

```python
load("@rules_k8s_cd//lib:release.bzl", "release_version_file")

release_version_file(
    name = "version_go",
    out = "version.go",
    substitutions = {"package": "main"},
    template = "@rules_k8s_cd//lib/release:version.go.tpl",
    version = ":api.version",
)
```

| Attribute | Description |
|-----------|-------------|
| `version` | A `release_version` target. |
| `template` | Template file. Ready-made templates for Go, TypeScript, Python and JSON are in `@rules_k8s_cd//lib/release` (`version.go.tpl`, `version.ts.tpl`, `version.py.tpl`, `version.json.tpl`). |
| `out` | Output file. |
| `substitutions` | Extra `{key}` substitutions overriding the version info. `{package}` defaults to `version`. |

Placeholders available in templates: `{version}`, `{commit}`, `{commit_full}`, `{date}`, `{date_iso}`, `{branch}`, `{component}`. Values are substituted as is, without escaping for the target language. Only `{branch}` can contain quotes or backslashes (a branch name chosen by a contributor); avoid it inside string literals, or use `{commit}` instead.

### Build-time version

The version of a build is resolved in this order:

1. `--embed_label` equals the component's tag for VERSION: the version is VERSION. The release tool always passes the exact tag to release targets.
2. The component's tag for VERSION is among the tags at `HEAD` (`STABLE_GIT_TAGS_AT_HEAD`): VERSION.
3. Otherwise the `untagged_version` template. Variables: `{commit}` (short), `{commit_full}`, `{branch}`, `{date}` (`YYYYMMDD` of the commit date) and `{component}`. A missing value is `unknown`.

A VERSION file is required (and must be valid) as soon as the embed label or a tag at `HEAD` matches the component's tag prefix: a broken release build is an error, never a silent untagged build. An owned `--embed_label` that differs from the tag of VERSION is an error, and an owned tag at `HEAD` for another version gives an untagged version with a warning. Unstamped builds (`--nostamp`) resolve every git value to `unknown`, for example `main-unknown`.

Builds must be stamped with `lib/stamp.sh` (see [Stamping in other repositories](#stamping-in-other-repositories)).

## Commands

```bash
bazel run //go/release -- <command> [flags]
```

| Command | Description |
|---------|-------------|
| `status` | Prints the release plan of every component and writes nothing. It never uses the forge API, but it needs access to the git remote: whether a tag is published is read with `git ls-remote`. |
| `pr` | Opens, updates or closes the release PR of every component. |
| `tag` | Tags and publishes the releases whose release PR is merged. |
| `run` | `tag`, then `pr`. `pr` is skipped when `tag` fails, so an unpublished VERSION is never overwritten. |

`status` prints one line per component:

```
component    | current | next  | state
rules_k8s_cd | 4.1.0   | 4.2.0 | release PR
```

`state` is `up to date`, `release PR`, `first release` or `awaiting tag`.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--forge` | detected | `github` or `forgejo` (`gitea` is an alias). Detected from `FORGEJO_ACTIONS` / `GITEA_ACTIONS`, `GITHUB_ACTIONS`, then a `github.com` remote. |
| `--forge-url` | | Web URL of the forge: required for Forgejo outside Actions, and for GitHub Enterprise (the API is `<url>/api/v3`). |
| `--repo` | from the remote URL | `owner/name`. |
| `--remote` | `origin` | Git remote to push to. |
| `--main-branch` | the remote's HEAD branch, else `main` | Branch the release PRs target. `pr` requires `HEAD` to be its tip; when the branch only moved on since the checkout, `pr` warns and is skipped (exit 0): the run triggered by the newer commits updates the PRs. |
| `--components` | `//...` | Bazel query scope searched for `release_component` targets. |
| `--dry-run` | false | Prints what would be done; pushes, tags and writes nothing. |
| `--retry-targets` | false | `tag`: finish the release of the current VERSION of each component: re-create the forge release when missing and re-run the release targets (see below). |
| `--global-file` | | Extra glob of files that affect every component. Repeatable. See [change_report](change_report.md#how-a-target-is-judged-changed). |
| `--ignore` | | Glob of files never attributed to any component. Repeatable. |
| `--cache-dir` | `$XDG_CACHE_HOME/rules_k8s_cd/release`, else the OS user cache dir | Directory of the dependency graph cache. |
| `--workspace` | `$BUILD_WORKSPACE_DIRECTORY`, else the current directory | Bazel workspace directory. |
| `--bazel` | `bazel` | Bazel binary. |

The exit code is 0 on success, 1 on a runtime error and 2 on invalid usage. Warnings and errors go to stderr. A failure on one component does not stop the others; the errors are joined.

### Tokens

The token is read from `FORGE_TOKEN`, then `GITHUB_TOKEN`, then `FORGEJO_TOKEN`. It is never printed. `--dry-run` needs no token: without one it does not query the forge and prints what it would do (with a token it reads the forge, for a more precise preview). `status` never uses the forge API (it still reads the git remote).

On GitHub the token needs `contents: write` and `pull-requests: write`. On Forgejo it needs repository write access and the permission to open pull requests and releases.

### Publishing and retries

- The forge release is created only for a tag created in this run. Tags that already exist when you adopt the tool do not get a forge release.
- `tag` first checks the remote, so it can be re-run on a fresh CI clone: a remote tag on the release commit is fetched and left alone, a tag anywhere else is an error. `status` and `pr` also trust the remote: in a clone without tags, the remote tag of VERSION is the baseline.
- `--retry-targets` finishes the current-VERSION release of every component regardless of `HEAD`: it creates the forge release if it is missing and re-runs the release targets, in a worktree of the release commit when `HEAD` moved. It refuses when the tag does not point at the release commit. Trigger it with the `retry_targets` input of the workflow below after a failed release target.
- A VERSION already set by a release commit but not tagged yet (tag cut failed, or a push race) is cut by the next `tag`.

### Commits merged after the last update

The release PR is rebuilt on every push to main, but a commit can land between the last `pr` run and the merge of the release PR (for example while the release run of that commit is still queued). It is shipped in the release, yet absent from its changelog. The release commit, and the PR body, end with the trailer `Release-Base: <sha>` naming the main commit the PR was built on. When `tag` cuts the release, it lists the commits between that base and the release commit's parent (merges and release commits excluded) and warns:

```
warning: api 1.4.0: 2 commit(s) landed after the release PR was last updated and are not in its changelog: 1a2b3c4 5d6e7f8
```

They are not attributed to the next release either: edit the forge release notes by hand if they matter. To avoid it, wait for the release PR to be refreshed (the release run of the last commit on main has finished) before merging it. When the squash message keeps neither the commit message nor the PR description, the trailer is lost and nothing is checked.

## Setup

### First release in a repository

1. Declare the components in `BUILD` files (see above).
2. Merge. `release run` opens the first release PR(s): the changelog is `Initial release` with the latest commits, the version is `initial_version`.
3. Merge the PR with a **squash** (the squash title must be kept as is) or a merge commit. The next `release run` creates the tag, the forge release and runs the release targets.

For a repository that already has tags: create the VERSION file with the version of the latest tag (the tag format must match your existing tags, for example `v{version}`) and commit it. The first PR is then computed from the commits since that tag.

### Squash title rule

Squash-merge the release PR and keep the title `chore(release): <name> <version>`. A forge ` (#N)` suffix is fine. A merge commit works too: the release PR's own commit keeps the subject and is the one tagged. Editing the title breaks release detection: no tag is cut (the tool warns), and an edit of a shared bump file is then attributed to the other components. Restore the original title before merging.

### GitHub Actions

```yaml
name: release
on:
  push:
    branches: [main]
  workflow_dispatch:
    inputs:
      retry_targets:
        description: Finish the release of the current VERSION after a partial failure
        type: boolean
        default: false
permissions:
  contents: write
  pull-requests: write
concurrency:
  group: release
  cancel-in-progress: false
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0   # the whole history and the tags are required
          fetch-tags: true
      - uses: actions/cache@v4
        with:
          path: ~/.cache/rules_k8s_cd/release
          key: release-${{ runner.os }}-${{ github.sha }}
          restore-keys: release-${{ runner.os }}-
      - run: >
          bazel run @rules_k8s_cd//go/release --
          run ${{ inputs.retry_targets && '--retry-targets' || '' }}
        env:
          FORGE_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          # Only for release targets running `gh`: they inherit this
          # environment and start in their runfiles tree, not the checkout.
          # GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          # GH_REPO: ${{ github.repository }}
```

Pull requests opened with `GITHUB_TOKEN` do not trigger other workflows, so CI does not run on the release PR (and a required check would block the merge). Use a personal access token or a GitHub App token with the same permissions in the `FORGE_TOKEN` secret instead.

If the repository has `release_component` fixtures (for example in test packages), scope the search with `--components '//... - //path/to/fixtures/...'`; this repository does so for `//lib/tests/...`.

This repository's own workflow is `.github/workflows/release.yml`.

### Forgejo Actions

```yaml
name: release
on:
  push:
    branches: [main]
  workflow_dispatch:
    inputs:
      retry_targets:
        description: Finish the release of the current VERSION after a partial failure
        type: boolean
        default: false
jobs:
  release:
    runs-on: docker   # the label of your runner
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - run: git fetch --tags --force
      - run: >
          bazel run @rules_k8s_cd//go/release --
          run --forge forgejo --forge-url https://forgejo.example.com
          ${{ inputs.retry_targets && '--retry-targets' || '' }}
        env:
          FORGE_TOKEN: ${{ secrets.FORGE_TOKEN }}
```

Create the `FORGE_TOKEN` secret from an access token of a user (or bot) with write access to the repository. The runner image needs `bazel` and `git`. The automatic token of Forgejo Actions can be used if it is allowed to push branches and open pull requests; as on GitHub, a PR opened with it may not trigger the CI workflows.

### Stamping in other repositories

The build-time version relies on these workspace status keys, written by `lib/stamp.sh` and used with `--workspace_status_command`:

```bash
#!/bin/bash
STABLE_GIT_COMMIT=$(git rev-parse --short HEAD)
STABLE_GIT_COMMIT_FULL=$(git rev-parse HEAD)
STABLE_GIT_COMMIT_DATE=$(git show -s --format=%cI HEAD)
STABLE_GIT_TAGS_AT_HEAD=$(git tag --points-at HEAD | tr '\n' ' ' | sed 's/ *$//')
STABLE_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

echo "STABLE_GIT_COMMIT $STABLE_GIT_COMMIT"
echo "STABLE_GIT_COMMIT_FULL $STABLE_GIT_COMMIT_FULL"
echo "STABLE_GIT_COMMIT_DATE $STABLE_GIT_COMMIT_DATE"
echo "STABLE_GIT_TAGS_AT_HEAD $STABLE_GIT_TAGS_AT_HEAD"
echo "STABLE_GIT_BRANCH $STABLE_GIT_BRANCH"
```

Copy the script into your repository and add it to your `.bazelrc`:

```
build --workspace_status_command=./stamp.sh
```

`lib/stamp.sh` of rules_k8s_cd also emits `STABLE_GIT_TAGS` and `STABLE_GIT_PR`, which this tool does not use.

## Migrating from release-please

This repository is the example. It used `google-github-actions/release-please-action` with the `simple` release type and the tags `v4.0.0` to `v4.1.0`.

1. Create `VERSION` with `4.1.0` (the latest tag, without the `v`).
2. Declare the component in the root `BUILD`:

   ```python
   release_component(
       name = "rules_k8s_cd",
       bump_files = {"MODULE.bazel": r'(?s)module\([^)]*?\bversion = "([^"]+)"'},
       changelog = "CHANGELOG.md",
       release_targets = ["//tools/release:release_notes_snippet"],
       tag_format = "v{version}",
       targets_query = "//...",
       version_file = "VERSION",
   )
   ```

   `tag_format = "v{version}"` keeps the existing tags valid. The regex matches the version inside `module(...)` only, not the `bazel_dep` versions. The `CHANGELOG.md` of release-please is kept: new sections are prepended in the same layout.
3. Port the release-notes script to a release target (`tools/release/release_notes_snippet.sh` reads `RELEASE_TAG`) and replace `release-please.yml` with `release.yml`.
4. Delete the release-please configuration (`release-please.yml`, `.release-please-manifest.json` and `release-please-config.json` if present).

The first release PR is computed from the commits since `v4.1.0`. The `MODULE.bazel` version, which release-please never updated, is set by that release.

## Known limitations

- Only `MAJOR.MINOR.PATCH` versions: no pre-release or build metadata in VERSION.
- Only the main branch is released: no release branches or backports. `pr` requires `HEAD` to be the tip of the main branch.
- The release PR must be squash-merged with its title kept, or merged with a merge commit (the release PR's commit keeps its subject).
- The forge release is created only for tags created by the tool, and a forge release deleted by hand is only re-created with `--retry-targets`.
- A release PR branch (`release/<name>`) is force-pushed: do not commit to it.
- The tool needs the full git history and the tags; shallow clones do not work.
- A deliberate downgrade of VERSION is not supported: the highest tag wins.
- Release targets that run in the release worktree use their own Bazel output base: the first run there is a cold build.
- Commits merged between the last `pr` run and the merge of the release PR are released but not in its changelog; `tag` warns about them.
- `status` and `pr` need access to the git remote (`git ls-remote`), even in dry-run.
- Changes outside the Bazel workspace and git-ignored files are not tracked (see [change_report](change_report.md#known-limitations)).
