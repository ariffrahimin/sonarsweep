package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var cliExportPath string

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
