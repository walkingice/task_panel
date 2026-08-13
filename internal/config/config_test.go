package config

import (
	"errors"
	"strings"
	"testing"
)

type fakeFileReader struct {
	data []byte
	err  error
	path string
}

func (reader *fakeFileReader) ReadFile(path string) ([]byte, error) {
	reader.path = path
	return reader.data, reader.err
}

func TestTOMLDecoderUnmarshalsProcess(t *testing.T) {
	var configuration Configuration
	data := []byte("[[process]]\nname = 'web'\nstart = 'serve-web'\n")

	if err := (TOMLDecoder{}).Unmarshal(data, &configuration); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(configuration.Processes) != 1 {
		t.Fatalf("process count = %d, want 1", len(configuration.Processes))
	}
	if got := configuration.Processes[0].Name; got != "web" {
		t.Fatalf("process name = %q, want %q", got, "web")
	}
}

func TestLoadDefaultsAndReadsConfirmationOptions(t *testing.T) {
	for _, test := range []struct {
		name      string
		data      string
		wantStart bool
		wantStop  bool
	}{
		{
			name:      "defaults to confirmations enabled",
			data:      "[[process]]\nname = 'web'\nstart = 'serve-web'\n",
			wantStart: true,
			wantStop:  true,
		},
		{
			name: "reads disabled confirmations",
			data: "show_start_confirmation = false\nshow_stop_confirmation = false\n" +
				"[[process]]\nname = 'web'\nstart = 'serve-web'\n",
			wantStart: false,
			wantStop:  false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			configuration, err := Load("test.toml", &fakeFileReader{data: []byte(test.data)}, TOMLDecoder{})
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if configuration.ShowStartConfirmation != test.wantStart {
				t.Errorf("ShowStartConfirmation = %t, want %t", configuration.ShowStartConfirmation, test.wantStart)
			}
			if configuration.ShowStopConfirmation != test.wantStop {
				t.Errorf("ShowStopConfirmation = %t, want %t", configuration.ShowStopConfirmation, test.wantStop)
			}
		})
	}
}

func TestLoadReadsAndValidatesConfiguration(t *testing.T) {
	reader := &fakeFileReader{data: []byte("[[process]]\nname = 'web'\nstart = 'serve-web'\n")}

	configuration, err := Load("test.toml", reader, TOMLDecoder{})

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if reader.path != "test.toml" {
		t.Errorf("ReadFile() path = %q, want %q", reader.path, "test.toml")
	}
	if got := configuration.Processes[0].Start; got != "serve-web" {
		t.Errorf("process start = %q, want %q", got, "serve-web")
	}
}

func TestLoadReportsReadAndParseErrors(t *testing.T) {
	readError := errors.New("not found")
	for _, test := range []struct {
		name   string
		reader *fakeFileReader
		want   string
	}{
		{"missing file", &fakeFileReader{err: readError}, "read configuration \"test.toml\": not found"},
		{"malformed TOML", &fakeFileReader{data: []byte("[[process]")}, "parse configuration \"test.toml\":"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load("test.toml", test.reader, TOMLDecoder{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Load() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestValidateRejectsInvalidProcesses(t *testing.T) {
	for _, test := range []struct {
		name          string
		configuration Configuration
		want          string
	}{
		{"no processes", Configuration{}, "no process entries"},
		{"missing name", Configuration{Processes: []Process{{Start: "run"}}}, "process 1: missing name"},
		{"missing start", Configuration{Processes: []Process{{Name: "web"}}}, "process 1: missing start"},
		{"find without stop", Configuration{Processes: []Process{{Name: "web", Start: "run", Find: "pgrep web"}}}, "process 1: find requires stop"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.configuration.Validate()
			if err == nil || err.Error() != test.want {
				t.Fatalf("Validate() error = %v, want %q", err, test.want)
			}
		})
	}
}
