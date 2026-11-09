package main

import (
	"encoding/csv"
	"encoding/json"
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

func TestFormatIssuesJSON_ProducesAgentReadablePayload(t *testing.T) {
	res := FetchResult{
		Total:     12500,
		Truncated: true,
		Issues: []Issue{
			{
				Key:             "AX-1",
				Rule:            "go:S100",
				Severity:        "BLOCKER",
				SoftwareQuality: "SECURITY",
				Component:       "my-proj:internal/auth/token.go",
				Line:            42,
				Message:         "Hardcoded secret",
				Status:          "OPEN",
				Effort:          "15min",
			},
			{
				Key:             "AX-2",
				Rule:            "go:S101",
				Severity:        "HIGH",
				SoftwareQuality: "RELIABILITY",
				Component:       "my-proj:main.go",
				Line:            12,
				Message:         "Nil dereference",
				Status:          "OPEN",
				Effort:          "5min",
			},
		},
	}

	raw, err := formatIssuesJSON(res, "my-proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed JSONReport
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}

	if parsed.Project != "my-proj" {
		t.Errorf("expected project 'my-proj', got %q", parsed.Project)
	}
	if !parsed.Truncated || parsed.Total != 12500 {
		t.Errorf("expected Truncated=true and Total=12500, got Truncated=%v Total=%d", parsed.Truncated, parsed.Total)
	}
	if parsed.Summary["BLOCKER"] != 1 || parsed.Summary["HIGH"] != 1 || parsed.Summary["MEDIUM"] != 0 || parsed.Summary["LOW"] != 0 {
		t.Errorf("unexpected summary counts: %+v", parsed.Summary)
	}
	if len(parsed.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(parsed.Issues))
	}
	if parsed.Issues[0].File != "internal/auth/token.go" {
		t.Errorf("expected prefix-stripped file 'internal/auth/token.go', got %q", parsed.Issues[0].File)
	}
	if parsed.Issues[0].SoftwareQuality != "SECURITY" || parsed.Issues[0].ImpactSeverity != "BLOCKER" {
		t.Errorf("unexpected quality/severity on issue 0: %+v", parsed.Issues[0])
	}
}


