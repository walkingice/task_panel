// Package process isolates operating-system process integration.
package process

import gopsutilprocess "github.com/shirou/gopsutil/v4/process"

// Inspector enumerates operating-system processes.
type Inspector interface {
	Processes() ([]*gopsutilprocess.Process, error)
}

// SystemInspector enumerates processes through gopsutil.
type SystemInspector struct{}

// Processes returns all visible operating-system processes.
func (SystemInspector) Processes() ([]*gopsutilprocess.Process, error) {
	return gopsutilprocess.Processes()
}
