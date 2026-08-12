package main

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/walkingice/process_manager/config"
	"github.com/walkingice/process_manager/ui"
)

type fakeApplication struct {
	err error
	ran bool
}

func (app *fakeApplication) Run() (tea.Model, error) {
	app.ran = true
	return nil, app.err
}

func TestRunStartsAndExitsApplication(t *testing.T) {
	program := tea.NewProgram(
		ui.New(),
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

var _ config.FileReader = (*testFileReader)(nil)
