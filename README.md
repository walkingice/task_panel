# Process Manager

Process Manager (PM) is a terminal UI for managing a configured subset of
processes. It shows each process's running state and lets users start or stop
them quickly.

## Project metadata

- Module: `task_panel`
- Version: `0.1.0` (also recorded in [`VERSION`](VERSION))
- Go: `1.26.0`

Go modules use [`go.mod`](go.mod) for the module path, Go version, and
dependencies. They do not define standard fields for a project description or
release version. Releases should use matching Git tags, such as `v0.1.0`.

## Development

```sh
make fmt
make test
make build
```
