// Package config defines Process Manager configuration types.
package config

import "github.com/pelletier/go-toml/v2"

// Configuration contains all processes managed by Process Manager.
type Configuration struct {
	Processes []Process `toml:"process"`
}

// Process describes one managed process from the configuration file.
type Process struct {
	Name    string
	Start   string
	ID      string
	Find    string
	Pattern string
	Stop    string
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
