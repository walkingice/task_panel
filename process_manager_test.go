package main

import (
	"bytes"
	"errors"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
