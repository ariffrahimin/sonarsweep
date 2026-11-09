package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeMCP_InitializeListAndCallTools(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/issues/search":
			_ = json.NewEncoder(w).Encode(Response{
				Paging: Paging{Total: 1},
				Issues: []Issue{
					{
						Key:       "MCP-ISSUE-1",
						Rule:      "go:S100",
						Component: "mcp-proj:cmd/app.go",
						Line:      19,
						Message:   "Fix naming",
						Status:    "OPEN",
						Impacts: []struct {
							SoftwareQuality string `json:"softwareQuality"`
							Severity        string `json:"severity"`
						}{
							{SoftwareQuality: "MAINTAINABILITY", Severity: "HIGH"},
						},
					},
				},
			})
		case "/api/rules/show":
			_, _ = w.Write([]byte(`{"rule":{"key":"go:S100","name":"Naming convention","mdDesc":"Use MixedCaps."}}`))
		}
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	originalConfigPath := configPath
	configPath = filepath.Join(tmpDir, "config.json")
	defer func() { configPath = originalConfigPath }()

	t.Setenv("USER_TOKEN", "mcp-test-token")

	_ = saveConfig(Config{
		SonarURL:          srv.URL,
		Projects:          []string{"mcp-proj"},
		SoftwareQualities: []string{"MAINTAINABILITY"},
		Severities:        []string{"HIGH"},
	})

	requests := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"sonarsweep_list_projects","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"sonarsweep_fetch_issues","arguments":{"project":"mcp-proj","severity":"HIGH"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"sonarsweep_explain_rule","arguments":{"rule":"go:S100"}}}`,
	}

	in := strings.NewReader(strings.Join(requests, "\n") + "\n")
	var out bytes.Buffer

	if err := serveMCP(in, &out); err != nil {
		t.Fatalf("serveMCP returned unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 JSON-RPC responses (notification produces none), got %d:\n%s", len(lines), out.String())
	}

	if !strings.Contains(lines[1], "sonarsweep_fetch_issues") || !strings.Contains(lines[1], "sonarsweep_explain_rule") {
		t.Errorf("expected tools/list response to contain tool names, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "mcp-proj") {
		t.Errorf("expected sonarsweep_list_projects response to contain 'mcp-proj', got: %s", lines[2])
	}
	if !strings.Contains(lines[3], "MCP-ISSUE-1") || !strings.Contains(lines[3], "cmd/app.go") {
		t.Errorf("expected sonarsweep_fetch_issues response to contain 'MCP-ISSUE-1' and stripped file path, got: %s", lines[3])
	}
	if !strings.Contains(lines[4], "Use MixedCaps.") {
		t.Errorf("expected sonarsweep_explain_rule response to contain rule description, got: %s", lines[4])
	}
}
