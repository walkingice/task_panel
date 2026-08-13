package main

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"task_panel/internal/config"
	"task_panel/internal/diagnostic"
	"task_panel/internal/process"
	"task_panel/internal/ui"
)

type fakeApplication struct {
	err error
	ran bool
}

func (app *fakeApplication) Run() (tea.Model, error) {
	app.ran = true
	return nil, app.err
}

type recordingApplicationFactory struct {
	application application
	processes   []config.Process
	lookup      diagnostic.Lookup
	calls       int
}

func (factory *recordingApplicationFactory) New(processes []config.Process, lookup diagnostic.Lookup) application {
	factory.calls++
	factory.processes = processes
	factory.lookup = lookup
	return factory.application
}

func TestRunStartsAndExitsApplication(t *testing.T) {
	program := tea.NewProgram(
		ui.New(nil, process.Lookup{}, process.Controller{}),
		tea.WithInput(bytes.NewBufferString("q")),
		tea.WithOutput(io.Discard),
	)

	if err := run(program); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
}

func TestRunReturnsApplicationError(t *testing.T) {
	want := errors.New("application failed")

	if err := run(&fakeApplication{err: want}); !errors.Is(err, want) {
		t.Fatalf("run() error = %v, want %v", err, want)
	}
}

func TestNewApplicationUsesAlternateScreen(t *testing.T) {
	var output bytes.Buffer
	newProgram := func(model tea.Model, options ...tea.ProgramOption) *tea.Program {
		options = append(options,
			tea.WithInput(bytes.NewBufferString("q")),
			tea.WithOutput(&output),
		)
		return tea.NewProgram(model, options...)
	}

	application := newApplicationWithProgram(nil, process.Lookup{}, newProgram)
	if err := run(application); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
	if !bytes.Contains(output.Bytes(), []byte("\x1b[?1049h")) {
		t.Errorf("output = %q, want alternate-screen escape sequence", output.String())
	}
}

type testFileReader struct {
	data []byte
	path string
}

func (reader *testFileReader) ReadFile(path string) ([]byte, error) {
	reader.path = path
	return reader.data, nil
}

func TestConfigurationPathUsesDefaultAndFileFlag(t *testing.T) {
	homeDirectory := func() (string, error) { return "/test/home", nil }
	for _, test := range []struct {
		name      string
		arguments []string
		want      string
	}{
		{"default path", nil, filepath.Join("/test/home", defaultConfigFile)},
		{"file flag", []string{"-f", "custom.toml"}, "custom.toml"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path, err := configurationPath(test.arguments, homeDirectory)
			if err != nil {
				t.Fatalf("configurationPath() error = %v", err)
			}
			if path != test.want {
				t.Errorf("configurationPath() = %q, want %q", path, test.want)
			}
		})
	}
}

func TestLoadConfigurationUsesSelectedPath(t *testing.T) {
	reader := &testFileReader{data: []byte("[[process]]\nname = 'web'\nstart = 'serve-web'\n")}
	homeDirectory := func() (string, error) { return "/test/home", nil }

	configuration, err := loadConfiguration([]string{"-f", "custom.toml"}, homeDirectory, reader)

	if err != nil {
		t.Fatalf("loadConfiguration() error = %v", err)
	}
	if reader.path != "custom.toml" {
		t.Errorf("ReadFile() path = %q, want %q", reader.path, "custom.toml")
	}
	if got := configuration.Processes[0].Name; got != "web" {
		t.Errorf("process name = %q, want %q", got, "web")
	}
}

func TestConfigurationPathReportsFlagAndHomeErrors(t *testing.T) {
	if _, err := configurationPath([]string{"-unknown"}, func() (string, error) {
		return "/test/home", nil
	}); err == nil {
		t.Fatal("configurationPath() error = nil, want flag error")
	}

	homeError := errors.New("unavailable")
	if _, err := configurationPath(nil, func() (string, error) {
		return "", homeError
	}); !errors.Is(err, homeError) {
		t.Fatalf("configurationPath() error = %v, want %v", err, homeError)
	}
}

func TestDiagnosticEnabled(t *testing.T) {
	for _, test := range []struct {
		arguments []string
		want      bool
	}{
		{arguments: []string{"-diagnostic"}, want: true},
		{arguments: []string{"--diagnostic"}, want: true},
		{arguments: []string{"-diagnostic=false"}, want: false},
		{arguments: []string{"-f", "custom.toml"}, want: false},
	} {
		got, err := diagnosticEnabled(test.arguments)
		if err != nil {
			t.Fatalf("diagnosticEnabled(%q) error = %v", test.arguments, err)
		}
		if got != test.want {
			t.Errorf("diagnosticEnabled(%q) = %t, want %t", test.arguments, got, test.want)
		}
	}
}

func TestExecuteWritesVersionWithoutLoadingConfiguration(t *testing.T) {
	for _, arguments := range [][]string{{"-v"}, {"-version"}, {"--version"}} {
		var output bytes.Buffer
		factory := &recordingApplicationFactory{application: &fakeApplication{}}
		lookup := &countingLookup{}

		err := execute(
			arguments,
			func() (string, error) { return "", errors.New("home directory accessed") },
			errorFileReader{err: errors.New("configuration read")},
			&output,
			lookup,
			factory.New,
		)

		if err != nil {
			t.Fatalf("execute(%q) error = %v", arguments, err)
		}
		if got, want := output.String(), "tp "+applicationVersion+"\n"; got != want {
			t.Errorf("execute(%q) output = %q, want %q", arguments, got, want)
		}
		if factory.calls != 0 {
			t.Errorf("execute(%q) started application", arguments)
		}
		if lookup.calls != 0 {
			t.Errorf("execute(%q) performed process lookup", arguments)
		}
	}
}

func TestExecuteWritesUsageWithoutLoadingConfiguration(t *testing.T) {
	for _, arguments := range [][]string{{"-h"}, {"--help"}, {"help"}, {"-h", "-v"}} {
		var output bytes.Buffer
		factory := &recordingApplicationFactory{application: &fakeApplication{}}
		lookup := &countingLookup{}

		err := execute(
			arguments,
			func() (string, error) { return "", errors.New("home directory accessed") },
			errorFileReader{err: errors.New("configuration read")},
			&output,
			lookup,
			factory.New,
		)

		if err != nil {
			t.Fatalf("execute(%q) error = %v", arguments, err)
		}
		if got, want := output.String(), usageText; got != want {
			t.Errorf("execute(%q) output = %q, want %q", arguments, got, want)
		}
		if factory.calls != 0 {
			t.Errorf("execute(%q) started application", arguments)
		}
		if lookup.calls != 0 {
			t.Errorf("execute(%q) performed process lookup", arguments)
		}
	}
}

func TestParseCommandLineOptionsRejectsUnexpectedArgument(t *testing.T) {
	_, err := parseCommandLineOptions([]string{"unexpected"})
	if err == nil {
		t.Fatal("parseCommandLineOptions() error = nil, want unexpected argument error")
	}
}

func TestConfigurationPathAcceptsDiagnosticFlag(t *testing.T) {
	path, err := configurationPath([]string{"-diagnostic"}, func() (string, error) {
		return "/test/home", nil
	})
	if err != nil {
		t.Fatalf("configurationPath() error = %v", err)
	}
	if want := filepath.Join("/test/home", defaultConfigFile); path != want {
		t.Errorf("configurationPath() = %q, want %q", path, want)
	}
}

type countingLookup struct {
	calls int
}

func (lookup *countingLookup) Check(config.Process) (process.Status, error) {
	lookup.calls++
	return process.Status{}, nil
}

func TestExecuteReturnsConfigurationErrorBeforeDiagnosticLookup(t *testing.T) {
	loadError := errors.New("read configuration")
	reader := errorFileReader{err: loadError}
	lookup := &countingLookup{}
	factory := &recordingApplicationFactory{application: &fakeApplication{}}

	err := execute(
		[]string{"-diagnostic"},
		func() (string, error) { return "/test/home", nil },
		reader,
		io.Discard,
		lookup,
		factory.New,
	)

	if !errors.Is(err, loadError) {
		t.Fatalf("execute() error = %v, want %v", err, loadError)
	}
	if lookup.calls != 0 {
		t.Errorf("lookup calls = %d, want 0", lookup.calls)
	}
	if factory.calls != 0 {
		t.Error("application was created after configuration loading failed")
	}
}

func TestExecuteWritesDiagnosticListWithoutRunningProgram(t *testing.T) {
	reader := &testFileReader{data: []byte("[[process]]\nname = 'web'\nstart = 'serve-web'\n")}
	lookup := &countingLookup{}
	factory := &recordingApplicationFactory{application: &fakeApplication{}}
	var output bytes.Buffer

	err := execute(
		[]string{"-diagnostic"},
		func() (string, error) { return "/test/home", nil },
		reader,
		&output,
		lookup,
		factory.New,
	)

	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}
	if got, want := output.String(), "web: stopped\n"; got != want {
		t.Errorf("diagnostic output = %q, want %q", got, want)
	}
	if lookup.calls != 1 {
		t.Errorf("lookup calls = %d, want 1", lookup.calls)
	}
	if factory.calls != 0 {
		t.Error("application was created in diagnostic mode")
	}
}

func TestExecuteRunsProgramWhenDiagnosticIsDisabled(t *testing.T) {
	reader := &testFileReader{data: []byte("[[process]]\nname = 'web'\nstart = 'serve-web'\n")}
	lookup := &countingLookup{}
	program := &fakeApplication{}
	factory := &recordingApplicationFactory{application: program}

	err := execute(
		[]string{"-diagnostic=false"},
		func() (string, error) { return "/test/home", nil },
		reader,
		io.Discard,
		lookup,
		factory.New,
	)

	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}
	if lookup.calls != 0 {
		t.Errorf("lookup calls = %d, want 0", lookup.calls)
	}
	if !program.ran {
		t.Error("program did not run when diagnostic mode was disabled")
	}
	if factory.calls != 1 {
		t.Errorf("application factory calls = %d, want 1", factory.calls)
	}
	if got := factory.processes[0].Name; got != "web" {
		t.Errorf("factory process name = %q, want web", got)
	}
	if factory.lookup != lookup {
		t.Error("factory did not receive lookup")
	}
}

type errorFileReader struct {
	err error
}

func (reader errorFileReader) ReadFile(string) ([]byte, error) {
	return nil, reader.err
}

var _ diagnostic.Lookup = (*countingLookup)(nil)

var _ config.FileReader = (*testFileReader)(nil)
