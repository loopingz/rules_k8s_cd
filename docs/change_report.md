# change_report

`change_report` is a Go CLI shipped with rules_k8s_cd. It answers three questions for a GitOps pipeline:

1. Which Bazel targets (typically `gitops.<env>` targets) changed since a baseline commit?
2. Which commits caused each change? The output can be a Markdown changelog for the GitOps pull request.
3. Which gitops targets were removed, so their exported files can be deleted from the gitops directory?

It runs in any CI, including GitHub Actions and Forgejo Actions. It only needs `git` and `bazel` on the `PATH`.

```bash
bazel run @rules_k8s_cd//go/change_report -- \
  --since "$SINCE" \
  --targets 'filter("gitops\\.", //deployments/...)' \
  --format targets
```

Inside the rules_k8s_cd repository itself use `//go/change_report` instead of `@rules_k8s_cd//go/change_report`.

Under `bazel run`, relative paths given to `--since-file`, `--output`, `--cache-dir`, `--workspace` and `--gitops-dir` are resolved against the directory where you invoked `bazel run`.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--targets` | (required) | Bazel query expression of the universe of targets to report on. |
| `--since` | empty | Baseline commit used for all targets. When empty, every target is reported as changed ("no baseline"). |
| `--since-file` | | JSON file of per-target baselines: `{"//pkg:target": "<sha>"}`. Overrides `--since` for the targets it lists. |
| `--format` | `targets` | One of `targets`, `json`, `markdown`, `removed` (see below). |
| `--output` | stdout | Write the result to this file (atomically) instead of stdout. |
| `--gitops-dir` | | Directory scanned for exported gitops manifests. Repeatable. Enables removed-target detection. |
| `--global-file` | | Extra glob of files that affect every target. Repeatable. |
| `--commit-url` | derived from `origin` | Commit URL template containing `{sha}`, e.g. `https://git.example.com/org/repo/commit/{sha}`. |
| `--cache-dir` | `$XDG_CACHE_HOME/rules_k8s_cd/change_report`, else the OS user cache dir | Directory for the dependency graph cache. |
| `--workspace` | `$BUILD_WORKSPACE_DIRECTORY`, else the current directory | Bazel workspace directory. |
| `--bazel` | `bazel` | Bazel binary to invoke. |
| `--debug-cache` | false | Print the graph cache mode to stderr. |

Warnings and errors go to stderr; the report goes to stdout (or `--output`). The exit code is 0 on success, 1 on a runtime error and 2 on invalid usage.

## Output formats

### `targets`

The labels of changed targets, one per line, sorted. Targets without a baseline are included. Removed targets are not.

### `removed`

The files (relative to the repository root) that belong to removed gitops targets, sorted, one per line. Requires `--gitops-dir`. Feed it to `git rm`.

### `json`

```json
{
  "since": "aaaaaaa0000000000000000000000000000000aa",
  "head": "ddddddd00000000000000000000000000000ddd0",
  "baseline": true,
  "targets": [
    {
      "label": "//apps/api:deploy",
      "status": "changed",
      "since": "aaaaaaa0000000000000000000000000000000aa",
      "commits": [
        "2222222000000000000000000000000000000222",
        "1111111000000000000000000000000000000111"
      ],
      "files": ["apps/api/main.go", "apps/api/values.yaml"]
    },
    {
      "label": "//apps/old:deploy",
      "status": "removed",
      "since": "aaaaaaa0000000000000000000000000000000aa",
      "commits": [],
      "files": ["gitops/old/a.yaml", "gitops/old/b.yaml"]
    },
    {
      "label": "//apps/worker:deploy",
      "status": "unchanged",
      "since": "aaaaaaa0000000000000000000000000000000aa",
      "commits": [],
      "files": []
    }
  ],
  "unmanaged_commits": ["3333333000000000000000000000000000000333"],
  "commits": {
    "1111111000000000000000000000000000000111": {
      "short": "1111111",
      "author": "Alice",
      "email": "alice@example.com",
      "date": "2026-10-01T12:30:00Z",
      "subject": "feat: add things",
      "body": "",
      "url": "https://example.com/commit/1111111000000000000000000000000000000111"
    }
  }
}
```

(Abridged.) `status` is `changed`, `unchanged` or `removed`. For a changed target, `files` lists the changed files that explain the change. `unmanaged_commits` lists merge commits that could not be attributed.

### `markdown`

A changelog suitable for a pull request body. Commit subjects and bodies are HTML-escaped. Example with a baseline:

````markdown
# Changes aaaaaaa..ddddddd

2 target(s) changed.

## //apps/api:deploy

| Commit | Subject | Author |
|---|---|---|
| [2222222](https://example.com/commit/2222222000000000000000000000000000000222) | fix: handle a\|b &lt;nil&gt; | Bob |
| [1111111](https://example.com/commit/1111111000000000000000000000000000000111) | feat: add things | Alice |

<details><summary>Full commit messages</summary>

### 2222222 — fix: handle a|b &lt;nil&gt;

Longer explanation.

### 1111111 — feat: add things

</details>

## Removed

- //apps/old:deploy
  - gitops/old/a.yaml
  - gitops/old/b.yaml

## Not analysed

Merge commits are not attributed to targets:

- 3333333 Merge branch 'x' &lt;into&gt; main
````

When the baselines differ per target, each section starts with `_Since aaaaaaa_`. A target with no baseline gets `_No baseline: full deployment._` instead of a commit table, and without any baseline the title is `# Changes (no baseline: full deployment)`.

## Gitops header and removed targets

`kustomize_gitops` (and the `gitops` key of the `gitops` macro, see the README "GitOps Macro" section) writes this as the first line of every exported manifest:

```yaml
# rules_k8s_cd:target //deployments/app:gitops.prod
```

With `--gitops-dir gitops`, `change_report` scans the directory, groups files by that header and reports as `removed` every labelled target that no longer exists in the `--targets` universe at `HEAD`. Gitops files without the header are ignored, with a warning.

## How a target is judged changed

For each target, `change_report` computes the set of files and external repositories it depends on (via `bazel query --consistent_labels`), then looks at the files changed between the target's baseline and `HEAD`.

- Source files are matched against the target's dependency closure.
- Global files always affect every target: `MODULE.bazel` (see below), `WORKSPACE`, `WORKSPACE.bazel`, `WORKSPACE.bzlmod`, `.bazelrc`, `.bazelversion`, plus any `--global-file` glob.
- Dependency-looking files that cannot be mapped to a target mark everything as changed: `**/go.mod`, `**/go.sum`, `**/package.json`, `**/pnpm-lock.yaml`, `**/package-lock.json`, `**/yarn.lock`, `**/requirements*.txt`, `**/*.lock`, `**/Chart.yaml`, `**/Chart.lock`.
- External dependencies are attributed through `MODULE.bazel.lock`: each module extension records its input files (`recordedInputs`), which maps those files (for example `go.mod`) to the repositories the extension generates, and from there to the targets using them. A `MODULE.bazel` edit is only narrowed when it touches nothing but `bazel_dep` declarations; any other edit marks every target changed.
- The tool never under-reports: when something is uncertain (unknown baseline, unparsable optional input, unmapped file, doubtful cache) the result is "changed".

Commit `MODULE.bazel.lock` to get per-repository precision for external dependencies. Without it, only extension-level precision is available and `change_report` warns.

### Graph cache

Computing the dependency graph is the expensive part. `change_report` caches it in `--cache-dir`, keyed by the commit and the query, and reuses the cache of an earlier commit when the Bazel-relevant files are unchanged. Persist that directory between CI runs (see the workflow). `--debug-cache` prints which mode was used.

## Baselines

`--since` applies one baseline to everything. For independent environments, pass `--since-file` with the last deployed commit of each target. Merge commits between a baseline and `HEAD` are listed under "Not analysed". Unknown or unreachable baselines cause the affected target to be reported as changed, with a warning.

## Workflow example

The workflow below resolves the baseline, restores the cache, lists the changed gitops targets, regenerates their manifests, removes the files of deleted targets and opens a pull request with the changelog. It is valid on GitHub Actions and Forgejo Actions. Adapt the placeholders in `UPPER_CASE`.

Baseline resolution order:

1. the `since` workflow input;
2. the commit recorded by the latest gitops pull request (here, a `Baseline: <sha>` line that the workflow writes in the PR body);
3. the head SHA of the last successful run of this workflow.

If none is found, `SINCE` stays empty and every target is deployed.

```yaml
name: gitops
on:
  push:
    branches: [main]
  workflow_dispatch:
    inputs:
      since:
        description: Baseline commit (optional)
        required: false

jobs:
  gitops:
    runs-on: ubuntu-latest   # Forgejo: use a label your runner provides, e.g. docker
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0     # full history is required to attribute commits

      - uses: actions/cache@v4   # Forgejo: mirrored at https://code.forgejo.org/actions/cache
        with:
          path: ~/.cache/rules_k8s_cd/change_report
          key: ${{ runner.os }}-change-report-${{ github.sha }}
          restore-keys: ${{ runner.os }}-change-report-

      - name: Resolve baseline
        env:
          INPUT_SINCE: ${{ inputs.since }}
          TOKEN: ${{ secrets.GITHUB_TOKEN }}   # Forgejo: a token able to read PRs and runs
          API: ${{ github.api_url }}           # Forgejo: https://FORGE_HOST/api/v1
          REPO: ${{ github.repository }}
        run: |
          SINCE="$INPUT_SINCE"
          if [ -z "$SINCE" ]; then
            # Latest gitops PR (any state) labelled "gitops"; commit recorded in its body.
            # GitHub (gh):
            SINCE=$(GH_TOKEN="$TOKEN" gh pr list --label gitops --state all --limit 1 \
              --json body --jq '.[0].body' 2>/dev/null \
              | sed -n 's/^Baseline: \([0-9a-f]\{40\}\)$/\1/p' | head -n1 || true)
            # Forgejo (curl) alternative:
            # SINCE=$(curl -fsS -H "Authorization: token $TOKEN" \
            #   "$API/repos/$REPO/pulls?state=all&limit=1&labels=gitops" \
            #   | jq -r '.[0].body // ""' | sed -n 's/^Baseline: \([0-9a-f]\{40\}\)$/\1/p' | head -n1)
          fi
          if [ -z "$SINCE" ]; then
            # Last successful run of this workflow (GitHub; Forgejo: query $API/repos/$REPO/actions/runs).
            SINCE=$(curl -fsS -H "Authorization: Bearer $TOKEN" \
              "$API/repos/$REPO/actions/workflows/gitops.yml/runs?status=success&branch=main&per_page=1" \
              | jq -r '.workflow_runs[0].head_sha // ""' || true)
          fi
          echo "SINCE=$SINCE" >> "$GITHUB_ENV"

      - name: Detect changes
        run: |
          OUT="$RUNNER_TEMP/change_report"; mkdir -p "$OUT"; echo "OUT=$OUT" >> "$GITHUB_ENV"
          bazel run @rules_k8s_cd//go/change_report -- \
            --since "$SINCE" \
            --targets 'filter("gitops\\.", //deployments/...)' \
            --gitops-dir gitops \
            --format targets --output "$OUT/changed.txt"
          bazel run @rules_k8s_cd//go/change_report -- \
            --since "$SINCE" \
            --targets 'filter("gitops\\.", //deployments/...)' \
            --gitops-dir gitops \
            --format markdown --output "$OUT/changelog.md"
          bazel run @rules_k8s_cd//go/change_report -- \
            --since "$SINCE" \
            --targets 'filter("gitops\\.", //deployments/...)' \
            --gitops-dir gitops \
            --format removed --output "$OUT/removed.txt"

      - name: Regenerate gitops files
        run: |
          while read -r target; do
            [ -n "$target" ] && bazel run "$target"
          done < "$OUT/changed.txt"
          while read -r f; do
            [ -n "$f" ] && git rm -f -- "$f"
          done < "$OUT/removed.txt"
          { cat "$OUT/changelog.md"; echo; echo "Baseline: $(git rev-parse HEAD)"; } > "$OUT/pr-body.md"

      - name: Open pull request
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          if [ -z "$(git status --porcelain)" ]; then echo "nothing to do"; exit 0; fi
          git config user.name  "GITOPS_BOT_NAME"
          git config user.email "GITOPS_BOT_EMAIL"
          git checkout -b gitops/update
          git add -A gitops
          git commit -m "chore: update gitops manifests"
          git push -f origin gitops/update
          gh pr create --title "Update gitops manifests" --body-file "$OUT/pr-body.md" --label gitops \
            --head gitops/update   # Forgejo: use `tea pulls create` or POST $API/repos/$REPO/pulls
```

Notes:

- Write the outputs outside the workspace (`$RUNNER_TEMP` above) and run all `change_report` calls before generating files: any uncommitted or untracked file in the workspace disables the graph cache for that run.
- `github.api_url`, `GITHUB_ENV` and `actions/cache` are available in Forgejo Actions, but Forgejo's API lives at `https://FORGE_HOST/api/v1`, `gh` is GitHub-only (use `curl` or `tea`), and actions are resolved from your Forgejo instance (mirrors of common actions are on `code.forgejo.org`). The `runs-on` value must match a label registered on your runner.
- The cache `key` is unique per commit while `restore-keys` restores the newest earlier cache, which `change_report` then refreshes.

## Known limitations

- Changes outside the Bazel workspace directory are ignored, including `local_path_override` modules such as `path = "../x"`.
- Upgrading an unpinned Bazel binary does not invalidate the graph cache. Pin the version in `.bazelversion`.
- Git-ignored files (for example a user.bazelrc that is `try-import`ed) are not tracked.
- A dirty working tree disables the graph cache for that run.
- Merge commits are listed under "Not analysed" and not attributed to targets. Squash-merge (or rebase-merge) is recommended.
- Commit `MODULE.bazel.lock` for per-repository dependency precision.
- Gitops files without the `# rules_k8s_cd:target` header are ignored, with a warning.
