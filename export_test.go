package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func TestExportToCSV_CreatesFileAndHandlesErrors(t *testing.T) {
	tmpDir := t.TempDir()
	tmpExportFile := filepath.Join(tmpDir, "custom_export.csv")

	// Temporarily override cliExportPath
	cliExportPath = tmpExportFile
	defer func() { cliExportPath = "" }()

	issues := []Issue{
		{
			Key:             "ISSUE-1",
			Rule:            "rule-1",
			Severity:        "HIGH",
			SoftwareQuality: "SECURITY",
			Component:       "project:src/main.go",
			Line:            10,
			Message:         "Bad code",
			Status:          "OPEN",
			Effort:          "10min",
		},
	}

	savedFile, err := exportToCSV(issues, "project")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if savedFile != tmpExportFile {
		t.Errorf("Expected savedFile to be %s, got %s", tmpExportFile, savedFile)
	}

	// Verify file was actually created and contains data
	file, err := os.Open(tmpExportFile)
	if err != nil {
		t.Fatalf("Expected exported file to exist, got error: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 rows (header + 1 issue), got %d", len(records))
	}

	if records[1][0] != "ISSUE-1" {
		t.Errorf("Expected first column of data to be 'ISSUE-1', got %s", records[1][0])
	}
	if records[0][3] != "software_quality" || records[1][3] != "SECURITY" {
		t.Errorf("Expected column 3 to be software_quality='SECURITY', got header=%q val=%q", records[0][3], records[1][3])
	}
}

func TestExportToCSV_SanitizesProjectKeyDirectoryName(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)

	originalExportPath := cliExportPath
	cliExportPath = ""
	defer func() { cliExportPath = originalExportPath }()

	issues := []Issue{
		{
			Key:       "ISSUE-2",
			Rule:      "go:S100",
			Severity:  "BLOCKER",
			Component: "my-org:my/project:pkg/service.go",
			Line:      25,
			Message:   "Fix me",
			Status:    "OPEN",
		},
	}

	savedFile, err := exportToCSV(issues, "my-org:my/project")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedDir := filepath.Join(tmpHome, "Downloads", "my-org_my_project")
	if filepath.Dir(savedFile) != expectedDir {
		t.Errorf("Expected output directory %s, got %s", expectedDir, filepath.Dir(savedFile))
	}
}

