// Package ui contains Process Manager's terminal user interface.
package ui

import tea "github.com/charmbracelet/bubbletea"

// Model is the initial Process Manager user interface state.
type Model struct{}

// New returns the initial user interface model.
func New() Model {
	return Model{}
}

// Init starts no background work during the foundation phase.
func (Model) Init() tea.Cmd {
	return nil
}

// Update handles application exit keys.
func (model Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if ok && (key.String() == "q" || key.String() == "esc") {
		return model, tea.Quit
	}
	return model, nil
}

// View renders the foundation screen.
func (Model) View() string {
	return "Process Manager\n\nPress q to quit.\n"
}
