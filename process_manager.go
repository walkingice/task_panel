package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/walkingice/process_manager/config"
	"github.com/walkingice/process_manager/ui"
)

const defaultConfigFile = ".conf/jchu/process_manager.toml"

type application interface {
	Run() (tea.Model, error)
}

func main() {
	if _, err := loadConfiguration(os.Args[1:], os.UserHomeDir, osFileReader{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

type osFileReader struct{}

func (osFileReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func loadConfiguration(
	arguments []string,
	homeDirectory func() (string, error),
	reader config.FileReader,
) (config.Configuration, error) {
	path, err := configurationPath(arguments, homeDirectory)
	if err != nil {
		return config.Configuration{}, err
	}
	return config.Load(path, reader, config.TOMLDecoder{})
}

func configurationPath(arguments []string, homeDirectory func() (string, error)) (string, error) {
	flags := flag.NewFlagSet("pm", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("f", "", "configuration file path")
	if err := flags.Parse(arguments); err != nil {
		return "", fmt.Errorf("parse command-line flags: %w", err)
	}
	if *path != "" {
		return *path, nil
	}

	home, err := homeDirectory()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, defaultConfigFile), nil
}
