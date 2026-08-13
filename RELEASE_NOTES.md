# Task Panel 0.3.0

## What's new

- Add the `start_status_timeout_seconds` configuration option to control how
  long Task Panel waits for a process to report as running after a start.
  The default is five seconds.
- Retry process-status checks every 500 milliseconds after a start request,
  so processes that take time to initialize can be identified as running.
- Report a start timeout when the configured wait period expires without a
  successful status check.

## Downloads

Release archives are available for Linux and macOS on `amd64` and `arm64`.
Verify downloaded archives with the included `checksums.txt` file.
