package types

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config is the configuration for the interceptor binary.
type Config struct {
	GethEngineAddr string `json:"gethEngineAddr"`
	GethAuthSecret []byte `json:"gethAuthSecret"`

	EngineServerAddr string `json:"engineServerAddr"`
}

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
