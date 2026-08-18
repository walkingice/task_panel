## Config

- By default, TP loads `~/.config/task_panel/config.toml`. The configuration
  file can be changed with the `-f` parameter.
- If the file does not exist, is invalid, or contains no process entries, TP
  displays an appropriate message and exits.
- The configuration format is defined in `config.example.toml`.
- `start_status_timeout_seconds` sets the post-start status retry timeout. It
  defaults to five seconds and must be a positive whole number.

## UI

- After loading the configuration file, TP lists all defined processes.
- Initially, each view item is disabled while TP asynchronously checks the
  process status. The item is enabled after its check completes.
- The process table reserves an icon column before status. A running process
  shows ✔ in that column and displays `running` in bold.
- Users can move the cursor with the Up and Down arrow keys or the Vim-style
  `j` and `k` keys.
- Pressing Enter starts or stops a process after a confirmation modal prevents
  accidental actions. The `show_start_confirmation` and
  `show_stop_confirmation` configuration options can independently disable
  that modal; both default to true.
- In a confirmation modal, Enter or `y` confirms the action. Esc, `n`, or `q`
  cancels it without changing the process.
- A persistent panel on the right displays start, stop, error, and debugging
  messages.
- On the main screen, pressing `q` or Esc exits TP.

## Implementation

- Version: defined by `applicationVersion` in `task_panel.go`
- TP uses `bubbletea` to build the TUI.
- TP uses `gopsutil` to inspect process command lines.

## Behavior

- TP determines whether a process is running by using the first applicable
  configured method in this order:
  1. `id`: run a command that prints one or more process PIDs.
  2. `find`: run a command that exits successfully only while the process is
     running.
  3. `pattern`: match a regular expression against process command lines.
  4. `start`: compare process command lines with the start command.
- Commands in `id`, `find`, `start`, and `stop` are executed by the system
  shell. Configuration commands must be appropriate for the target platform.
- A process configured with `find` must also configure `stop`; `find` does not
  provide a PID for TP's default stop behavior.
- A process started by TP must continue running after TP exits. TP does not
  restart processes.
- After a successful start request, TP checks the process status immediately
  and retries every 500 milliseconds for the configured timeout. It marks the
  process running as soon as a check succeeds; otherwise it reports a start
  timeout. Lookup errors are reported without retrying.
- A process may be stopped by TP or may stop on its own.
