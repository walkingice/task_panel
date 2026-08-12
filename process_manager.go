package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/walkingice/process_manager/config"
	"github.com/walkingice/process_manager/diagnostic"
	"github.com/walkingice/process_manager/process"
	"github.com/walkingice/process_manager/ui"
)

const defaultConfigFile = ".conf/jchu/process_manager.toml"

type application interface {
	Run() (tea.Model, error)
}

type applicationFactory func([]config.Process, diagnostic.Lookup) application

func main() {
	lookup := process.Lookup{Shell: process.SystemShell{}, Inspector: process.SystemInspector{}}
	if err := execute(
		os.Args[1:], os.UserHomeDir, osFileReader{}, os.Stdout, lookup, newApplication,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(
	arguments []string,
	homeDirectory func() (string, error),
	reader config.FileReader,
	output io.Writer,
	lookup diagnostic.Lookup,
	newApplication applicationFactory,
) error {
	configuration, err := loadConfiguration(arguments, homeDirectory, reader)
	if err != nil {
		return err
	}
	diagnosticMode, err := diagnosticEnabled(arguments)
	if err != nil {
		return err
	}
	if diagnosticMode {
		return diagnostic.List(output, configuration.Processes, lookup)
	}
	return run(newApplication(configuration.Processes, lookup))
}

func newApplication(processes []config.Process, lookup diagnostic.Lookup) application {
	controller := process.Controller{
		Shell:    process.SystemShell{},
		Launcher: process.SystemLauncher{},
		Signaler: process.SystemSignaler{},
	}
	return tea.NewProgram(ui.New(processes, lookup, controller))
}

func diagnosticEnabled(arguments []string) (bool, error) {
	flags := flag.NewFlagSet("pm", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.String("f", "", "configuration file path")
	diagnostic := flags.Bool("diagnostic", false, "list process states without starting the TUI")
	if err := flags.Parse(arguments); err != nil {
		return false, fmt.Errorf("parse command-line flags: %w", err)
	}
	return *diagnostic, nil
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
	flags.Bool("diagnostic", false, "list process states without starting the TUI")
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
