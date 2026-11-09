package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

var VERSION = "dev"

func printHelp() {
	helpText := `SonarSweep v%s - SonarQube Issue Exporter

USAGE:
  sonarsweep [flags]

FLAGS:
  -h, --help                Show this help message and exit
  -v, --version             Show version information and exit
  --reset                   Reset the saved URL, Projects, and Keychain Token
  -c, --config <path>       Use a different configuration file
  --view-config             Print current configuration and exit
  --list-projects           List all saved projects and exit
  --add-project <key>       Add a project to the configuration
  --remove-project <key>    Remove a project from the configuration
  --set-url <url>           Set or update the SonarQube URL in configuration
  --set-token <token>       Save or update the SonarQube token in OS keychain
  -p, --project <key>       Select project key for headless run
  --export <path>           Override the CSV export path
  --dry-run                 Fetch issues but skip CSV export
  -q, --quiet               Run in headless mode (no TUI)
  -s, --severity <list>     Filter by severities (BLOCKER,HIGH,MEDIUM,LOW)
  --quality <list>          Filter by software qualities (RELIABILITY,SECURITY,MAINTAINABILITY)
  -b, --branch <name>       Filter issues by SonarQube branch
  --pr <id>                 Filter issues by SonarQube pull request ID
  --overall                 Fetch issues from overall code (default: new code only)
  --json                    Output issues or rule details as JSON to stdout (AI/agent mode)
  --explain-rule <key>      Fetch and display SonarQube rule description and remediation
  --mcp                     Run as a Model Context Protocol (MCP) stdio server for AI agents

EXAMPLES:
  sonarsweep --help
  sonarsweep --version
  sonarsweep --reset
  sonarsweep --list-projects
  sonarsweep --add-project my-new-project
  sonarsweep --export /tmp/output.csv
  sonarsweep --dry-run
  sonarsweep --quiet

CONFIGURATION:
  Configuration is stored in '~/.config/sonarsweep/config.json' (fallback: 'sonarsweep.json'). You can edit this file
  directly or use the --add-project flag to add projects.

  The token (USER_TOKEN) is managed securely via your operating system's native keychain. It can also be provided via a .env file or entered securely in the TUI prompt.

  Default config location: ~/.config/sonarsweep/config.json
  Override with: --config /path/to/config.json
`
	fmt.Printf(helpText, VERSION)
}

func main() {
	initDebug()
	defer closeDebug()

	var (
		help          bool
		version       bool
		reset         bool
		viewConfig    bool
		listProjects  bool
		addProject    string
		removeProject string
		setURL        string
		setToken      string
		project       string
		exportPath    string
		dryRun        bool
		quiet         bool
		severities    string
		qualities     string
		branch        string
		pullRequest   string
		overall       bool
		jsonOutput    bool
		explainRule   string
		mcpMode       bool
	)

	flag.BoolVar(&help, "help", false, "Show this help message and exit")
	flag.BoolVar(&help, "h", false, "Show this help message and exit (shorthand)")
	flag.BoolVar(&version, "version", false, "Show version information and exit")
	flag.BoolVar(&version, "v", false, "Show version information and exit (shorthand)")
	flag.BoolVar(&reset, "reset", false, "Reset the saved URL, Projects, and Keychain Token")
	flag.BoolVar(&viewConfig, "view-config", false, "Print current configuration and exit")
	flag.BoolVar(&listProjects, "list-projects", false, "List all saved projects and exit")
	flag.StringVar(&addProject, "add-project", "", "Add a project to the configuration")
	flag.StringVar(&removeProject, "remove-project", "", "Remove a project from the configuration")
	flag.StringVar(&setURL, "set-url", "", "Set or update the SonarQube URL in configuration")
	flag.StringVar(&setToken, "set-token", "", "Save or update the SonarQube token in OS keychain")
	flag.StringVar(&project, "project", "", "Select project key for headless run")
	flag.StringVar(&project, "p", "", "Select project key for headless run (shorthand)")
	flag.StringVar(&exportPath, "export", "", "Override the CSV export path")
	flag.BoolVar(&dryRun, "dry-run", false, "Fetch issues but skip CSV export")
	flag.BoolVar(&quiet, "quiet", false, "Run in headless mode (no TUI)")
	flag.BoolVar(&quiet, "q", false, "Run in headless mode (no TUI) (shorthand)")
	flag.StringVar(&configPath, "config", "", "Use a different configuration file")
	flag.StringVar(&configPath, "c", "", "Use a different configuration file (shorthand)")
	flag.StringVar(&severities, "severity", "", "Filter by impact severities (comma-separated: BLOCKER,HIGH,MEDIUM,LOW)")
	flag.StringVar(&severities, "s", "", "Filter by impact severities (comma-separated: BLOCKER,HIGH,MEDIUM,LOW) (shorthand)")
	flag.StringVar(&qualities, "quality", "", "Filter by software qualities (comma-separated: RELIABILITY,SECURITY,MAINTAINABILITY)")
	flag.StringVar(&branch, "branch", "", "Filter issues by SonarQube branch")
	flag.StringVar(&branch, "b", "", "Filter issues by SonarQube branch (shorthand)")
	flag.StringVar(&pullRequest, "pr", "", "Filter issues by SonarQube pull request ID")
	flag.BoolVar(&overall, "overall", false, "Fetch issues from overall code instead of new code period")
	flag.BoolVar(&jsonOutput, "json", false, "Output issues or rule details as JSON to stdout")
	flag.StringVar(&explainRule, "explain-rule", "", "Fetch and display SonarQube rule description and remediation")
	flag.BoolVar(&mcpMode, "mcp", false, "Run as a Model Context Protocol (MCP) stdio server")

	flag.Usage = printHelp
	flag.Parse()

	if mcpMode {
		if err := serveMCP(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if help {
		printHelp()
		os.Exit(0)
	}

	if version {
		fmt.Printf("SonarSweep v%s\n", VERSION)
		os.Exit(0)
	}

	if reset {
		cfg := defaultConfig
		if err := saveConfig(cfg); err != nil {
			fmt.Printf("Failed to reset config: %v\n", err)
			os.Exit(1)
		}
		_ = DeleteToken()
		fmt.Println("Configuration reset successfully.")
		os.Exit(0)
	}

	if viewConfig {
		cfg := loadConfig()
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(data))
		os.Exit(0)
	}

	if listProjects {
		cfg := loadConfig()
		if len(cfg.Projects) == 0 {
			fmt.Println("No projects saved. Run the TUI to add projects.")
		} else {
			for _, p := range cfg.Projects {
				fmt.Println(p)
			}
		}
		os.Exit(0)
	}

	isHeadless := quiet || dryRun || exportPath != "" || project != "" || branch != "" || pullRequest != "" || jsonOutput || explainRule != ""

	if setURL != "" {
		trimmedURL := strings.TrimRight(strings.TrimSpace(setURL), "/")
		if !isValidURL(trimmedURL) {
			fmt.Fprintln(os.Stderr, "Error: Invalid URL. Must start with http:// or https:// and have a valid host.")
			os.Exit(1)
		}
		cfg := loadConfig()
		cfg.SonarURL = trimmedURL
		if err := saveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
			os.Exit(1)
		}
		if !quiet && !isHeadless {
			fmt.Printf("SonarQube URL updated to '%s'.\n", trimmedURL)
		}
		if !isHeadless && addProject == "" && removeProject == "" && setToken == "" {
			os.Exit(0)
		}
	}

	if setToken != "" {
		trimmedToken := strings.TrimSpace(setToken)
		if err := StoreToken(trimmedToken); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to store token in keychain: %v\n", err)
			os.Exit(1)
		}
		if !quiet && !isHeadless {
			fmt.Println("Authentication token updated in OS keychain.")
		}
		if !isHeadless && addProject == "" && removeProject == "" {
			os.Exit(0)
		}
	}

	if removeProject != "" {
		cfg := loadConfig()
		updated, removed := removeProjectFromSlice(cfg.Projects, removeProject)
		if !removed {
			if !quiet {
				fmt.Printf("Project '%s' not found in configuration.\n", removeProject)
			}
			os.Exit(1)
		}
		cfg.Projects = updated
		if err := saveConfig(cfg); err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
			}
			os.Exit(1)
		}
		if !quiet && !isHeadless {
			fmt.Printf("Project '%s' removed successfully.\n", removeProject)
		}
		if !isHeadless && addProject == "" {
			os.Exit(0)
		}
	}

	if addProject != "" {
		cfg := loadConfig()
		exists := false
		for _, p := range cfg.Projects {
			if p == addProject {
				exists = true
				break
			}
		}
		if !exists {
			cfg.Projects = append(cfg.Projects, addProject)
			if err := saveConfig(cfg); err != nil {
				if !quiet {
					fmt.Printf("Failed to save config: %v\n", err)
				}
				os.Exit(1)
			}
			if !quiet && !isHeadless {
				fmt.Printf("Project '%s' added successfully.\n", addProject)
			}
		} else if !quiet && !isHeadless {
			fmt.Printf("Project '%s' already exists.\n", addProject)
		}

		if !isHeadless {
			os.Exit(0)
		}
	}

	if isHeadless {
		godotenv.Load()
		cfg := loadConfig()
		cwdProj, cwdHost := detectSonarProperties(".")

		if cfg.SonarURL == "" && cwdHost != "" {
			cfg.SonarURL = cwdHost
		}

		if cfg.SonarURL == "" {
			if !quiet {
				fmt.Fprintln(os.Stderr, "Error: SonarQube URL is not configured. Use --set-url <url> or run the TUI first.")
			}
			os.Exit(1)
		}
		SONAR_URL = cfg.SonarURL

		token := os.Getenv("USER_TOKEN")
		if token == "" {
			token, _ = GetToken()
		}
		if token == "" {
			if !quiet {
				fmt.Fprintln(os.Stderr, "Error: Authentication token is missing. Set USER_TOKEN env var, use --set-token, or run the TUI.")
			}
			os.Exit(1)
		}

		if explainRule != "" {
			detail, err := fetchRuleDetails(cfg.SonarURL, token, strings.TrimSpace(explainRule))
			if err != nil {
				if !quiet {
					fmt.Fprintf(os.Stderr, "Error fetching rule: %v\n", err)
				}
				os.Exit(1)
			}
			if jsonOutput {
				raw, _ := json.MarshalIndent(detail, "", "  ")
				fmt.Println(string(raw))
			} else {
				fmt.Printf("Rule: %s — %s\n", detail.Key, detail.Name)
				if detail.Type != "" || detail.Severity != "" {
					fmt.Printf("Type: %s | Severity: %s\n", detail.Type, detail.Severity)
				}
				fmt.Printf("\n%s\n", detail.Description)
			}
			os.Exit(0)
		}

		projectKey := project
		if projectKey == "" {
			projectKey = addProject
		}
		if projectKey == "" {
			projectKey = cwdProj
		}
		if projectKey == "" {
			if len(cfg.Projects) == 0 {
				if !quiet {
					fmt.Fprintln(os.Stderr, "Error: No projects configured. Use --project <key>, --add-project <key>, or sonar-project.properties.")
				}
				os.Exit(1)
			}
			projectKey = cfg.Projects[0]
		}

		if exportPath != "" {
			cliExportPath = exportPath
		}

		severitiesToUse := cfg.Severities
		if severities != "" {
			severitiesToUse = parseSeverities(severities)
		}

		qualitiesToUse := cfg.SoftwareQualities
		if qualities != "" {
			qualitiesToUse = parseSoftwareQualities(qualities)
		}

		if !quiet && !jsonOutput {
			fmt.Printf("Fetching issues for project: %s\n", projectKey)
		}

		res, err := fetchIssuesWithOptions(FetchOptions{
			BaseURL:           cfg.SonarURL,
			ProjectKey:        projectKey,
			Token:             token,
			SoftwareQualities: qualitiesToUse,
			Severities:        severitiesToUse,
			IsNewCodePeriod:   !overall,
			Branch:            branch,
			PullRequest:       pullRequest,
		})
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
			}
			os.Exit(1)
		}
		issues := res.Issues
		if res.Truncated && !quiet && !jsonOutput {
			fmt.Fprintf(os.Stderr, "Warning: Export capped at SonarQube's 10,000 issue limit (out of %d total).\n", res.Total)
		}

		if jsonOutput {
			raw, err := formatIssuesJSON(res, projectKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(raw))
			os.Exit(0)
		}

		if dryRun {
			if !quiet {
				severityCounts := map[string]int{"BLOCKER": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0}
				for _, issue := range issues {
					severityCounts[issue.Severity]++
				}
				fmt.Println("\nDry Run Summary:")
				fmt.Printf("Total Issues: %d\n", len(issues))
				fmt.Printf("BLOCKER: %d\n", severityCounts["BLOCKER"])
				fmt.Printf("HIGH:    %d\n", severityCounts["HIGH"])
				fmt.Printf("MEDIUM:  %d\n", severityCounts["MEDIUM"])
				fmt.Printf("LOW:     %d\n", severityCounts["LOW"])
			}
			os.Exit(0)
		}

		savedFile, err := exportToCSV(issues, projectKey)
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Error exporting CSV: %v\n", err)
			}
			os.Exit(1)
		}

		if !quiet {
			fmt.Printf("Export complete! Data saved to: %s\n", savedFile)
		}
		os.Exit(0)
	}

	// No flags, proceed with TUI
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}
