package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpToolResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

func resolveToken() string {
	_ = godotenv.Load()
	token := os.Getenv("USER_TOKEN")
	if token == "" {
		token, _ = GetToken()
	}
	return token
}

func resolveSonarContext(explicitProject string) (baseURL, projectKey string, cfg Config) {
	cfg = loadConfig()
	cwdProj, cwdHost := detectSonarProperties(".")

	baseURL = cfg.SonarURL
	if baseURL == "" {
		baseURL = cwdHost
	}

	projectKey = strings.TrimSpace(explicitProject)
	if projectKey == "" {
		projectKey = cwdProj
	}
	if projectKey == "" && len(cfg.Projects) > 0 {
		projectKey = cfg.Projects[0]
	}
	return baseURL, projectKey, cfg
}

func serveMCP(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	enc := json.NewEncoder(out)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		// JSON-RPC notifications have no ID and expect no response
		if len(req.ID) == 0 || string(req.ID) == "null" {
			continue
		}

		switch req.Method {
		case "initialize":
			_ = enc.Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{},
					},
					"serverInfo": map[string]interface{}{
						"name":    "sonarsweep",
						"version": VERSION,
					},
				},
			})

		case "ping":
			_ = enc.Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  map[string]interface{}{},
			})

		case "tools/list":
			tools := []mcpTool{
				{
					Name:        "sonarsweep_list_projects",
					Description: "List configured SonarQube projects, server URL, and any project key auto-detected from sonar-project.properties in the current repository.",
					InputSchema: map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
				{
					Name:        "sonarsweep_fetch_issues",
					Description: "Fetch open SonarQube issues for a project without needing sonar-scanner CLI. Returns repo-relative file paths, line numbers, impact severities, software qualities, and rule keys.",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"project": map[string]interface{}{
								"type":        "string",
								"description": "SonarQube project key (optional if sonar-project.properties or saved projects exist)",
							},
							"severity": map[string]interface{}{
								"type":        "string",
								"description": "Comma-separated severities: BLOCKER,HIGH,MEDIUM,LOW",
							},
							"quality": map[string]interface{}{
								"type":        "string",
								"description": "Comma-separated software qualities: SECURITY,RELIABILITY,MAINTAINABILITY",
							},
							"branch": map[string]interface{}{
								"type":        "string",
								"description": "Branch name to query (optional)",
							},
							"pr": map[string]interface{}{
								"type":        "string",
								"description": "Pull request ID to query (optional)",
							},
							"overall": map[string]interface{}{
								"type":        "boolean",
								"description": "If true, fetch issues across overall code instead of new code period only",
							},
						},
					},
				},
				{
					Name:        "sonarsweep_explain_rule",
					Description: "Fetch the SonarQube rule description, root cause, and remediation instructions for a given rule key (e.g. 'go:S100').",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"rule": map[string]interface{}{
								"type":        "string",
								"description": "SonarQube rule key (e.g. 'go:S100', 'typescript:S6759')",
							},
						},
						"required": []string{"rule"},
					},
				},
			}
			_ = enc.Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"tools": tools,
				},
			})

		case "tools/call":
			var callParams struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &callParams); err != nil {
				_ = enc.Encode(jsonRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &jsonRPCError{Code: -32602, Message: "Invalid tool call params"},
				})
				continue
			}

			result := handleMCPToolCall(callParams.Name, callParams.Arguments)
			_ = enc.Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  result,
			})

		default:
			_ = enc.Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &jsonRPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
			})
		}
	}

	return scanner.Err()
}

func handleMCPToolCall(name string, rawArgs json.RawMessage) mcpToolResult {
	switch name {
	case "sonarsweep_list_projects":
		baseURL, detectedProj, cfg := resolveSonarContext("")
		cwdProj, _ := detectSonarProperties(".")
		payload, _ := json.MarshalIndent(map[string]interface{}{
			"sonarqube_url":         baseURL,
			"configured_projects":   cfg.Projects,
			"detected_repo_project": cwdProj,
			"default_project":       detectedProj,
		}, "", "  ")
		return mcpToolResult{Content: []mcpContent{{Type: "text", Text: string(payload)}}}

	case "sonarsweep_fetch_issues":
		var args struct {
			Project  string `json:"project"`
			Severity string `json:"severity"`
			Quality  string `json:"quality"`
			Branch   string `json:"branch"`
			PR       string `json:"pr"`
			Overall  bool   `json:"overall"`
		}
		if len(rawArgs) > 0 {
			_ = json.Unmarshal(rawArgs, &args)
		}

		baseURL, projectKey, cfg := resolveSonarContext(args.Project)
		if baseURL == "" {
			return mcpToolResult{
				IsError: true,
				Content: []mcpContent{{Type: "text", Text: "SonarQube URL is not configured. Run 'sonarsweep --set-url <url>' or add sonar.host.url to sonar-project.properties."}},
			}
		}
		if projectKey == "" {
			return mcpToolResult{
				IsError: true,
				Content: []mcpContent{{Type: "text", Text: "No project key specified or detected. Pass 'project' or add sonar.projectKey to sonar-project.properties."}},
			}
		}
		token := resolveToken()
		if token == "" {
			return mcpToolResult{
				IsError: true,
				Content: []mcpContent{{Type: "text", Text: "Authentication token missing. Run 'sonarsweep --set-token <token>' or set USER_TOKEN."}},
			}
		}

		severities := cfg.Severities
		if args.Severity != "" {
			severities = parseSeverities(args.Severity)
		}
		qualities := cfg.SoftwareQualities
		if args.Quality != "" {
			qualities = parseSoftwareQualities(args.Quality)
		}

		res, err := fetchIssuesWithOptions(FetchOptions{
			BaseURL:           baseURL,
			ProjectKey:        projectKey,
			Token:             token,
			SoftwareQualities: qualities,
			Severities:        severities,
			IsNewCodePeriod:   !args.Overall,
			Branch:            args.Branch,
			PullRequest:       args.PR,
		})
		if err != nil {
			return mcpToolResult{
				IsError: true,
				Content: []mcpContent{{Type: "text", Text: err.Error()}},
			}
		}

		formatted, err := formatIssuesJSON(res, projectKey)
		if err != nil {
			return mcpToolResult{
				IsError: true,
				Content: []mcpContent{{Type: "text", Text: err.Error()}},
			}
		}
		return mcpToolResult{Content: []mcpContent{{Type: "text", Text: string(formatted)}}}

	case "sonarsweep_explain_rule":
		var args struct {
			Rule string `json:"rule"`
		}
		if len(rawArgs) > 0 {
			_ = json.Unmarshal(rawArgs, &args)
		}
		if strings.TrimSpace(args.Rule) == "" {
			return mcpToolResult{
				IsError: true,
				Content: []mcpContent{{Type: "text", Text: "Missing required argument: 'rule'"}},
			}
		}

		baseURL, _, _ := resolveSonarContext("")
		if baseURL == "" {
			return mcpToolResult{
				IsError: true,
				Content: []mcpContent{{Type: "text", Text: "SonarQube URL is not configured."}},
			}
		}
		token := resolveToken()
		if token == "" {
			return mcpToolResult{
				IsError: true,
				Content: []mcpContent{{Type: "text", Text: "Authentication token missing."}},
			}
		}

		detail, err := fetchRuleDetails(baseURL, token, strings.TrimSpace(args.Rule))
		if err != nil {
			return mcpToolResult{
				IsError: true,
				Content: []mcpContent{{Type: "text", Text: err.Error()}},
			}
		}
		raw, _ := json.MarshalIndent(detail, "", "  ")
		return mcpToolResult{Content: []mcpContent{{Type: "text", Text: string(raw)}}}

	default:
		return mcpToolResult{
			IsError: true,
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Unknown tool: %s", name)}},
		}
	}
}
