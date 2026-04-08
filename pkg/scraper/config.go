package scraper

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadConfig reads and parses a mapping.json configuration file.
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}

	if len(config.Organizations) == 0 {
		return nil, fmt.Errorf("config contains no organizations")
	}

	return &config, nil
}
