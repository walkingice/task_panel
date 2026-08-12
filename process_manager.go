package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/walkingice/process_manager/ui"
)

type application interface {
	Run() (tea.Model, error)
}

func main() {
	program := tea.NewProgram(ui.New())
	if err := run(program); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(program application) error {
	_, err := program.Run()
	return err
}
