package diagnostic

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"task_panel/internal/config"
	"task_panel/internal/process"
)

type fakeLookup struct {
	results map[string]lookupResult
}

type lookupResult struct {
	status process.Status
	err    error
}

func (lookup fakeLookup) Check(configured config.Process) (process.Status, error) {
	result := lookup.results[configured.Name]
	return result.status, result.err
}

func TestListWritesProcessStates(t *testing.T) {
	lookupError := errors.New("inspect process")
	lookup := fakeLookup{results: map[string]lookupResult{
		"web":    {status: process.Status{Running: true}},
		"worker": {status: process.Status{}},
		"cache":  {err: lookupError},
	}}
	processes := []config.Process{{Name: "web"}, {Name: "worker"}, {Name: "cache"}}
	var output bytes.Buffer

	if err := List(&output, processes, lookup); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	want := "web: running\nworker: stopped\ncache: error: inspect process\n"
	if got := output.String(); got != want {
		t.Errorf("List() output = %q, want %q", got, want)
	}
}

func TestListReturnsWriteError(t *testing.T) {
	writeError := errors.New("write failed")
	err := List(errorWriter{err: writeError}, []config.Process{{Name: "web"}}, fakeLookup{})

	if !errors.Is(err, writeError) {
		t.Fatalf("List() error = %v, want %v", err, writeError)
	}
	if !strings.Contains(err.Error(), "write diagnostic list") {
		t.Errorf("List() error = %q, want diagnostic context", err)
	}
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

var _ io.Writer = errorWriter{}
