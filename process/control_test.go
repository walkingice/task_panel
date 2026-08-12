package process

import (
	"errors"
	"reflect"
	"runtime"
	"testing"

	"github.com/walkingice/process_manager/config"
)

type fakeLauncher struct {
	commands []string
	err      error
}

func (launcher *fakeLauncher) Launch(command string) error {
	launcher.commands = append(launcher.commands, command)
	return launcher.err
}

type fakeSignaler struct {
	pids []int32
	err  error
}

type fakeReleasedProcess struct {
	released bool
	err      error
}

func (process *fakeReleasedProcess) Release() error {
	process.released = true
	return process.err
}

type fakeCommandStarter struct {
	name      string
	arguments []string
	process   releasedProcess
	err       error
}

func (starter *fakeCommandStarter) Start(name string, arguments []string) (releasedProcess, error) {
	starter.name = name
	starter.arguments = arguments
	return starter.process, starter.err
}

func TestLaunchCommandReleasesChildProcess(t *testing.T) {
	child := &fakeReleasedProcess{}
	starter := &fakeCommandStarter{process: child}

	if err := launchCommand(starter, "serve-web"); err != nil {
		t.Fatalf("launchCommand() error = %v", err)
	}
	if !child.released {
		t.Fatal("launchCommand() did not release child process")
	}
	if runtime.GOOS == "windows" {
		if starter.name != "cmd" || !reflect.DeepEqual(starter.arguments, []string{"/C", "serve-web"}) {
			t.Errorf("command = %q %v, want Windows shell", starter.name, starter.arguments)
		}
		return
	}
	if starter.name != "sh" || !reflect.DeepEqual(starter.arguments, []string{"-c", "serve-web"}) {
		t.Errorf("command = %q %v, want POSIX shell", starter.name, starter.arguments)
	}
}

func (signaler *fakeSignaler) Stop(pid int32) error {
	signaler.pids = append(signaler.pids, pid)
	return signaler.err
}

func TestControllerStartsWithReplaceableLauncher(t *testing.T) {
	launcher := &fakeLauncher{}
	controller := Controller{Launcher: launcher}

	if err := controller.Start(config.Process{Name: "web", Start: "serve-web"}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got, want := launcher.commands, []string{"serve-web"}; !reflect.DeepEqual(got, want) {
		t.Errorf("launcher commands = %v, want %v", got, want)
	}
}

func TestControllerReportsStartErrors(t *testing.T) {
	want := errors.New("cannot start")
	controller := Controller{Launcher: &fakeLauncher{err: want}}
	if err := controller.Start(config.Process{Name: "web", Start: "serve-web"}); !errors.Is(err, want) {
		t.Fatalf("Start() error = %v, want %v", err, want)
	}
}

func TestControllerStopsWithConfiguredCommand(t *testing.T) {
	shell := &fakeShell{}
	signaler := &fakeSignaler{}
	controller := Controller{Shell: shell, Signaler: signaler}

	err := controller.Stop(config.Process{Name: "web", Stop: "stop-web"}, Status{PIDs: []int32{12}})

	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if got, want := shell.commands, []string{"stop-web"}; !reflect.DeepEqual(got, want) {
		t.Errorf("shell commands = %v, want %v", got, want)
	}
	if len(signaler.pids) != 0 {
		t.Errorf("signaler PIDs = %v, want none", signaler.pids)
	}
}

func TestControllerStopsEveryMatchedPID(t *testing.T) {
	signaler := &fakeSignaler{}
	controller := Controller{Signaler: signaler}

	err := controller.Stop(config.Process{Name: "web"}, Status{PIDs: []int32{12, 34}})

	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if got, want := signaler.pids, []int32{12, 34}; !reflect.DeepEqual(got, want) {
		t.Errorf("signaler PIDs = %v, want %v", got, want)
	}
}

func TestControllerReportsStopErrors(t *testing.T) {
	want := errors.New("cannot stop")
	controller := Controller{Shell: &fakeShell{err: want}}
	if err := controller.Stop(config.Process{Name: "web", Stop: "stop-web"}, Status{}); !errors.Is(err, want) {
		t.Fatalf("Stop() error = %v, want %v", err, want)
	}

	err := Controller{}.Stop(config.Process{Name: "web"}, Status{})
	if err == nil {
		t.Fatal("Stop() error = nil, want unavailable PID error")
	}

	signaler := &fakeSignaler{err: want}
	controller = Controller{Signaler: signaler}
	err = controller.Stop(config.Process{Name: "web"}, Status{PIDs: []int32{12, 34}})
	if !errors.Is(err, want) {
		t.Fatalf("Stop() error = %v, want %v", err, want)
	}
	if got, want := signaler.pids, []int32{12}; !reflect.DeepEqual(got, want) {
		t.Errorf("signaler PIDs = %v, want %v", got, want)
	}
}
