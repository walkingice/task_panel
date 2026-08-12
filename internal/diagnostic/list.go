// Package diagnostic formats development status listings.
package diagnostic

import (
	"fmt"
	"io"

	"process_manager/internal/config"
	"process_manager/internal/process"
)

// Lookup checks the running status of a configured process.
type Lookup interface {
	Check(config.Process) (process.Status, error)
}

// List writes the configured process names and their current states.
func List(writer io.Writer, processes []config.Process, lookup Lookup) error {
	for _, configured := range processes {
		state := statusText(configured, lookup)
		if _, err := fmt.Fprintf(writer, "%s: %s\n", configured.Name, state); err != nil {
			return fmt.Errorf("write diagnostic list: %w", err)
		}
	}
	return nil
}

func statusText(configured config.Process, lookup Lookup) string {
	status, err := lookup.Check(configured)
	if err != nil {
		return "error: " + err.Error()
	}
	if status.Running {
		return "running"
	}
	return "stopped"
}
