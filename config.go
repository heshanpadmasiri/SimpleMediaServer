package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	MediaSource  string   `toml:"MediaSource"`  // Single media source (legacy)
	MediaSources []string `toml:"MediaSources"` // Multiple media sources (new)
	Port         int      `toml:"Port"`
}

func loadConfig() (*Config, error) {
	// First, try to load from current directory
	configPath := "config.toml"
	if _, err := os.Stat(configPath); err == nil {
		return loadConfigFromFile(configPath)
	}

	// If not found in current directory, try ~/.config/simpleMediaServer/config.toml
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath = filepath.Join(homeDir, ".config", "simpleMediaServer", "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		return loadConfigFromFile(configPath)
	}

	// If neither file exists, return an error
	return nil, fmt.Errorf("no configuration file found. Please create either:\n" +
		"1. config.toml in the current directory\n" +
		"2. ~/.config/simpleMediaServer/config.toml\n" +
		"See config.toml.example for a sample configuration")
}

func loadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Handle both single and multiple media sources
	var mediaPaths []string

	if len(config.MediaSources) > 0 {
		// Use MediaSources array if provided
		mediaPaths = config.MediaSources
	} else if config.MediaSource != "" {
		// Fall back to single MediaSource for backward compatibility
		mediaPaths = []string{config.MediaSource}
	} else {
		return nil, fmt.Errorf("either MediaSource or MediaSources is required in config file %s", path)
	}

	// Validate all paths exist
	for _, path := range mediaPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("media source path does not exist: %s", path)
		}
	}

	// Update the config with the resolved paths
	config.MediaSources = mediaPaths

	// Set default port if not set
	if config.Port == 0 {
		config.Port = 8080
		log.Println("Port not set in config, using default 8080")
	}

	return &config, nil
}
