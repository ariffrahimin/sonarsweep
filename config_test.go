package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FileNotFound(t *testing.T) {
	// Save original configPath and restore after test
	originalConfigPath := configPath
	configPath = "nonexistent_config_file_12345.json"
	defer func() { configPath = originalConfigPath }()

	cfg := loadConfig()

	if cfg.SonarURL != "" {
		t.Errorf("Expected empty SonarURL for non-existent file, got %s", cfg.SonarURL)
	}
	if len(cfg.Projects) != 0 {
		t.Errorf("Expected empty Projects for non-existent file, got %d", len(cfg.Projects))
	}
	if len(cfg.SoftwareQualities) != 3 {
		t.Errorf("Expected 3 default SoftwareQualities, got %d", len(cfg.SoftwareQualities))
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")

	testConfig := Config{
		SonarURL:          "http://test.example.com:9000",
		Projects:          []string{"test-project-1", "test-project-2"},
		SoftwareQualities: []string{"RELIABILITY", "SECURITY"},
	}

	data, _ := json.Marshal(testConfig)
	os.WriteFile(tmpFile, data, 0644)

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	cfg := loadConfig()

	if cfg.SonarURL != testConfig.SonarURL {
		t.Errorf("Expected SonarURL %s, got %s", testConfig.SonarURL, cfg.SonarURL)
	}
	if len(cfg.Projects) != len(testConfig.Projects) {
		t.Errorf("Expected %d projects, got %d", len(testConfig.Projects), len(cfg.Projects))
	}
	if cfg.Projects[0] != "test-project-1" || cfg.Projects[1] != "test-project-2" {
		t.Errorf("Projects not parsed correctly, got %v", cfg.Projects)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")

	os.WriteFile(tmpFile, []byte("this is not valid json{"), 0644)

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	cfg := loadConfig()

	if cfg.SonarURL != "" || len(cfg.Projects) != 0 {
		t.Errorf("Expected default config for invalid JSON, got %+v", cfg)
	}
}

func TestSaveConfig_AndLoadConfig_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	original := Config{
		SonarURL:          "http://roundtrip.test:9000",
		Projects:          []string{"project-a", "project-b", "project-c"},
		SoftwareQualities: []string{"MAINTAINABILITY"},
	}

	_ = saveConfig(original)

	loaded := loadConfig()

	if loaded.SonarURL != original.SonarURL {
		t.Errorf("SonarURL mismatch after round-trip: got %s, want %s", loaded.SonarURL, original.SonarURL)
	}
	if len(loaded.Projects) != len(original.Projects) {
		t.Errorf("Projects count mismatch after round-trip: got %d, want %d", len(loaded.Projects), len(original.Projects))
	}
	if loaded.Projects[0] != "project-a" || loaded.Projects[2] != "project-c" {
		t.Errorf("Projects content mismatch after round-trip: got %v", loaded.Projects)
	}
	if loaded.SoftwareQualities[0] != "MAINTAINABILITY" {
		t.Errorf("SoftwareQualities mismatch after round-trip: got %v", loaded.SoftwareQualities)
	}
}

func TestSaveConfig_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "new_config.json")

	originalConfigPath := configPath
	configPath = tmpFile
	defer func() { configPath = originalConfigPath }()

	cfg := Config{
		SonarURL:          "http://createtest.com:9000",
		Projects:          []string{"auto-created-project"},
		SoftwareQualities: []string{"RELIABILITY", "SECURITY", "MAINTAINABILITY"},
	}

	_ = saveConfig(cfg)

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Errorf("Expected config file to be created at %s", tmpFile)
	}
}

func TestDefaultConfig_HasCorrectSoftwareQualities(t *testing.T) {
	if len(defaultConfig.SoftwareQualities) != 3 {
		t.Errorf("Expected 3 default software qualities, got %d", len(defaultConfig.SoftwareQualities))
	}

	expected := []string{"RELIABILITY", "SECURITY", "MAINTAINABILITY"}
	for i, sq := range expected {
		if defaultConfig.SoftwareQualities[i] != sq {
			t.Errorf("Expected SoftwareQuality[%d] to be %s, got %s", i, sq, defaultConfig.SoftwareQualities[i])
		}
	}
}

func TestLoadConfig_StripsTrailingSlash(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "sonarsweep.json")
	configPath = tmpFile
	defer func() { configPath = "" }()

	jsonContent := `{
		"sonarqube_url": "http://example.com:9000/"
	}`
	if err := os.WriteFile(tmpFile, []byte(jsonContent), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg := loadConfig()

	if cfg.SonarURL != "http://example.com:9000" {
		t.Errorf("Expected trailing slash to be stripped, got %s", cfg.SonarURL)
	}
}
