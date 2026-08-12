package ui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"process_manager/internal/config"
	"process_manager/internal/process"
)

type fakeLookup struct {
	statuses map[string]process.Status
	errors   map[string]error
}

type fakeController struct {
	starts []string
	stops  []string
	err    error
}

func (controller *fakeController) Start(configured config.Process) error {
	controller.starts = append(controller.starts, configured.Name)
	return controller.err
}

func (controller *fakeController) Stop(configured config.Process, _ process.Status) error {
	controller.stops = append(controller.stops, configured.Name)
	return controller.err
}

func (lookup fakeLookup) Check(configured config.Process) (process.Status, error) {
	return lookup.statuses[configured.Name], lookup.errors[configured.Name]
}

func TestModelInitializesDisabledItemsAndChecksStatuses(t *testing.T) {
	model := New([]config.Process{{Name: "web"}, {Name: "worker"}}, fakeLookup{
		statuses: map[string]process.Status{"web": {Running: true, PIDs: []int32{123}}},
	}, nil)
	if got := model.View(); !strings.Contains(got, "> [ ] web (checking)") ||
		!strings.Contains(got, "  [ ] worker (checking)") {
		t.Fatalf("initial View() = %q, want disabled items", got)
	}

	commands, ok := model.Init()().(tea.BatchMsg)
	if !ok {
		t.Fatal("Init() did not return a batch of status checks")
	}
	for index := len(commands) - 1; index >= 0; index-- {
		updated, _ := model.Update(commands[index]())
		model = updated.(Model)
	}
	if !model.items[0].enabled || !model.items[1].enabled {
		t.Fatal("items remain disabled after status checks")
	}
	if got, want := model.items[0].status, (process.Status{Running: true, PIDs: []int32{123}}); !reflect.DeepEqual(got, want) {
		t.Errorf("web status = %+v, want %+v", got, want)
	}
	if got := model.View(); !strings.Contains(got, "> [✔] web") ||
		!strings.Contains(got, "  [ ] worker") {
		t.Errorf("updated View() = %q, want running and stopped items", got)
	}
}

func TestModelDisplaysLookupErrors(t *testing.T) {
	lookupError := errors.New("lookup failed")
	model := New([]config.Process{{Name: "web"}}, fakeLookup{errors: map[string]error{"web": lookupError}}, nil)

	updated, _ := model.Update(model.Init()())
	model = updated.(Model)

	if !model.items[0].enabled {
		t.Fatal("item is disabled after a failed status check")
	}
	if got := model.View(); !strings.Contains(got, "Process Manager │ Messages:") ||
		!strings.Contains(got, "web: lookup failed") {
		t.Errorf("View() = %q, want persistent lookup error", got)
	}
}

func TestModelIgnoresUnknownStatusUpdate(t *testing.T) {
	model := New([]config.Process{{Name: "web"}}, fakeLookup{}, nil)

	updated, _ := model.Update(statusCheckedMsg{index: 1, status: process.Status{Running: true}})
	model = updated.(Model)

	if model.items[0].enabled || model.items[0].status.Running {
		t.Errorf("item changed after unknown status update: %+v", model.items[0])
	}
}

func TestModelNavigatesWithArrowsAndVimKeys(t *testing.T) {
	model := New([]config.Process{{Name: "web"}, {Name: "worker"}}, fakeLookup{}, nil)
	for _, key := range []tea.KeyMsg{{Type: tea.KeyDown}, {Type: tea.KeyRunes, Runes: []rune{'k'}}} {
		updated, _ := model.Update(key)
		model = updated.(Model)
	}
	if model.selected != 0 {
		t.Errorf("selected = %d, want 0", model.selected)
	}
	for _, key := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'j'}}, {Type: tea.KeyDown}} {
		updated, _ := model.Update(key)
		model = updated.(Model)
	}
	if model.selected != 1 {
		t.Errorf("selected = %d, want 1", model.selected)
	}
}

func TestModelQuitsWithMainExitKeys(t *testing.T) {
	model := New(nil, fakeLookup{}, nil)
	for _, key := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'q'}}, {Type: tea.KeyEsc}} {
		_, command := model.Update(key)
		if command == nil {
			t.Fatalf("Update(%q) returned nil command", key.String())
		}
		if _, ok := command().(tea.QuitMsg); !ok {
			t.Fatalf("Update(%q) did not return a quit command", key.String())
		}
	}
}

func TestModelConfirmsStartAndRefreshesStatus(t *testing.T) {
	lookup := fakeLookup{statuses: map[string]process.Status{
		"web": {Running: true, PIDs: []int32{123}},
	}}
	controller := &fakeController{}
	model := New([]config.Process{{Name: "web", Start: "serve-web"}}, lookup, controller)
	model.items[0].enabled = true

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.confirmation == nil || model.confirmation.action != "start" {
		t.Fatalf("confirmation = %#v, want start confirmation", model.confirmation)
	}
	if got := model.View(); !strings.Contains(got, "Confirm start \"web\"?") {
		t.Errorf("View() = %q, want confirmation modal", got)
	}
	if command != nil {
		t.Fatal("opening confirmation returned a command")
	}

	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.items[0].enabled {
		t.Fatal("item remains enabled while its control operation is in flight")
	}
	updated, duplicate := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.confirmation != nil || duplicate != nil {
		t.Fatal("in-flight item opened another confirmation")
	}
	updated, refresh := model.Update(command())
	model = updated.(Model)
	if got, want := controller.starts, []string{"web"}; !reflect.DeepEqual(got, want) {
		t.Errorf("start calls = %v, want %v", got, want)
	}
	if model.items[0].enabled {
		t.Fatal("item remains enabled while its status is refreshed")
	}
	updated, _ = model.Update(refresh())
	model = updated.(Model)
	if !model.items[0].enabled || !model.items[0].status.Running {
		t.Fatalf("refreshed item = %#v, want enabled running item", model.items[0])
	}
	if got := model.View(); !strings.Contains(got, "web: start requested") {
		t.Errorf("View() = %q, want start message", got)
	}
}

func TestModelConfirmsStopAndSupportsCancellation(t *testing.T) {
	controller := &fakeController{}
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEsc},
		{Type: tea.KeyRunes, Runes: []rune{'n'}},
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
	} {
		model := runningModel(controller)
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model = updated.(Model)
		if model.confirmation == nil || model.confirmation.action != "stop" {
			t.Fatalf("confirmation = %#v, want stop confirmation", model.confirmation)
		}
		updated, command := model.Update(key)
		model = updated.(Model)
		if model.confirmation != nil || command != nil {
			t.Fatalf("%q did not cancel confirmation", key.String())
		}
	}
	if len(controller.stops) != 0 {
		t.Errorf("stop calls = %v, want none", controller.stops)
	}

	model := runningModel(controller)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	var command tea.Cmd
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(Model)
	if command == nil {
		t.Fatal("y did not confirm action")
	}
	if _, ok := command().(controlFinishedMsg); !ok {
		t.Fatal("confirmation command did not control process")
	}
	if got, want := controller.stops, []string{"web"}; !reflect.DeepEqual(got, want) {
		t.Errorf("stop calls = %v, want %v", got, want)
	}
}

func runningModel(controller Controller) Model {
	model := New([]config.Process{{Name: "web", Start: "serve-web"}}, fakeLookup{}, controller)
	model.items[0] = item{configured: config.Process{Name: "web", Start: "serve-web"}, enabled: true,
		status: process.Status{Running: true, PIDs: []int32{123}}}
	return model
}

func TestModelReportsControlErrorsAndIgnoresDisabledItems(t *testing.T) {
	want := errors.New("cannot start")
	controller := &fakeController{err: want}
	model := New([]config.Process{{Name: "web", Start: "serve-web"}}, fakeLookup{}, controller)

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if model.confirmation != nil || command != nil {
		t.Fatal("disabled item opened a confirmation")
	}
	model.items[0].enabled = true
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, refresh := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(refresh())
	model = updated.(Model)
	if got := model.View(); !strings.Contains(got, "web: cannot start") {
		t.Errorf("View() = %q, want control error", got)
	}
}
