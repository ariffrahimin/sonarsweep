package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var cliExportPath string

type JSONIssue struct {
	Key             string `json:"key"`
	Rule            string `json:"rule"`
	ImpactSeverity  string `json:"impact_severity"`
	SoftwareQuality string `json:"software_quality"`
	File            string `json:"file"`
	Line            int    `json:"line,omitempty"`
	Message         string `json:"message"`
	Status          string `json:"status"`
	Effort          string `json:"effort,omitempty"`
	Author          string `json:"author,omitempty"`
	CreationDate    string `json:"creationDate,omitempty"`
}

type JSONReport struct {
	Project   string         `json:"project"`
	Total     int            `json:"total"`
	Truncated bool           `json:"truncated"`
	Summary   map[string]int `json:"summary"`
	Issues    []JSONIssue    `json:"issues"`
}

func formatIssuesJSON(res FetchResult, projectKey string) ([]byte, error) {
	summary := map[string]int{
		"BLOCKER": 0,
		"HIGH":    0,
		"MEDIUM":  0,
		"LOW":     0,
	}
	prefix := projectKey + ":"
	items := make([]JSONIssue, 0, len(res.Issues))
	for _, issue := range res.Issues {
		summary[issue.Severity]++
		items = append(items, JSONIssue{
			Key:             issue.Key,
			Rule:            issue.Rule,
			ImpactSeverity:  issue.Severity,
			SoftwareQuality: issue.SoftwareQuality,
			File:            strings.TrimPrefix(issue.Component, prefix),
			Line:            issue.Line,
			Message:         issue.Message,
			Status:          issue.Status,
			Effort:          issue.Effort,
			Author:          issue.Author,
			CreationDate:    issue.CreationDate,
		})
	}
	total := res.Total
	if total == 0 {
		total = len(res.Issues)
	}
	report := JSONReport{
		Project:   projectKey,
		Total:     total,
		Truncated: res.Truncated,
		Summary:   summary,
		Issues:    items,
	}
	return json.MarshalIndent(report, "", "  ")
}

var invalidDirChars = strings.NewReplacer(
	":", "_",
	"/", "_",
	"\\", "_",
	"*", "_",
	"?", "_",
	"\"", "_",
	"<", "_",
	">", "_",
	"|", "_",
)

func sanitizeDirName(name string) string {
	cleaned := invalidDirChars.Replace(strings.TrimSpace(name))
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return "sonarsweep_export"
	}
	return cleaned
}

func exportToCSV(issues []Issue, projectKey string) (string, error) {
	dateStr := time.Now().Format("20060102_150405")

	var outputFilename string
	if cliExportPath != "" {
		outputFilename = cliExportPath
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}

		downloadsDir := filepath.Join(homeDir, "Downloads")
		projectDir := filepath.Join(downloadsDir, sanitizeDirName(projectKey))

		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
		outputFilename = filepath.Join(projectDir, fmt.Sprintf("sonarqube_issues_%s.csv", dateStr))
	}

	file, err := os.Create(outputFilename)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)

	fields := []string{
		"key", "rule", "impact_severity", "software_quality", "component", "line",
		"message", "status", "effort", "author", "creationDate",
	}
	if err := writer.Write(fields); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	prefix := projectKey + ":"

	for _, issue := range issues {
		component := strings.TrimPrefix(issue.Component, prefix)

		lineStr := ""
		if issue.Line != 0 {
			lineStr = strconv.Itoa(issue.Line)
		}

		row := []string{
			issue.Key,
			issue.Rule,
			issue.Severity,
			issue.SoftwareQuality,
			component,
			lineStr,
			issue.Message,
			issue.Status,
			issue.Effort,
			issue.Author,
			issue.CreationDate,
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("failed to flush CSV writer: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("failed to close CSV file: %w", err)
	}

	return outputFilename, nil
}
