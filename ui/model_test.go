package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelQuitsWithMainExitKeys(t *testing.T) {
	model := New()

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
