# Process Manager Development Plan

Each phase should leave the project buildable and have focused automated tests.
Later phases build on the public behavior defined in `SPEC.md`.

## Phase 0: Project Foundation

1. Create the Go module and application entry point.
2. Add the required dependencies: `bubbletea`, TOML parsing, and `gopsutil`.
3. Add a `Makefile` with format, test, and build targets.
4. Establish package boundaries for configuration, process integration, and UI.
5. Add a minimal application smoke test.

### Acceptance Criteria

- `make fmt`, `make test`, and `make build` succeed.
- The application can start and exit cleanly.

## Phase 1: Configuration Loading and Validation

1. Load the default configuration from `~/.conf/jchu/process_manager.toml`.
2. Support `-f` to select a different configuration file.
3. Parse `[[process]]` entries into explicit internal types.
4. Validate required `name` and `start` fields and the `find` plus `stop`
   constraint.
5. Display a clear message and exit for missing, invalid, or empty
   configuration files.

### Tests

- Default and overridden configuration paths.
- Valid TOML, malformed TOML, missing files, and no process entries.
- Validation of missing `name` or `start`, and `find` entries without `stop`.

### Acceptance Criteria

- PM either returns a validated process list or reports a useful error.

## Phase 2: Process Status Detection

1. Define a small process lookup interface so operating-system integration is
   testable.
2. Implement status checks using the first applicable method in this order:
   `id`, `find`, `pattern`, then `start`.
3. Run `id` and `find` through the system shell.
4. Match `pattern` and `start` against process command lines using
   `gopsutil`.
5. Preserve the PIDs needed by the default stop behavior.

### Tests

- Lookup-method precedence and each successful and unsuccessful lookup path.
- PID parsing, regular-expression matching, and command-line comparison.
- Error propagation from shell commands and process enumeration.

### Acceptance Criteria

- Given a configured process, PM accurately reports whether it is running and
  retains available matching PIDs.

## Phase 3: Diagnostic Command-Line Listing

1. Provide a development diagnostic command path that loads configuration and
   lists managed process names and states to standard output.
2. Keep presentation separate from configuration and lookup logic.

### Tests

- List output for running, stopped, and lookup-error processes.
- Exit behavior when configuration loading fails.

### Acceptance Criteria

- Developers can inspect configuration and status detection without starting
  the TUI. This diagnostic path is not a required end-user interface.

## Phase 4: Read-Only TUI

1. Build the main Bubble Tea view with a process list and persistent message
   panel.
2. Render process items as disabled until their asynchronous status check
   completes.
3. Render running processes with `✔` and stopped processes without it.
4. Support Up/Down and `j`/`k` navigation.
5. Exit from the main view with `q` or Esc.

### Tests

- Initial disabled state and state updates from asynchronous checks.
- Rendering of selection, running state, and messages.
- Keyboard navigation and exit commands.

### Acceptance Criteria

- The TUI lists every configured process, remains responsive during status
  checks, and supports the documented navigation and exit keys.

## Phase 5: Process Control and Confirmation

1. Open a confirmation modal when Enter is pressed on an enabled process.
2. Start stopped processes with their configured `start` command through the
   system shell.
3. Stop running processes with configured `stop`, or otherwise signal the
   matched PIDs.
4. Ensure a process started by PM continues after PM exits.
5. Refresh the selected process state after a control operation.
6. Define, document, and test the confirmation and cancellation keys.

### Tests

- Modal open, confirm, cancel, and close behavior.
- Start and configured-stop command execution.
- Default PID-based stop behavior and unavailable-PID errors.
- Status refresh and process-control errors.
- A replaceable process launcher verifies that started processes are detached
  from PM's lifecycle.

### Acceptance Criteria

- Enter never changes a process without confirmation.
- Successful and failed start/stop actions are reflected in the process list.
- A process launched by PM remains running after the UI exits.

## Phase 6: Messages, Reliability, and Release Verification

1. Send start, stop, error, and debugging messages to the persistent panel.
2. Handle processes that stop independently and failures during asynchronous
   checks without crashing the UI.
3. Review cross-platform process lookup behavior and shell-command boundaries.
4. Update user documentation with configuration and keyboard usage.
5. Run formatting, unit tests, build, and a manual end-to-end smoke test.

### Tests

- Message-panel updates for success, failure, and diagnostic events.
- UI recovery from lookup and command failures.
- End-to-end coverage using fakes for shell and process dependencies.

### Acceptance Criteria

- All behavior in `SPEC.md` is implemented, tested where practical, and
  documented.
