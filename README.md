# Task Panel

Task Panel (TP) is a terminal UI for managing a configured subset of
processes. It shows each process's running state and lets users start or stop
them quickly.

## Project metadata

- Module: `task_panel`
- Version: defined by `applicationVersion` in `job_runner.go`
- Go: `1.26.0`

Go modules use [`go.mod`](go.mod) for the module path, Go version, and
dependencies. They do not define standard fields for a project description or
release version. Releases should use a Git tag matching `applicationVersion`.
Run `tp -v` to print the current version.

## Development

```sh
make fmt
make test
make build
```
