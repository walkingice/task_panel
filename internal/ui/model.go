// Package ui contains Process Manager's terminal user interface.
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"task_panel/internal/config"
	"task_panel/internal/process"
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
	items                 []item
	lookup                Lookup
	controller            Controller
	showStartConfirmation bool
	showStopConfirmation  bool
	selected              int
	confirmation          *confirmation
	messages              []string
	width                 int
	height                int
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
func New(
	configured []config.Process,
	lookup Lookup,
	controller Controller,
	confirmations ...bool,
) Model {
	showStartConfirmation := true
	showStopConfirmation := true
	if len(confirmations) > 0 {
		showStartConfirmation = confirmations[0]
	}
	if len(confirmations) > 1 {
		showStopConfirmation = confirmations[1]
	}
	items := make([]item, len(configured))
	for index, process := range configured {
		items[index].configured = process
	}
	return Model{
		items:                 items,
		lookup:                lookup,
		controller:            controller,
		showStartConfirmation: showStartConfirmation,
		showStopConfirmation:  showStopConfirmation,
	}
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
		model.height = message.Height
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
		action, ok := model.selectedAction()
		if !ok {
			return model, nil
		}
		if model.shouldConfirm(action) {
			model.openConfirmation(action)
			return model, nil
		}
		return model.startControl(confirmation{index: model.selected, action: action})
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
		return model.startControl(confirmation)
	}
	return model, nil
}

func (model Model) startControl(confirmation confirmation) (tea.Model, tea.Cmd) {
	model.items[confirmation.index].enabled = false
	return model, controlProcess(confirmation, model.items[confirmation.index], model.controller)
}

func (model Model) selectedAction() (string, bool) {
	if model.selected < 0 || model.selected >= len(model.items) || !model.items[model.selected].enabled {
		return "", false
	}
	action := "start"
	if model.items[model.selected].status.Running {
		action = "stop"
	}
	return action, true
}

func (model Model) shouldConfirm(action string) bool {
	return (action == "start" && model.showStartConfirmation) ||
		(action == "stop" && model.showStopConfirmation)
}

func (model *Model) openConfirmation(action string) {
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
	height := model.height
	if height == 0 {
		height = 30
	}
	if width < 24 || height < 16 {
		return renderCompact(model, width)
	}
	detailsHeight := max(5, height*30/100)
	processHeight := height - detailsHeight - 2
	view := renderTitle(width) + "\n" + renderProcessTable(model.items, model.selected, width, processHeight)
	view += "\n" + renderDetails(model, width, detailsHeight) + "\n" + renderFooter(width)
	if model.confirmation == nil {
		return view + "\n"
	}
	return renderConfirmation(model, view, width, height)
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
	bold   = "\033[1m"
	green  = "\033[38;5;42m"
	red    = "\033[38;5;196m"
	purple = "\033[38;5;141m"
	gray   = "\033[38;5;245m"
	accent = "\033[48;5;60m\033[97m"
)

func renderTitle(width int) string {
	return accent + pad(" Process Manager ", width) + reset
}

func renderProcessTable(items []item, selected, width, height int) string {
	lines := []string{fmt.Sprintf("%-3s %-24s %-1s %-11s %-9s %s", "", "PROCESS", "", "STATUS", "PID", "COMMAND")}
	visibleItems := min(len(items), max(0, height-5))
	for index, item := range items[:visibleItems] {
		marker := " "
		if index == selected {
			marker = ">"
		}
		status := statusLabel(item)
		lines = append(lines, fmt.Sprintf("%-3s %-24s %-1s %-11s %-9s %s", marker,
			truncate(item.configured.Name, 24), statusIcon(status), status,
			pidLabel(item), truncate(item.configured.Start, max(1, width-56))))
	}
	if len(items) == 0 {
		lines = append(lines, "   No configured processes")
	}
	selectedLine := -1
	if selected < visibleItems {
		selectedLine = selected + 1
	}
	view := renderBox("Tasks", lines, width, selectedLine, height-2, 1)
	for _, item := range items {
		status := statusLabel(item)
		view = strings.Replace(view, pad(status, 11), renderStatusCell(status), 1)
	}
	return view
}

func renderDetails(model Model, width, height int) string {
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
	view := renderBox("Details", lines, width, -1, height-2, 0)
	if len(model.items) > 0 {
		view = strings.Replace(view, "Status: "+statusLabel(model.items[model.selected]),
			"Status: "+renderStatus(statusLabel(model.items[model.selected])), 1)
	}
	return view
}

func renderFooter(width int) string {
	return gray + pad(truncate(" ↑/k ↓/j navigate  Enter start/stop  q quit ", width), width) + reset
}

func renderBox(title string, lines []string, width, selectedLine, contentHeight, verticalPadding int) string {
	inside := width - 2
	top := purple + "┌ " + title + " " + strings.Repeat("─", max(0, inside-len(title)-2)) + "┐" + reset
	bottom := purple + "└" + strings.Repeat("─", inside) + "┘" + reset
	rendered := make([]string, 0, contentHeight+2)
	rendered = append(rendered, top)
	for index := 0; index < contentHeight; index++ {
		line := ""
		lineIndex := index - verticalPadding
		if lineIndex >= 0 && lineIndex < len(lines) {
			line = lines[lineIndex]
		}
		content := " " + truncate(line, inside-1)
		if lineIndex >= 0 && lineIndex == selectedLine {
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

func renderStatus(status string) string {
	if status == "running" {
		return bold + status + reset
	}
	return status
}

func renderStatusCell(status string) string {
	if status == "running" {
		return bold + status + reset + strings.Repeat(" ", 11-len(status))
	}
	return pad(status, 11)
}

func statusIcon(status string) string {
	if status == "running" {
		return "✔"
	}
	return ""
}

func renderConfirmation(model Model, view string, width, height int) string {
	item := model.items[model.confirmation.index]
	dialogWidth := min(60, width-4)
	lines := []string{
		"Confirm " + strings.ToUpper(model.confirmation.action),
		"",
		"Process: " + item.configured.Name,
		"This will " + model.confirmation.action + " the selected process.",
		"",
		"[Enter/y] Confirm    [Esc/n/q] Cancel",
	}
	dialog := renderBox("Confirmation", lines, dialogWidth, -1, len(lines), 0)
	dialog = strings.Replace(dialog, "Confirm "+strings.ToUpper(model.confirmation.action),
		renderConfirmationAction(model.confirmation.action), 1)
	dialogLines := strings.Split(dialog, "\n")
	top := max(1, (height-len(dialogLines))/2+1)
	left := max(1, (width-dialogWidth)/2+1)
	viewLines := strings.Split(view, "\n")
	for index, line := range dialogLines {
		viewLines[top-1+index] = strings.Repeat(" ", left-1) + line
	}
	return strings.Join(viewLines, "\n") + "\n"
}

func renderConfirmationAction(action string) string {
	color := green
	if action == "stop" {
		color = red
	}
	return "Confirm " + color + strings.ToUpper(action) + reset
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
