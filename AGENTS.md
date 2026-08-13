# Process Manager Agent Guide

## Project

Process Manager, or PM for short.

PM is a Go terminal UI that manages a subset of processes.

PM loads a configuration file that determines which processes are managed. The
interface displays a list of processes and their running state. Users can
quickly start or stop processes through the interface.

## Layout

- `task_panel.go`: application entry point.

Keep UI behavior in the relevant view package. Keep input and operating-system
integration isolated behind their existing package boundaries.

## Development

```bash
make fmt
make test
make build
```

Add or update focused `*_test.go` tests for behavior changes.

The product specification is defined in [SPEC.md](SPEC.md).

## Conventions

- Use standard Go formatting (`go fmt ./...`).
- Keep functions small and make dependencies explicit.
- Preserve cross-platform process lookup.
- Do not introduce dependencies unless they are necessary for the change.
- Keep documentation and code comments in English.
