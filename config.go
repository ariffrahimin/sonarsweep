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
	Severities        []string `json:"severities"`
}

var defaultConfig = Config{
	SonarURL: "",
	Projects: []string{},
	SoftwareQualities: []string{
		"RELIABILITY",
		"SECURITY",
		"MAINTAINABILITY",
	},
	Severities: []string{
		"BLOCKER",
		"HIGH",
		"MEDIUM",
		"LOW",
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
	if len(config.Severities) == 0 {
		config.Severities = defaultConfig.Severities
	} else {
		config.Severities = normalizeSeverities(config.Severities)
	}
	return config
}

func normalizeSeverities(raw []string) []string {
	validSeverities := []string{}
	seen := map[string]bool{}
	for _, s := range raw {
		normalized := strings.ToUpper(strings.TrimSpace(s))
		for _, valid := range defaultConfig.Severities {
			if normalized == valid && !seen[normalized] {
				validSeverities = append(validSeverities, normalized)
				seen[normalized] = true
				break
			}
		}
	}
	if len(validSeverities) == 0 {
		return defaultConfig.Severities
	}
	return validSeverities
}

func parseSeverities(input string) []string {
	if strings.TrimSpace(input) == "" {
		return defaultConfig.Severities
	}
	return normalizeSeverities(strings.Split(input, ","))
}

func saveConfig(config Config) error {
	path := getConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
