package process

import (
	"errors"
	"reflect"
	"runtime"
	"testing"

	"github.com/walkingice/process_manager/config"
)

type fakeShell struct {
	output   string
	err      error
	commands []string
}

func (shell *fakeShell) Run(command string) (string, error) {
	shell.commands = append(shell.commands, command)
	return shell.output, shell.err
}

type fakeInspector struct {
	processes []Process
	err       error
	called    bool
}

func (inspector *fakeInspector) Processes() ([]Process, error) {
	inspector.called = true
	return inspector.processes, inspector.err
}

type fakeProcess struct {
	pid         int32
	commandLine string
	err         error
}

func (process fakeProcess) PID() int32 { return process.pid }

func (process fakeProcess) CommandLine() (string, error) {
	return process.commandLine, process.err
}

func TestLookupCheckUsesFirstApplicableMethod(t *testing.T) {
	shell := &fakeShell{output: "42"}
	inspector := &fakeInspector{processes: []Process{fakeProcess{pid: 7, commandLine: "serve"}}}
	lookup := Lookup{Shell: shell, Inspector: inspector}
	configured := config.Process{
		ID: "pid-command", Find: "find-command", Pattern: "serve", Start: "serve",
	}

	status, err := lookup.Check(configured)

	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !status.Running || !reflect.DeepEqual(status.PIDs, []int32{42}) {
		t.Fatalf("Check() = %#v, want running PID 42", status)
	}
	if !reflect.DeepEqual(shell.commands, []string{"pid-command"}) {
		t.Errorf("shell commands = %v, want only id command", shell.commands)
	}
	if inspector.called {
		t.Error("Inspector.Processes() called, want false")
	}
}

func TestLookupCheckPrioritizesFindAndPattern(t *testing.T) {
	findShell := &fakeShell{}
	findInspector := &fakeInspector{processes: []Process{fakeProcess{pid: 7, commandLine: "serve"}}}
	findLookup := Lookup{Shell: findShell, Inspector: findInspector}
	_, err := findLookup.Check(config.Process{
		Find: "find-command", Pattern: "worker", Start: "serve",
	})
	if err != nil {
		t.Fatalf("find Check() error = %v", err)
	}
	if !reflect.DeepEqual(findShell.commands, []string{"find-command"}) || findInspector.called {
		t.Errorf("find lookup used shell = %v, inspector = %t", findShell.commands, findInspector.called)
	}

	patternInspector := &fakeInspector{processes: []Process{fakeProcess{pid: 8, commandLine: "worker"}}}
	patternLookup := Lookup{Inspector: patternInspector}
	status, err := patternLookup.Check(config.Process{Pattern: "worker", Start: "serve"})
	if err != nil {
		t.Fatalf("pattern Check() error = %v", err)
	}
	if !reflect.DeepEqual(status.PIDs, []int32{8}) {
		t.Errorf("pattern Check() = %#v, want PID 8", status)
	}
}

func TestLookupCheckIDPaths(t *testing.T) {
	for _, test := range []struct {
		name    string
		output  string
		err     error
		want    Status
		wantErr bool
	}{
		{"running", "12 34\n", nil, Status{Running: true, PIDs: []int32{12, 34}}, false},
		{"stopped", "", nil, Status{}, false},
		{"invalid PID", "pid", nil, Status{}, true},
		{"zero PID", "0", nil, Status{}, true},
		{"negative PID", "-1", nil, Status{}, true},
		{"out-of-range PID", "2147483648", nil, Status{}, true},
		{"shell error", "", errors.New("failed"), Status{}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			lookup := Lookup{Shell: &fakeShell{output: test.output, err: test.err}}
			status, err := lookup.Check(config.Process{ID: "pid-command", Start: "serve"})
			if (err != nil) != test.wantErr {
				t.Fatalf("Check() error = %v, want error = %t", err, test.wantErr)
			}
			if !reflect.DeepEqual(status, test.want) {
				t.Errorf("Check() = %#v, want %#v", status, test.want)
			}
		})
	}
}

func TestLookupCheckFindPaths(t *testing.T) {
	for _, test := range []struct {
		name     string
		shellErr error
		want     bool
	}{
		{"running", nil, true},
		{"stopped", errors.New("exit status 1"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			lookup := Lookup{Shell: &fakeShell{err: test.shellErr}}
			status, err := lookup.Check(config.Process{Find: "check", Stop: "stop", Start: "serve"})
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}
			if status.Running != test.want || len(status.PIDs) != 0 {
				t.Errorf("Check() = %#v, want running = %t without PIDs", status, test.want)
			}
		})
	}
}

func TestLookupCheckPatternAndStartPaths(t *testing.T) {
	processes := []Process{
		fakeProcess{pid: 10, commandLine: "worker --queue=fast"},
		fakeProcess{pid: 11, commandLine: "serve-web"},
	}
	lookup := Lookup{Inspector: &fakeInspector{processes: processes}}

	patternStatus, err := lookup.Check(config.Process{Pattern: `worker --queue=\w+`, Start: "serve-web"})
	if err != nil || !reflect.DeepEqual(patternStatus, Status{Running: true, PIDs: []int32{10}}) {
		t.Errorf("pattern Check() = %#v, %v", patternStatus, err)
	}

	startStatus, err := lookup.Check(config.Process{Start: "serve-web"})
	if err != nil || !reflect.DeepEqual(startStatus, Status{Running: true, PIDs: []int32{11}}) {
		t.Errorf("start Check() = %#v, %v", startStatus, err)
	}
}

func TestLookupCheckReturnsStoppedWhenNothingMatches(t *testing.T) {
	lookup := Lookup{Inspector: &fakeInspector{processes: []Process{
		fakeProcess{pid: 10, commandLine: "worker --queue=fast"},
	}}}

	for _, configured := range []config.Process{
		{Pattern: `web`, Start: "serve-web"},
		{Start: "serve-web"},
	} {
		status, err := lookup.Check(configured)
		if err != nil {
			t.Fatalf("Check() error = %v", err)
		}
		if status.Running || len(status.PIDs) != 0 {
			t.Errorf("Check() = %#v, want stopped", status)
		}
	}
}

func TestLookupCheckReportsProcessErrors(t *testing.T) {
	enumerationError := errors.New("not available")
	lookup := Lookup{Inspector: &fakeInspector{err: enumerationError}}
	if _, err := lookup.Check(config.Process{Pattern: "worker", Start: "serve"}); !errors.Is(err, enumerationError) {
		t.Fatalf("Check() error = %v, want %v", err, enumerationError)
	}

	commandLineError := errors.New("access denied")
	lookup = Lookup{Inspector: &fakeInspector{processes: []Process{fakeProcess{err: commandLineError}}}}
	if _, err := lookup.Check(config.Process{Start: "serve"}); !errors.Is(err, commandLineError) {
		t.Fatalf("Check() error = %v, want %v", err, commandLineError)
	}
}

func TestLookupCheckRejectsInvalidPattern(t *testing.T) {
	lookup := Lookup{Inspector: &fakeInspector{}}
	if _, err := lookup.Check(config.Process{Pattern: "[", Start: "serve"}); err == nil {
		t.Fatal("Check() error = nil, want invalid pattern error")
	}
}

func TestShellCommandUsesPlatformDefault(t *testing.T) {
	name, arguments := shellCommand("echo test")
	if runtime.GOOS == "windows" {
		if name != "cmd" || !reflect.DeepEqual(arguments, []string{"/C", "echo test"}) {
			t.Errorf("shellCommand() = %q, %v", name, arguments)
		}
		return
	}
	if name != "sh" || !reflect.DeepEqual(arguments, []string{"-c", "echo test"}) {
		t.Errorf("shellCommand() = %q, %v", name, arguments)
	}
}
