package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"task_panel/internal/config"
	"task_panel/internal/diagnostic"
	"task_panel/internal/process"
	"task_panel/internal/ui"
)

const (
	applicationVersion = "0.1.0"
	defaultConfigFile  = ".config/task_panel/config.toml"
	usageText          = `Usage: tp [options]

Manage configured processes.

Options:
  -f <path>        configuration file path
  -diagnostic      list process states without starting the TUI
  -v, -version     print version and exit
  -h, -help        show this help message
`
)

type application interface {
	Run() (tea.Model, error)
}

type applicationFactory func(config.Configuration, diagnostic.Lookup) application

type programFactory func(tea.Model, ...tea.ProgramOption) *tea.Program

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
	options, err := parseCommandLineOptions(arguments)
	if err != nil {
		return err
	}
	if options.help {
		return writeUsage(output)
	}
	if options.version {
		_, err := fmt.Fprintf(output, "tp %s\n", applicationVersion)
		return err
	}

	path, err := configurationPathFromOption(options.configFile, homeDirectory)
	if err != nil {
		return err
	}
	configuration, err := config.Load(path, reader, config.TOMLDecoder{})
	if err != nil {
		return err
	}
	if options.diagnostic {
		return diagnostic.List(output, configuration.Processes, lookup)
	}
	return run(newApplication(configuration, lookup))
}

func newApplication(configuration config.Configuration, lookup diagnostic.Lookup) application {
	return newApplicationWithProgram(configuration, lookup, tea.NewProgram)
}

func newApplicationWithProgram(
	configuration config.Configuration,
	lookup diagnostic.Lookup,
	newProgram programFactory,
) application {
	controller := process.Controller{
		Shell:    process.SystemShell{},
		Launcher: process.SystemLauncher{},
		Signaler: process.SystemSignaler{},
	}
	return newProgram(ui.New(
		configuration.Processes,
		lookup,
		controller,
		configuration.ShowStartConfirmation,
		configuration.ShowStopConfirmation,
	), tea.WithAltScreen())
}

func diagnosticEnabled(arguments []string) (bool, error) {
	options, err := parseCommandLineOptions(arguments)
	return options.diagnostic, err
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
	options, err := parseCommandLineOptions(arguments)
	if err != nil {
		return "", err
	}
	return configurationPathFromOption(options.configFile, homeDirectory)
}

func configurationPathFromOption(configFile string, homeDirectory func() (string, error)) (string, error) {
	if configFile != "" {
		return configFile, nil
	}

	home, err := homeDirectory()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, defaultConfigFile), nil
}

type commandLineOptions struct {
	configFile string
	diagnostic bool
	help       bool
	version    bool
}

func parseCommandLineOptions(arguments []string) (commandLineOptions, error) {
	if len(arguments) == 1 && arguments[0] == "help" {
		return commandLineOptions{help: true}, nil
	}

	flags := flag.NewFlagSet("tp", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := commandLineOptions{}
	flags.StringVar(&options.configFile, "f", "", "configuration file path")
	flags.BoolVar(&options.diagnostic, "diagnostic", false, "list process states without starting the TUI")
	flags.BoolVar(&options.help, "h", false, "show this help message")
	flags.BoolVar(&options.help, "help", false, "show this help message")
	flags.BoolVar(&options.version, "v", false, "print version and exit")
	flags.BoolVar(&options.version, "version", false, "print version and exit")
	if err := flags.Parse(arguments); err != nil {
		return commandLineOptions{}, fmt.Errorf("parse command-line flags: %w", err)
	}
	if len(flags.Args()) != 0 {
		return commandLineOptions{}, fmt.Errorf("unexpected argument: %q", flags.Arg(0))
	}
	return options, nil
}

func writeUsage(output io.Writer) error {
	_, err := fmt.Fprint(output, usageText)
	return err
}
