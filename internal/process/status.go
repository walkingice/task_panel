package process

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"process_manager/internal/config"
)

// Shell runs configured commands through the operating system shell.
type Shell interface {
	Run(command string) (string, error)
}

// SystemShell runs commands through the platform's default command shell.
type SystemShell struct{}

// Run executes command and returns its combined standard output and error.
func (SystemShell) Run(command string) (string, error) {
	name, arguments := shellCommand(command)
	output, err := exec.Command(name, arguments...).CombinedOutput()
	return string(output), err
}

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "sh", []string{"-c", command}
}

// Status is the current running state and any PIDs available for stopping it.
type Status struct {
	Running bool
	PIDs    []int32
}

// Lookup checks configured process status using shell and process inspection.
type Lookup struct {
	Shell     Shell
	Inspector Inspector
}

// Check determines the process status using the first configured lookup method.
func (lookup Lookup) Check(process config.Process) (Status, error) {
	switch {
	case strings.TrimSpace(process.ID) != "":
		return lookup.checkID(process.ID)
	case strings.TrimSpace(process.Find) != "":
		return lookup.checkFind(process.Find)
	case strings.TrimSpace(process.Pattern) != "":
		return lookup.checkPattern(process.Pattern)
	default:
		return lookup.checkStart(process.Start)
	}
}

func (lookup Lookup) checkID(command string) (Status, error) {
	output, err := lookup.Shell.Run(command)
	if err != nil {
		return Status{}, fmt.Errorf("run id command: %w", err)
	}
	pids, err := parsePIDs(output)
	if err != nil {
		return Status{}, err
	}
	return Status{Running: len(pids) > 0, PIDs: pids}, nil
}

func (lookup Lookup) checkFind(command string) (Status, error) {
	_, err := lookup.Shell.Run(command)
	return Status{Running: err == nil}, nil
}

func (lookup Lookup) checkPattern(pattern string) (Status, error) {
	matcher, err := regexp.Compile(pattern)
	if err != nil {
		return Status{}, fmt.Errorf("compile pattern: %w", err)
	}
	return lookup.checkCommandLine(func(commandLine string) bool {
		return matcher.MatchString(commandLine)
	})
}

func (lookup Lookup) checkStart(start string) (Status, error) {
	return lookup.checkCommandLine(func(commandLine string) bool {
		return commandLine == start
	})
}

func (lookup Lookup) checkCommandLine(matches func(string) bool) (Status, error) {
	processes, err := lookup.Inspector.Processes()
	if err != nil {
		return Status{}, fmt.Errorf("enumerate processes: %w", err)
	}

	var pids []int32
	for _, process := range processes {
		commandLine, err := process.CommandLine()
		if err != nil {
			return Status{}, fmt.Errorf("read process command line: %w", err)
		}
		if matches(commandLine) {
			pids = append(pids, process.PID())
		}
	}
	return Status{Running: len(pids) > 0, PIDs: pids}, nil
}

func parsePIDs(output string) ([]int32, error) {
	fields := strings.Fields(output)
	if len(fields) == 0 {
		return nil, nil
	}
	pids := make([]int32, 0, len(fields))
	for _, field := range fields {
		pid, err := strconv.ParseInt(field, 10, 32)
		if err != nil || pid <= 0 {
			return nil, fmt.Errorf("parse process ID %q", field)
		}
		pids = append(pids, int32(pid))
	}
	return pids, nil
}
