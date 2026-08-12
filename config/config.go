// Package config defines Process Manager configuration types.
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Configuration contains all processes managed by Process Manager.
type Configuration struct {
	Processes []Process `toml:"process"`
}

// Process describes one managed process from the configuration file.
type Process struct {
	Name    string `toml:"name"`
	Start   string `toml:"start"`
	ID      string `toml:"id"`
	Find    string `toml:"find"`
	Pattern string `toml:"pattern"`
	Stop    string `toml:"stop"`
}

// Decoder unmarshals configuration data.
type Decoder interface {
	Unmarshal(data []byte, destination any) error
}

// TOMLDecoder unmarshals TOML configuration data.
type TOMLDecoder struct{}

// Unmarshal decodes TOML data into destination.
func (TOMLDecoder) Unmarshal(data []byte, destination any) error {
	return toml.Unmarshal(data, destination)
}

// FileReader reads configuration data from a file.
type FileReader interface {
	ReadFile(name string) ([]byte, error)
}

// Load reads, decodes, and validates a configuration file.
func Load(path string, reader FileReader, decoder Decoder) (Configuration, error) {
	data, err := reader.ReadFile(path)
	if err != nil {
		return Configuration{}, fmt.Errorf("read configuration %q: %w", path, err)
	}

	var configuration Configuration
	if err := decoder.Unmarshal(data, &configuration); err != nil {
		return Configuration{}, fmt.Errorf("parse configuration %q: %w", path, err)
	}
	if err := configuration.Validate(); err != nil {
		return Configuration{}, fmt.Errorf("invalid configuration %q: %w", path, err)
	}
	return configuration, nil
}

// Validate ensures every configured process can be managed safely.
func (configuration Configuration) Validate() error {
	if len(configuration.Processes) == 0 {
		return errors.New("no process entries")
	}
	for index, process := range configuration.Processes {
		if strings.TrimSpace(process.Name) == "" {
			return fmt.Errorf("process %d: missing name", index+1)
		}
		if strings.TrimSpace(process.Start) == "" {
			return fmt.Errorf("process %d: missing start", index+1)
		}
		if strings.TrimSpace(process.Find) != "" && strings.TrimSpace(process.Stop) == "" {
			return fmt.Errorf("process %d: find requires stop", index+1)
		}
	}
	return nil
}
