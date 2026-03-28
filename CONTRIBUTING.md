# Contributing to rules_k8s_cd

## Development Setup

### Prerequisites

- [Bazel](https://bazel.build/) (see `.bazelversion` for the required version)
- Go toolchain (for building Go-based tools like `tar_filter` and `kustomizer`)
- [Buildifier](https://github.com/bazelbuild/buildtools/tree/master/buildifier) for linting Starlark files

### Getting Started

```bash
# Build everything
bazel build //...

# Run all tests
bazel test //...
```

## Project Structure

```
lib/                          Starlark rules, macros, and toolchain definitions
  kubectl.bzl                 kubectl and kubectl_export rules
  kustomizer.bzl              kustomize macros and kustomization_injector rule
  grype.bzl                   grype_scan rule (built with scanner factory)
  trivy.bzl                   trivy_scan and trivy_sbom rules (built with scanner factory)
  kyverno.bzl                 kyverno_test rule
  dive.bzl                    dive rule
  utils.bzl                   run_all, show, write_source_file(s), tar_filter, download_binary
  oci.bzl                     OCI helpers and ContainerPushInfo provider
  private/scanner_factory.bzl Scanner rule factory (create_scanner_rule)
  private/toolchain_factory.bzl Toolchain factory (create_toolchain)
go/                           Go source for tar_filter and kustomizer
```

## Adding a New Toolchain

Use the toolchain factory in `lib/private/toolchain_factory.bzl`:

```python
load("//lib/private:toolchain_factory.bzl", "create_toolchain")

create_toolchain(
    name = "mytool",
    binary_name = "mytool",
)
```

This generates the toolchain type, info provider, and toolchain rule. Then register the toolchain in `MODULE.bazel`.

## Adding a New Scanner Rule

Use the scanner rule factory in `lib/private/scanner_factory.bzl`:

```python
load("//lib/private:scanner_factory.bzl", "create_scanner_rule")

my_scan = create_scanner_rule(
    toolchain_type = "@rules_k8s_cd//lib:mytool_toolchain_type",
    info_field = "mytoolinfo",
    build_commands = _my_scan_commands,
)
```

The `build_commands` function receives `(bin_short_path, ctx)` and returns the shell command string. See `lib/grype.bzl` or `lib/trivy.bzl` for examples.

## Testing

```bash
# Run all tests
bazel test //...

# Run a specific test
bazel test //go/tar_filter:tar_filter_test

# Run tests with verbose output
bazel test //... --test_output=all
```

## Linting

Format all Starlark files with buildifier:

```bash
buildifier -r .
```

## Release Process

Releases are managed via [release-please](https://github.com/googleapis/release-please). Merging to `main` with [Conventional Commits](https://www.conventionalcommits.org/) messages triggers automatic version bumps and changelog generation.

Commit prefixes:
- `feat:` — minor version bump
- `fix:` — patch version bump
- `chore:` / `docs:` / `ci:` — no version bump
