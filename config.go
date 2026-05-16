package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var configPath string

type Config struct {
	SonarURL          string   `json:"sonarqube_url"`
	Projects          []string `json:"projects"`
	SoftwareQualities []string `json:"software_qualities"`
}

var defaultConfig = Config{
	SonarURL: "",
	Projects: []string{},
	SoftwareQualities: []string{
		"RELIABILITY",
		"SECURITY",
		"MAINTAINABILITY",
	},
}

func getConfigPath() string {
	if configPath != "" {
		return configPath
	}
	home, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(home, ".config", "sonarsweep")
		os.MkdirAll(configDir, 0700)
		return filepath.Join(configDir, "config.json")
	}
	return "sonarsweep.json"
}

func loadConfig() Config {
	path := getConfigPath()
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig
		}
		return defaultConfig
	}
	defer file.Close()
	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return defaultConfig
	}
	config.SonarURL = strings.TrimRight(config.SonarURL, "/")
	if len(config.SoftwareQualities) == 0 {
		config.SoftwareQualities = defaultConfig.SoftwareQualities
	}
	return config
}

func saveConfig(config Config) error {
	path := getConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
