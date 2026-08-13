package ui

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"task_panel/internal/config"
	"task_panel/internal/process"
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
	if got := model.View(); !strings.Contains(got, "Processes") ||
		!strings.Contains(got, ">   web") || !strings.Contains(got, "checking") {
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
	if got := model.View(); !strings.Contains(got, ">   web") ||
		!strings.Contains(got, "running") || !strings.Contains(got, "stopped") {
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
	if got := model.View(); !strings.Contains(got, "Process Manager") ||
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

func TestModelUsesTerminalWidthForDashboard(t *testing.T) {
	model := New([]config.Process{{Name: "web", Start: "serve-web"}}, fakeLookup{}, nil)

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 72, Height: 24})
	model = updated.(Model)

	if model.width != 72 {
		t.Errorf("width = %d, want 72", model.width)
	}
	if got := model.View(); !strings.Contains(got, "┌ Processes ") ||
		!strings.Contains(got, " ↑/k ↓/j navigate") {
		t.Errorf("View() = %q, want dashboard framing and footer", got)
	}
}

func TestModelReservesBottomThirtyPercentForDetails(t *testing.T) {
	model := New([]config.Process{{Name: "web"}}, fakeLookup{}, nil)

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	model = updated.(Model)

	view := stripANSI(model.View())
	lines := strings.Split(view, "\n")
	detailsStart := indexOfLine(lines, "┌ Details ")
	footer := indexOfLine(lines, " ↑/k ↓/j navigate")
	if got, want := footer-detailsStart, 9; got != want {
		t.Errorf("Details height = %d, want %d", got, want)
	}
}

func TestModelFitsWithinTerminalHeight(t *testing.T) {
	model := New([]config.Process{{Name: "web"}}, fakeLookup{}, nil)

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	model = updated.(Model)

	if got, want := len(strings.Split(strings.TrimSuffix(model.View(), "\n"), "\n")), 30; got != want {
		t.Errorf("View() has %d lines, want %d", got, want)
	}
}

func TestProcessTableKeepsVerticalPadding(t *testing.T) {
	items := make([]item, 10)
	for index := range items {
		items[index].configured.Name = fmt.Sprintf("process-%d", index)
	}

	lines := strings.Split(stripANSI(renderProcessTable(items, 0, 80, 9)), "\n")
	if got, want := lines[1], "│"+strings.Repeat(" ", 78)+"│"; got != want {
		t.Errorf("top padding = %q, want %q", got, want)
	}
	if got, want := lines[len(lines)-2], "│"+strings.Repeat(" ", 78)+"│"; got != want {
		t.Errorf("bottom padding = %q, want %q", got, want)
	}
}

func TestProcessTableUsesSeparateRunningIconCell(t *testing.T) {
	items := []item{
		{configured: config.Process{Name: "web"}, enabled: true,
			status: process.Status{Running: true}},
		{configured: config.Process{Name: "worker"}, enabled: true},
	}

	view := renderProcessTable(items, 0, 100, 9)
	if !strings.Contains(stripANSI(view), "✔ running") ||
		!strings.Contains(view, bold+"running"+reset) {
		t.Errorf("renderProcessTable() = %q, want a check icon before bold running", view)
	}

	lines := strings.Split(stripANSI(view), "\n")
	running := strings.Index(lines[3], "running")
	stopped := strings.Index(lines[4], "stopped")
	if got, want := utf8.RuneCountInString(lines[3][:running]), utf8.RuneCountInString(lines[4][:stopped]); got != want {
		t.Errorf("status positions = %d and %d, want aligned", got, want)
	}
}

func TestConfirmationDialogIsCentered(t *testing.T) {
	model := New([]config.Process{{Name: "web"}}, fakeLookup{}, nil)
	model.confirmation = &confirmation{index: 0, action: "start"}

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	model = updated.(Model)

	lines := strings.Split(model.View(), "\n")
	if got := stripANSI(lines[11]); !strings.HasPrefix(got, strings.Repeat(" ", 10)+"┌ Confirmation ") ||
		!strings.Contains(model.View(), "[Enter/y] Confirm") ||
		!strings.Contains(model.View(), "[Esc/n/q] Cancel") {
		t.Errorf("View() = %q, want centered confirmation dialog", model.View())
	}
}

func TestConfirmationDialogDoesNotUseCursorPositioning(t *testing.T) {
	model := New([]config.Process{{Name: "web"}}, fakeLookup{}, nil)
	model.confirmation = &confirmation{index: 0, action: "start"}

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	model = updated.(Model)

	if got := model.View(); strings.Contains(got, "\033[") && strings.Contains(got, "H") {
		t.Errorf("View() = %q, want dialog composed without cursor positioning", got)
	}
}

func TestConfirmationDialogColorsAction(t *testing.T) {
	tests := []struct {
		action string
		color  string
	}{
		{action: "start", color: green},
		{action: "stop", color: red},
	}

	for _, test := range tests {
		t.Run(test.action, func(t *testing.T) {
			model := New([]config.Process{{Name: "web"}}, fakeLookup{}, nil)
			model.confirmation = &confirmation{index: 0, action: test.action}
			model.width = 80
			model.height = 30

			if got := model.View(); !strings.Contains(got,
				"Confirm "+test.color+strings.ToUpper(test.action)+reset) {
				t.Errorf("View() = %q, want %s action color", got, test.action)
			}
		})
	}
}

func TestModelKeepsDashboardWithinNarrowTerminal(t *testing.T) {
	model := New([]config.Process{{Name: "web", Start: "serve-web"}}, fakeLookup{}, nil)

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	model = updated.(Model)

	for _, line := range strings.Split(model.View(), "\n") {
		if len([]rune(stripANSI(line))) > 40 {
			t.Errorf("line width = %d, want at most 40: %q", len([]rune(stripANSI(line))), line)
		}
	}
}

func TestModelUsesCompactViewForVeryNarrowTerminal(t *testing.T) {
	model := New([]config.Process{{Name: "web"}}, fakeLookup{}, nil)

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 12, Height: 24})
	model = updated.(Model)

	if got, want := model.View(), "PM: web\n"; got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestModelUsesCompactViewForShortTerminal(t *testing.T) {
	model := New([]config.Process{{Name: "web"}}, fakeLookup{}, nil)

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 15})
	model = updated.(Model)

	if got, want := model.View(), "PM: web\n"; got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestModelShowsConfirmationInCompactView(t *testing.T) {
	for _, width := range []int{12, 20} {
		t.Run(fmt.Sprintf("width %d", width), func(t *testing.T) {
			model := New([]config.Process{{Name: "web"}}, fakeLookup{}, nil)
			model.confirmation = &confirmation{index: 0, action: "start"}

			updated, _ := model.Update(tea.WindowSizeMsg{Width: width, Height: 24})
			model = updated.(Model)

			if got, want := model.View(), "[y/n] start\n"; got != want {
				t.Errorf("View() = %q, want %q", got, want)
			}
		})
	}
}

func TestModelKeepsConfirmationWithinNarrowTerminal(t *testing.T) {
	model := New([]config.Process{{Name: "a-very-long-process-name"}}, fakeLookup{}, nil)
	model.confirmation = &confirmation{index: 0, action: "start"}

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	model = updated.(Model)

	view := model.View()
	for _, line := range strings.Split(view, "\n") {
		if len([]rune(stripANSI(line))) > 40 {
			t.Errorf("line width = %d, want at most 40: %q", len([]rune(stripANSI(line))), line)
		}
	}
}

func TestModelDoesNotHighlightEmptyProcessList(t *testing.T) {
	if got := renderProcessTable(nil, 0, 100, 20); strings.Contains(got, accent) {
		t.Errorf("renderProcessTable() = %q, want an unselected empty-list message", got)
	}
}

func stripANSI(value string) string {
	for {
		start := strings.Index(value, "\033[")
		if start == -1 {
			return value
		}
		end := strings.IndexAny(value[start:], "mH")
		if end == -1 {
			return value
		}
		value = value[:start] + value[start+end+1:]
	}
}

func indexOfLine(lines []string, prefix string) int {
	for index, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return index
		}
	}
	return -1
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
	if got := model.View(); !strings.Contains(got, "┌ Confirmation ") ||
		!strings.Contains(got, "Process: web") ||
		!strings.Contains(got, "[Enter/y] Confirm") {
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
