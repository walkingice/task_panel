// Package process isolates operating-system process integration.
package process

import gopsutilprocess "github.com/shirou/gopsutil/v4/process"

// Process describes the process data needed for status checks.
type Process interface {
	PID() int32
	CommandLine() (string, error)
}

// Inspector enumerates operating-system processes.
type Inspector interface {
	Processes() ([]Process, error)
}

// SystemInspector enumerates processes through gopsutil.
type SystemInspector struct{}

// Processes returns all visible operating-system processes.
func (SystemInspector) Processes() ([]Process, error) {
	processes, err := gopsutilprocess.Processes()
	if err != nil {
		return nil, err
	}

	inspected := make([]Process, len(processes))
	for index, process := range processes {
		inspected[index] = systemProcess{process: process}
	}
	return inspected, nil
}

type systemProcess struct {
	process *gopsutilprocess.Process
}

func (process systemProcess) PID() int32 {
	return process.process.Pid
}

func (process systemProcess) CommandLine() (string, error) {
	return process.process.Cmdline()
}
