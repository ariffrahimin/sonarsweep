# SonarSweep

```
 ▗▄▄▖ ▗▄▖ ▗▖  ▗▖ ▗▄▖ ▗▄▄▖  ▗▄▄▖▗▖ ▗▖▗▄▄▄▖▗▄▄▄▖▗▄▄▖ 
▐▌   ▐▌ ▐▌▐▛▚▖▐▌▐▌ ▐▌▐▌ ▐▌▐▌   ▐▌ ▐▌▐▌   ▐▌   ▐▌ ▐▌
 ▝▀▚▖▐▌ ▐▌▐▌ ▝▜▌▐▛▀▜▌▐▛▀▚▖ ▝▀▚▖▐▌ ▐▌▐▛▀▀▘▐▛▀▀▘▐▛▀▘ 
▗▄▄▞▘▝▚▄▞▘▐▌  ▐▌▐▌ ▐▌▐▌ ▐▌▗▄▄▞▘▐▙█▟▌▐▙▄▄▖▐▙▄▄▖▐▌   
```

### Fetch SonarQube issues to clean, readable CSV files — beautifully.

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux%20%7C%20Windows-blue?style=flat-square)](https://github.com/ariffrahimin/sonarsweep/releases)
[![Build](https://img.shields.io/badge/Build-Passing-brightgreen?style=flat-square)](https://github.com/ariffrahimin/sonarsweep/actions)

---

## Features

- **Beautiful TUI** — Interactive terminal interface with arrow-key navigation and real-time progress
- **Smart Pagination** — Automatically handles large SonarQube projects with hundreds of issues
- **Flexible Filtering** — Filter by impact severities (HIGH, MEDIUM, LOW) and software qualities (RELIABILITY, SECURITY, MAINTAINABILITY)
- **Clean CSV Export** — Outputs structured CSV files organized by project folder
- **Persistent Configuration** — JSON-based config saves your URL and projects between runs
- **CLI Flags** — Full automation support for CI/CD pipelines with `--dry-run`, `--export`, `--quiet`
- **Secure Token Handling** — Token stored in `.env`, never in config files

---

## Installation

### Option 1: Download Binary (Fastest)

Download the pre-built binary for your platform from the [Releases page](https://github.com/ariffrahimin/sonarsweep/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/ariffrahimin/sonarsweep/releases/latest/download/sonarsweep-darwin-arm64.tar.gz | tar -xz

# macOS (Intel)
curl -L https://github.com/ariffrahimin/sonarsweep/releases/latest/download/sonarsweep-darwin-amd64.tar.gz | tar -xz

# Linux
curl -L https://github.com/ariffrahimin/sonarsweep/releases/latest/download/sonarsweep-linux-amd64.tar.gz | tar -xz

# Windows (PowerShell)
irm https://github.com/ariffrahimin/sonarsweep/releases/latest/download/sonarsweep-windows-amd64.zip -o sonarsweep.zip
Expand-Archive sonarsweep.zip -DestinationPath .
```

### Option 2: Homebrew (Recommended for macOS/Linux)

```bash
brew install ariffrahimin/tap/sonarsweep
```

### Option 3: Build from Source

```bash
git clone https://github.com/ariffrahimin/sonarsweep.git
cd sonarsweep
go build -o sonarsweep main.go
```

---

## Quick Start

### Step 1: Set up your SonarQube Token

Create a `.env` file in the same directory as `sonarsweep`:

```bash
USER_TOKEN=your_sonarqube_token_here
```

> **Note:** Your token must have access to the SonarQube projects you want to fetch. Generate one from SonarQube → My Account → Security.

### Step 2: Run SonarSweep

```bash
./sonarsweep
```

On the first run, SonarSweep will ask for your SonarQube URL and Project Key. After that, it remembers your settings.

### Step 3: Navigate the TUI

```
↑/↓        Navigate the list
Space      Toggle selection (for Software Qualities)
Enter      Confirm / Select
Esc        Cancel / Quit
```

Your CSV files will be saved in folders named after each project (e.g. `people-web-ppd/sonarqube_issues_20260425.csv`).

---

## CLI Flags

SonarSweep supports full command-line operation for automation and scripting:

| Flag | Description | Example |
|------|-------------|---------|
| `-h`, `--help` | Show this help message | `sonarsweep --help` |
| `-v`, `--version` | Show version information | `sonarsweep --version` |
| `--reset` | Clear saved URL and Projects | `sonarsweep --reset` |
| `--config <path>` | Use a different config file | `sonarsweep --config /path/to/config.json` |
| `-c` | Shorthand for `--config` | `sonarsweep -c my-config.json` |
| `--view-config` | Print current configuration | `sonarsweep --view-config` |
| `--list-projects` | List all saved projects | `sonarsweep --list-projects` |
| `--add-project <key>` | Add a project to config | `sonarsweep --add-project my-project` |
| `--export <path>` | Override CSV export path | `sonarsweep --export /tmp/output.csv` |
| `--dry-run` | Fetch issues without saving | `sonarsweep --dry-run` |
| `-q`, `--quiet` | Headless mode (no TUI) | `sonarsweep --quiet` |

---

## Configuration File

SonarSweep stores configuration in `sonarsweep.json` in the same directory:

```json
{
  "sonarqube_url": "http://12.345.678.199:9000",
  "projects": [
    "project-web-ppd",
    "project-web-stg",
    "project-api-ppd"
  ],
  "software_qualities": [
    "RELIABILITY",
    "SECURITY",
    "MAINTAINABILITY"
  ]
}
```

### Managing Projects via CLI

```bash
# Add a new project
sonarsweep --add-project new-project-key

# List all saved projects
sonarsweep --list-projects

# Reset (clear URL and Projects)
sonarsweep --reset

# View current configuration
sonarsweep --view-config
```

---

## Security

**Your SonarQube token must never be committed to version control.**

| What TO DO | What NOT TO DO |
|------------|----------------|
| ✅ Store token in `.env` file | ❌ Put token in `sonarsweep.json` |
| ✅ Add `.env` to `.gitignore` | ❌ Commit `sonarsweep.json` with real URLs |
| ✅ Use `--config` for alternate setups | ❌ Share config files with secrets |

The `.env` file is already in `.gitignore` by default. If you accidentally commit secrets, rotate them immediately in SonarQube.

---

## Development

### Build

```bash
go build -o sonarsweep main.go
```

### Test

```bash
go test ./...
```

### Run

```bash
go run main.go
```

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

---

**Made with ❤️ for the SonarQube community**
