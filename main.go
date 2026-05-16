package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

var VERSION = "dev"

func printHelp() {
	helpText := `SonarSweep v%s - SonarQube Issue Exporter

USAGE:
  sonarsweep [flags]

FLAGS:
  -h, --help           Show this help message and exit
  -v, --version        Show version information and exit
  --reset              Reset the saved URL and Projects from config
  --config <path>      Use a different configuration file
  --view-config        Print current configuration and exit
  --list-projects      List all saved projects and exit
  --add-project <key>  Add a project to the configuration
  --export <path>      Override the CSV export path
  --dry-run            Fetch issues but skip CSV export
  -q, --quiet          Run in headless mode (no TUI)

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

  The token (USER_TOKEN) must be stored in a .env file, saved in 
  the config file, or entered securely in the TUI prompt.

  Default config location: ~/.config/sonarsweep/config.json
  Override with: --config /path/to/config.json
`
	fmt.Printf(helpText, VERSION)
}

func main() {
	var (
		help         bool
		version      bool
		reset        bool
		viewConfig   bool
		listProjects bool
		addProject   string
		exportPath   string
		dryRun       bool
		quiet        bool
	)

	flag.BoolVar(&help, "help", false, "Show this help message and exit")
	flag.BoolVar(&help, "h", false, "Show this help message and exit (shorthand)")
	flag.BoolVar(&version, "version", false, "Show version information and exit")
	flag.BoolVar(&version, "v", false, "Show version information and exit (shorthand)")
	flag.BoolVar(&reset, "reset", false, "Reset the saved URL and Projects from config")
	flag.BoolVar(&viewConfig, "view-config", false, "Print current configuration and exit")
	flag.BoolVar(&listProjects, "list-projects", false, "List all saved projects and exit")
	flag.StringVar(&addProject, "add-project", "", "Add a project to the configuration")
	flag.StringVar(&exportPath, "export", "", "Override the CSV export path")
	flag.BoolVar(&dryRun, "dry-run", false, "Fetch issues but skip CSV export")
	flag.BoolVar(&quiet, "quiet", false, "Run in headless mode (no TUI)")
	flag.BoolVar(&quiet, "q", false, "Run in headless mode (no TUI) (shorthand)")
	flag.StringVar(&configPath, "config", "", "Use a different configuration file")
	flag.StringVar(&configPath, "c", "", "Use a different configuration file (shorthand)")

	flag.Usage = printHelp
	flag.Parse()

	if help {
		printHelp()
		os.Exit(0)
	}

	if version {
		fmt.Printf("SonarSweep v%s\n", VERSION)
		os.Exit(0)
	}

	if reset {
		cfg := loadConfig()
		cfg.SonarURL = ""
		cfg.Projects = []string{}
		if err := saveConfig(cfg); err != nil {
			fmt.Printf("Failed to reset config: %v\n", err)
			os.Exit(1)
		}
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

	isHeadless := quiet || dryRun || exportPath != ""

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

		if cfg.SonarURL == "" {
			if !quiet {
				fmt.Fprintln(os.Stderr, "Error: SonarQube URL is not configured. Run the TUI first to set it.")
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
				fmt.Fprintln(os.Stderr, "Error: Authentication token is missing. Set USER_TOKEN env var or run the TUI to save it.")
			}
			os.Exit(1)
		}

		projectKey := addProject
		if projectKey == "" {
			if len(cfg.Projects) == 0 {
				if !quiet {
					fmt.Fprintln(os.Stderr, "Error: No projects configured. Use --add-project <key> to specify one.")
				}
				os.Exit(1)
			}
			projectKey = cfg.Projects[0]
		}

		if exportPath != "" {
			cliExportPath = exportPath
		}

		if !quiet {
			fmt.Printf("Fetching issues for project: %s\n", projectKey)
		}

		issues, err := fetchIssues(projectKey, token, cfg.SoftwareQualities, true)
		if err != nil {
			if !quiet {
				fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
			}
			os.Exit(1)
		}

		if dryRun {
			if !quiet {
				severityCounts := map[string]int{"HIGH": 0, "MEDIUM": 0, "LOW": 0}
				for _, issue := range issues {
					severityCounts[issue.Severity]++
				}
				fmt.Println("\nDry Run Summary:")
				fmt.Printf("Total Issues: %d\n", len(issues))
				fmt.Printf("HIGH:   %d\n", severityCounts["HIGH"])
				fmt.Printf("MEDIUM: %d\n", severityCounts["MEDIUM"])
				fmt.Printf("LOW:    %d\n", severityCounts["LOW"])
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
