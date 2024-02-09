package types

import (
	"encoding/json"
	"fmt"
	"os"
)

const DefaultConfigFilePath = "config.json"

// Config is the configuration for the interceptor binary.
type Config struct {
	// Accepted log levels are: "trace", "debug", "info", "warn", "error", "crit"
	LogLevel string `json:"logLevel"`

	GethEngineAddr string `json:"gethEngineAddr"`
	GethAuthSecret []byte `json:"gethAuthSecret"`

	EngineServerAddr string `json:"engineServerAddr"`
}

// ConfigFromFilePath reads a Config from a file.
func ConfigFromFilePath(filePath string) (*Config, error) {
	configFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config file: %w", err)
	}

	return &config, nil
}

// GetLogger returns a logger with the log level specified in the config and the given keyvals.
func (c *Config) GetLogger(keyvals ...any) (CompositeLogger, error) {
	return NewCompositeLogger(c.LogLevel, keyvals...)
}
