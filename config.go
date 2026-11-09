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
	} else {
		config.SoftwareQualities = normalizeSoftwareQualities(config.SoftwareQualities)
	}
	if len(config.Severities) == 0 {
		config.Severities = defaultConfig.Severities
	} else {
		config.Severities = normalizeSeverities(config.Severities)
	}
	return config
}

func normalizeChoices(raw, allowed []string) []string {
	validChoices := []string{}
	seen := map[string]bool{}
	for _, item := range raw {
		normalized := strings.ToUpper(strings.TrimSpace(item))
		for _, valid := range allowed {
			if normalized == valid && !seen[normalized] {
				validChoices = append(validChoices, normalized)
				seen[normalized] = true
				break
			}
		}
	}
	if len(validChoices) == 0 {
		return allowed
	}
	return validChoices
}

func normalizeSeverities(raw []string) []string {
	return normalizeChoices(raw, defaultConfig.Severities)
}

func parseSeverities(input string) []string {
	if strings.TrimSpace(input) == "" {
		return defaultConfig.Severities
	}
	return normalizeSeverities(strings.Split(input, ","))
}

func normalizeSoftwareQualities(raw []string) []string {
	return normalizeChoices(raw, defaultConfig.SoftwareQualities)
}

func parseSoftwareQualities(input string) []string {
	if strings.TrimSpace(input) == "" {
		return defaultConfig.SoftwareQualities
	}
	return normalizeSoftwareQualities(strings.Split(input, ","))
}

func removeProjectFromSlice(projects []string, target string) ([]string, bool) {
	result := make([]string, 0, len(projects))
	removed := false
	for _, p := range projects {
		if p == target {
			removed = true
			continue
		}
		result = append(result, p)
	}
	return result, removed
}

func saveConfig(config Config) error {
	path := getConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
