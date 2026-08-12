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

// Controller starts and stops configured processes.
type Controller interface {
	Start(config.Process) error
	Stop(config.Process, process.Status) error
}

// Model is the Process Manager main-view state.
type Model struct {
	items        []item
	lookup       Lookup
	controller   Controller
	selected     int
	confirmation *confirmation
	messages     []string
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

type confirmation struct {
	index  int
	action string
}

type controlFinishedMsg struct {
	index  int
	action string
	err    error
}

// New returns a main view populated with configured processes.
func New(configured []config.Process, lookup Lookup, controller Controller) Model {
	items := make([]item, len(configured))
	for index, process := range configured {
		items[index].configured = process
	}
	return Model{items: items, lookup: lookup, controller: controller}
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
	case controlFinishedMsg:
		return model.applyControl(message)
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
	if model.confirmation != nil {
		return model.updateConfirmation(key)
	}
	switch key.String() {
	case "q", "esc":
		return model, tea.Quit
	case "up", "k":
		model.moveSelection(-1)
	case "down", "j":
		model.moveSelection(1)
	case "enter":
		model.openConfirmation()
	}
	return model, nil
}

func (model Model) updateConfirmation(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "n", "q":
		model.confirmation = nil
	case "enter", "y":
		confirmation := *model.confirmation
		model.confirmation = nil
		model.items[confirmation.index].enabled = false
		return model, controlProcess(confirmation, model.items[confirmation.index], model.controller)
	}
	return model, nil
}

func (model *Model) openConfirmation() {
	if model.selected < 0 || model.selected >= len(model.items) || !model.items[model.selected].enabled {
		return
	}
	action := "start"
	if model.items[model.selected].status.Running {
		action = "stop"
	}
	model.confirmation = &confirmation{index: model.selected, action: action}
}

func controlProcess(confirmation confirmation, item item, controller Controller) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return controlFinishedMsg{index: confirmation.index, action: confirmation.action,
				err: fmt.Errorf("%s process %q: controller is unavailable", confirmation.action, item.configured.Name)}
		}
		var err error
		if confirmation.action == "start" {
			err = controller.Start(item.configured)
		} else {
			err = controller.Stop(item.configured, item.status)
		}
		return controlFinishedMsg{index: confirmation.index, action: confirmation.action, err: err}
	}
}

func (model Model) applyControl(message controlFinishedMsg) (tea.Model, tea.Cmd) {
	if message.index < 0 || message.index >= len(model.items) {
		return model, nil
	}
	name := model.items[message.index].configured.Name
	if message.err != nil {
		model.messages = append(model.messages, fmt.Sprintf("%s: %v", name, message.err))
	} else {
		model.messages = append(model.messages, fmt.Sprintf("%s: %s requested", name, message.action))
	}
	model.items[message.index].enabled = false
	return model, checkStatus(message.index, model.items[message.index].configured, model.lookup)
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
	view := renderColumns(processLines, messageLines)
	if model.confirmation == nil {
		return view
	}
	item := model.items[model.confirmation.index]
	return view + fmt.Sprintf("\nConfirm %s %q? [Enter/y] confirm [Esc/n/q] cancel\n",
		model.confirmation.action, item.configured.Name)
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
