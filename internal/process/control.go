package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"task_panel/internal/config"
)

// Launcher starts configured commands without waiting for them to finish.
type Launcher interface {
	Launch(string) error
}

// Signaler stops a process by PID.
type Signaler interface {
	Stop(int32) error
}

// SystemLauncher starts commands through the platform's default command shell.
type SystemLauncher struct{}

// Launch starts command and releases PM's reference to the child process.
func (SystemLauncher) Launch(command string) error {
	return launchCommand(systemCommandStarter{}, command)
}

type releasedProcess interface {
	Release() error
}

type commandStarter interface {
	Start(string, []string) (releasedProcess, error)
}

type systemCommandStarter struct{}

func (systemCommandStarter) Start(name string, arguments []string) (releasedProcess, error) {
	child := exec.Command(name, arguments...)
	if err := child.Start(); err != nil {
		return nil, err
	}
	return child.Process, nil
}

func launchCommand(starter commandStarter, command string) error {
	name, arguments := shellCommand(command)
	child, err := starter.Start(name, arguments)
	if err != nil {
		return err
	}
	return child.Release()
}

// SystemSignaler stops operating-system processes.
type SystemSignaler struct{}

// Stop forcefully stops the process identified by pid.
func (SystemSignaler) Stop(pid int32) error {
	child, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}
	return child.Kill()
}

// Controller starts and stops configured processes.
type Controller struct {
	Shell    Shell
	Launcher Launcher
	Signaler Signaler
}

// Start launches a configured process without waiting for it to exit.
func (controller Controller) Start(configured config.Process) error {
	if controller.Launcher == nil {
		return errors.New("start process: launcher is unavailable")
	}
	if err := controller.Launcher.Launch(configured.Start); err != nil {
		return fmt.Errorf("start process %q: %w", configured.Name, err)
	}
	return nil
}

// Stop uses a configured stop command or stops every matching PID.
func (controller Controller) Stop(configured config.Process, status Status) error {
	if configured.Stop != "" {
		if controller.Shell == nil {
			return errors.New("stop process: shell is unavailable")
		}
		if _, err := controller.Shell.Run(configured.Stop); err != nil {
			return fmt.Errorf("stop process %q: %w", configured.Name, err)
		}
		return nil
	}
	if len(status.PIDs) == 0 {
		return fmt.Errorf("stop process %q: no matching process IDs", configured.Name)
	}
	if controller.Signaler == nil {
		return errors.New("stop process: signaler is unavailable")
	}
	for _, pid := range status.PIDs {
		if err := controller.Signaler.Stop(pid); err != nil {
			return fmt.Errorf("stop process %q PID %d: %w", configured.Name, pid, err)
		}
	}
	return nil
}
