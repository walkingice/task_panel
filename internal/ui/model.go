// Package ui contains Process Manager's terminal user interface.
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"process_manager/internal/config"
	"process_manager/internal/process"
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
	width        int
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
	case tea.WindowSizeMsg:
		model.width = message.Width
		return model, nil
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

// View renders the process dashboard and persistent message panel.
func (model Model) View() string {
	width := model.width
	if width == 0 {
		width = 100
	}
	if width < 24 {
		return renderCompact(model, width)
	}
	view := renderTitle(width) + "\n" + renderProcessTable(model.items, model.selected, width)
	view += "\n" + renderDetails(model, width) + "\n" + renderFooter(width)
	if model.confirmation == nil {
		return view + "\n"
	}
	item := model.items[model.confirmation.index]
	confirmation := fmt.Sprintf("%s %q? Enter/y confirm; Esc/n/q cancel",
		model.confirmation.action, item.configured.Name)
	return view + "\n" + renderBox("Confirm", []string{confirmation}, width, -1) + "\n"
}

func renderCompact(model Model, width int) string {
	text := "PM"
	if model.confirmation != nil {
		return truncate("[y/n] "+model.confirmation.action, width) + "\n"
	}
	if len(model.items) > 0 {
		text += ": " + model.items[model.selected].configured.Name
	}
	return truncate(text, width) + "\n"
}

const (
	reset  = "\033[0m"
	purple = "\033[38;5;141m"
	gray   = "\033[38;5;245m"
	accent = "\033[48;5;60m\033[97m"
)

func renderTitle(width int) string {
	return accent + pad(" Process Manager ", width) + reset
}

func renderProcessTable(items []item, selected, width int) string {
	lines := []string{fmt.Sprintf("%-3s %-24s %-11s %-9s %s", "", "PROCESS", "STATUS", "PID", "COMMAND")}
	for index, item := range items {
		marker := " "
		if index == selected {
			marker = ">"
		}
		lines = append(lines, fmt.Sprintf("%-3s %-24s %-11s %-9s %s", marker,
			truncate(item.configured.Name, 24), statusLabel(item), pidLabel(item),
			truncate(item.configured.Start, max(1, width-54))))
	}
	if len(items) == 0 {
		lines = append(lines, "   No configured processes")
	}
	selectedLine := -1
	if len(items) > 0 {
		selectedLine = selected + 1
	}
	return renderBox("Processes", lines, width, selectedLine)
}

func renderDetails(model Model, width int) string {
	lines := []string{"No process selected"}
	if len(model.items) > 0 {
		item := model.items[model.selected]
		status := "stopped"
		if item.status.Running {
			status = "running"
		}
		if !item.enabled {
			status = "checking"
		}
		lines = []string{
			"Process: " + item.configured.Name,
			"Command: " + item.configured.Start,
			"Status: " + status,
			"PIDs: " + pidLabel(item),
		}
	}
	if len(model.messages) > 0 {
		lines = append(lines, "", "Activity:")
		lines = append(lines, model.messages...)
	}
	return renderBox("Details", lines, width, -1)
}

func renderFooter(width int) string {
	return gray + pad(truncate(" ↑/k ↓/j navigate  Enter start/stop  q quit ", width), width) + reset
}

func renderBox(title string, lines []string, width, selectedLine int) string {
	inside := width - 2
	top := purple + "┌ " + title + " " + strings.Repeat("─", max(0, inside-len(title)-2)) + "┐" + reset
	bottom := purple + "└" + strings.Repeat("─", inside) + "┘" + reset
	rendered := make([]string, 0, len(lines)+2)
	rendered = append(rendered, top)
	for index, line := range lines {
		content := " " + truncate(line, inside-1)
		if index == selectedLine {
			content = accent + pad(content, inside) + reset
		} else {
			content = pad(content, inside)
		}
		rendered = append(rendered, purple+"│"+reset+content+purple+"│"+reset)
	}
	rendered = append(rendered, bottom)
	return strings.Join(rendered, "\n")
}

func statusLabel(item item) string {
	if !item.enabled {
		return "checking"
	}
	if item.status.Running {
		return "running"
	}
	return "stopped"
}

func pidLabel(item item) string {
	if len(item.status.PIDs) == 0 {
		return "-"
	}
	return fmt.Sprint(item.status.PIDs[0])
}

func pad(value string, width int) string {
	return value + strings.Repeat(" ", max(0, width-len([]rune(value))))
}

func truncate(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width < 2 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}
