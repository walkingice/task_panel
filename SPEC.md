## Config

- By default, PM loads `~/.config/jchu/process_manager.toml`. The configuration
  file can be changed with the `-f` parameter.
- If the file does not exist, is invalid, or contains no process entries, PM
  displays an appropriate message and exits.
- The configuration format is defined in `process_manager.toml`.

## UI

- After loading the configuration file, PM lists all defined processes.
- Initially, each view item is disabled while PM asynchronously checks the
  process status. The item is enabled after its check completes.
- The process table reserves an icon column before status. A running process
  shows ✔ in that column and displays `running` in bold.
- Users can move the cursor with the Up and Down arrow keys or the Vim-style
  `j` and `k` keys.
- Pressing Enter starts or stops a process after a confirmation modal prevents
  accidental actions.
- In a confirmation modal, Enter or `y` confirms the action. Esc, `n`, or `q`
  cancels it without changing the process.
- A persistent panel on the right displays start, stop, error, and debugging
  messages.
- On the main screen, pressing `q` or Esc exits PM.

## Implementation

- PM uses `bubbletea` to build the TUI.
- PM uses `gopsutil` to inspect process command lines.

## Behavior

- PM determines whether a process is running by using the first applicable
  configured method in this order:
  1. `id`: run a command that prints one or more process PIDs.
  2. `find`: run a command that exits successfully only while the process is
     running.
  3. `pattern`: match a regular expression against process command lines.
  4. `start`: compare process command lines with the start command.
- Commands in `id`, `find`, `start`, and `stop` are executed by the system
  shell. Configuration commands must be appropriate for the target platform.
- A process configured with `find` must also configure `stop`; `find` does not
  provide a PID for PM's default stop behavior.
- A process started by PM must continue running after PM exits. PM does not
  restart processes.
- A process may be stopped by PM or may stop on its own.
