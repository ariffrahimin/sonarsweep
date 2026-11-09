package main

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestView_StateError_RendersErrorOnce(t *testing.T) {
	m := model{
		state:    stateError,
		fetchErr: errors.New("connection refused"),
	}

	out := m.View()
	if count := strings.Count(out, "Error occurred"); count != 1 {
		t.Errorf("Expected 'Error occurred' to be rendered once, got %d times in output:\n%s", count, out)
	}
}

func TestUpdate_StateSeverities_ReplacesInsteadOfAppendingAndPreservesOrder(t *testing.T) {
	tmpDir := t.TempDir()
	originalConfigPath := configPath
	configPath = filepath.Join(tmpDir, "config.json")
	defer func() { configPath = originalConfigPath }()

	availableSeverities = []string{"BLOCKER", "HIGH", "MEDIUM", "LOW"}
	m := model{
		state:      stateSeverities,
		config:     defaultConfig,
		severities: []string{"BLOCKER", "HIGH", "MEDIUM", "LOW"}, // pre-populated by initialModel
		selectedSeverities: map[int]struct{}{
			1: {}, // HIGH
			3: {}, // LOW
		},
	}

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updatedModel.(model)

	want := []string{"HIGH", "LOW"}
	if !reflect.DeepEqual(got.severities, want) {
		t.Errorf("Expected severities %v, got %v", want, got.severities)
	}
	if !reflect.DeepEqual(got.config.Severities, want) {
		t.Errorf("Expected config.Severities %v, got %v", want, got.config.Severities)
	}
}

func TestUpdate_StateNewProject_DoesNotDuplicateRemoveProjectItem(t *testing.T) {
	tmpDir := t.TempDir()
	originalConfigPath := configPath
	configPath = filepath.Join(tmpDir, "config.json")
	defer func() { configPath = originalConfigPath }()

	cfg := defaultConfig
	cfg.SonarURL = "http://localhost:9000"
	cfg.Projects = []string{"proj-1"}
	_ = saveConfig(cfg)

	m := initialModel()
	m.token = "test-token"
	m.state = stateNewProject
	m.newProjectInput.SetValue("proj-2")

	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updatedModel.(model)

	removeCount := 0
	for _, it := range got.projectList.Items() {
		if it.(item) == "- Remove Project" {
			removeCount++
		}
	}
	if removeCount != 1 {
		t.Errorf("Expected '- Remove Project' to appear once in projectList, got %d (items: %v)", removeCount, got.projectList.Items())
	}
	if got.newProjectInput.Value() != "" {
		t.Errorf("Expected newProjectInput to be cleared after adding project, got %q", got.newProjectInput.Value())
	}
}

func TestUpdate_StateFetching_BuildsAndRendersSummaryTableWithBlocker(t *testing.T) {
	tmpDir := t.TempDir()
	originalExportPath := cliExportPath
	cliExportPath = filepath.Join(tmpDir, "out.csv")
	defer func() { cliExportPath = originalExportPath }()

	m := model{
		state:             stateFetching,
		projectKey:        "demo-project",
		softwareQualities: []string{"SECURITY", "RELIABILITY"},
	}

	issues := []Issue{
		{Key: "1", Severity: "BLOCKER"},
		{Key: "2", Severity: "HIGH"},
		{Key: "3", Severity: "HIGH"},
		{Key: "4", Severity: "MEDIUM"},
		{Key: "5", Severity: "LOW"},
	}

	updatedModel, _ := m.Update(issuesFetchedMsg{issues: issues})
	got := updatedModel.(model)

	out := got.View()
	if !strings.Contains(out, "BLOCKER") {
		t.Errorf("Expected summary view to contain 'BLOCKER' row, got:\n%s", out)
	}
	if !strings.Contains(out, "Total") {
		t.Errorf("Expected summary view to contain summary table with 'Total' row, got:\n%s", out)
	}
}

func TestUpdate_StateQualities_PersistsSoftwareQualitiesAndEscGoesBack(t *testing.T) {
	tmpDir := t.TempDir()
	originalConfigPath := configPath
	configPath = filepath.Join(tmpDir, "config.json")
	defer func() { configPath = originalConfigPath }()

	availableSoftwareQualities = []string{"RELIABILITY", "SECURITY", "MAINTAINABILITY"}
	availableSeverities = []string{"BLOCKER", "HIGH", "MEDIUM", "LOW"}

	m := model{
		state:  stateQualities,
		config: defaultConfig,
		selectedQualities: map[int]struct{}{
			1: {}, // SECURITY
		},
	}

	// Pressing Esc in stateQualities should go back to stateProject instead of quitting
	backModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Errorf("Expected Esc in stateQualities not to quit, got non-nil cmd")
	}
	if backModel.(model).state != stateProject {
		t.Errorf("Expected stateProject after Esc in stateQualities, got %v", backModel.(model).state)
	}

	// Pressing Enter in stateQualities should persist config.SoftwareQualities
	updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updatedModel.(model)
	want := []string{"SECURITY"}
	if !reflect.DeepEqual(got.config.SoftwareQualities, want) {
		t.Errorf("Expected config.SoftwareQualities %v, got %v", want, got.config.SoftwareQualities)
	}
	loaded := loadConfig()
	if !reflect.DeepEqual(loaded.SoftwareQualities, want) {
		t.Errorf("Expected persisted SoftwareQualities %v, got %v", want, loaded.SoftwareQualities)
	}
}

func TestUpdate_StateFetching_ShowsTruncationWarningAndResetsOnRepeat(t *testing.T) {
	tmpDir := t.TempDir()
	originalExportPath := cliExportPath
	cliExportPath = filepath.Join(tmpDir, "out.csv")
	defer func() { cliExportPath = originalExportPath }()

	cfg := defaultConfig
	cfg.Severities = []string{"BLOCKER", "HIGH"}

	m := model{
		state:             stateFetching,
		config:            cfg,
		projectKey:        "huge-project",
		softwareQualities: []string{"SECURITY"},
	}

	updatedModel, _ := m.Update(issuesFetchedMsg{
		issues:    []Issue{{Key: "1", Severity: "BLOCKER"}},
		total:     12500,
		truncated: true,
	})
	got := updatedModel.(model)

	out := got.View()
	if !strings.Contains(out, "10,000") || !strings.Contains(out, "12500") {
		t.Errorf("Expected truncation warning mentioning 10,000 and 12500 in view, got:\n%s", out)
	}

	// Press Enter on "Export another project" (promptCursor == 0)
	resetModel, _ := got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	afterReset := resetModel.(model)
	if !reflect.DeepEqual(afterReset.severities, cfg.Severities) {
		t.Errorf("Expected severities to reset to config.Severities %v, got %v", cfg.Severities, afterReset.severities)
	}
	if afterReset.truncated {
		t.Errorf("Expected truncated flag to be cleared after reset")
	}
}




