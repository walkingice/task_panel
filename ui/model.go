// Package ui contains Process Manager's terminal user interface.
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/walkingice/process_manager/config"
	"github.com/walkingice/process_manager/process"
)

// Lookup checks the status of a configured process.
type Lookup interface {
	Check(config.Process) (process.Status, error)
}

// Model is the Process Manager main-view state.
type Model struct {
	items    []item
	lookup   Lookup
	selected int
	messages []string
}

type item struct {
	configured config.Process
	status     process.Status
	enabled    bool
}

type statusCheckedMsg struct {
	index  int
	status process.Status
	err    error
}

// New returns a main view populated with configured processes.
func New(configured []config.Process, lookup Lookup) Model {
	items := make([]item, len(configured))
	for index, process := range configured {
		items[index].configured = process
	}
	return Model{items: items, lookup: lookup}
}

// Init starts asynchronous status checks for all configured processes.
func (model Model) Init() tea.Cmd {
	commands := make([]tea.Cmd, len(model.items))
	for index, process := range model.items {
		commands[index] = checkStatus(index, process.configured, model.lookup)
	}
	return tea.Batch(commands...)
}

func checkStatus(index int, configured config.Process, lookup Lookup) tea.Cmd {
	return func() tea.Msg {
		status, err := lookup.Check(configured)
		return statusCheckedMsg{index: index, status: status, err: err}
	}
}

// Update handles status updates, navigation, and application exit keys.
func (model Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case statusCheckedMsg:
		return model.applyStatus(message), nil
	case tea.KeyMsg:
		return model.updateKey(message)
	}
	return model, nil
}

func (model Model) applyStatus(message statusCheckedMsg) Model {
	if message.index < 0 || message.index >= len(model.items) {
		return model
	}
	model.items[message.index].status = message.status
	model.items[message.index].enabled = true
	if message.err != nil {
		name := model.items[message.index].configured.Name
		model.messages = append(model.messages, fmt.Sprintf("%s: %v", name, message.err))
	}
	return model
}

func (model Model) updateKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "esc":
		return model, tea.Quit
	case "up", "k":
		model.moveSelection(-1)
	case "down", "j":
		model.moveSelection(1)
	}
	return model, nil
}

func (model *Model) moveSelection(change int) {
	if len(model.items) == 0 {
		return
	}
	model.selected += change
	if model.selected < 0 {
		model.selected = 0
	}
	if model.selected >= len(model.items) {
		model.selected = len(model.items) - 1
	}
}

// View renders the process list and persistent message panel.
func (model Model) View() string {
	processLines := []string{"Process Manager", "", "Processes:"}
	for index, item := range model.items {
		processLines = append(processLines, renderItem(index == model.selected, item))
	}
	messageLines := append([]string{"Messages:"}, model.messages...)
	return renderColumns(processLines, messageLines)
}

func renderColumns(left, right []string) string {
	width := 0
	for _, line := range left {
		if len(line) > width {
			width = len(line)
		}
	}
	lineCount := max(len(left), len(right))
	lines := make([]string, lineCount)
	for index := range lines {
		leftLine := ""
		if index < len(left) {
			leftLine = left[index]
		}
		rightLine := ""
		if index < len(right) {
			rightLine = right[index]
		}
		lines[index] = fmt.Sprintf("%-*s │ %s", width, leftLine, rightLine)
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderItem(selected bool, item item) string {
	prefix := " "
	if selected {
		prefix = ">"
	}
	if !item.enabled {
		return fmt.Sprintf("%s [ ] %s (checking)", prefix, item.configured.Name)
	}
	if item.status.Running {
		return fmt.Sprintf("%s [✔] %s", prefix, item.configured.Name)
	}
	return fmt.Sprintf("%s [ ] %s", prefix, item.configured.Name)
}
