# Task Panel

Task Panel (TP) is a terminal UI for managing a configured subset of
processes. It shows each process's running state and lets users start or stop
them quickly.

## Project metadata

- Module: `task_panel`
- Version: defined by `applicationVersion` in `task_panel.go`
- Go: `1.26.0`

Go modules use [`go.mod`](go.mod) for the module path, Go version, and
dependencies. They do not define standard fields for a project description or
release version. Releases should use a Git tag matching `applicationVersion`.
Run `tp --help` or `tp help` to show usage. Run `tp -v` to print the current
version.

## Release

Pushing a new Git tag to GitHub starts the release workflow. It runs the test
suite, cross-compiles `tp` for Linux and macOS on `amd64` and `arm64`, and
uploads one `tar.gz` archive per platform plus `checksums.txt` to the matching
GitHub Release. Each archive contains only the `tp` binary.

Before creating a release, update `applicationVersion` to the intended version
and commit the change. Then create and push an annotated tag with the same
version:

```sh
git tag -a v0.1.0 -m "Release 0.1.0"
git push origin v0.1.0
```

To verify the GoReleaser configuration locally without publishing a release,
install GoReleaser and run:

```sh
goreleaser release --snapshot --clean
```

## Development

```sh
make fmt
make test
make build
```
