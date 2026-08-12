package ui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/walkingice/process_manager/config"
	"github.com/walkingice/process_manager/process"
)

type fakeLookup struct {
	statuses map[string]process.Status
	errors   map[string]error
}

func (lookup fakeLookup) Check(configured config.Process) (process.Status, error) {
	return lookup.statuses[configured.Name], lookup.errors[configured.Name]
}

func TestModelInitializesDisabledItemsAndChecksStatuses(t *testing.T) {
	model := New([]config.Process{{Name: "web"}, {Name: "worker"}}, fakeLookup{
		statuses: map[string]process.Status{"web": {Running: true, PIDs: []int32{123}}},
	})
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
	model := New([]config.Process{{Name: "web"}}, fakeLookup{errors: map[string]error{"web": lookupError}})

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
	model := New([]config.Process{{Name: "web"}}, fakeLookup{})

	updated, _ := model.Update(statusCheckedMsg{index: 1, status: process.Status{Running: true}})
	model = updated.(Model)

	if model.items[0].enabled || model.items[0].status.Running {
		t.Errorf("item changed after unknown status update: %+v", model.items[0])
	}
}

func TestModelNavigatesWithArrowsAndVimKeys(t *testing.T) {
	model := New([]config.Process{{Name: "web"}, {Name: "worker"}}, fakeLookup{})
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
	model := New(nil, fakeLookup{})
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
