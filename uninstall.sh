#!/usr/bin/env bash
set -euo pipefail
APP=sonarsweep
INSTALL_DIR="$HOME/.sonarsweep/bin"
REPO="ariffrahimin/sonarsweep"

RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

usage() {
    cat <<EOF
SonarSweep Uninstaller

Usage: uninstall.sh [options]

Options:
    -h, --help              Display this help message
        --no-modify-path    Don't modify shell config files

Examples:
    curl -sSL https://raw.githubusercontent.com/$REPO/main/uninstall.sh | bash
    ./uninstall.sh
EOF
}

no_modify_path=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            usage
            exit 0
            ;;
        --no-modify-path)
            no_modify_path=true
            shift
            ;;
        *)
            echo -e "${CYAN}Warning: Unknown option '$1'${NC}" >&2
            shift
            ;;
    esac
done

echo -e "${CYAN}Uninstalling $APP...${NC}"

uninstalled=false

# 1. Check Homebrew first
if command -v brew >/dev/null 2>&1; then
    if brew list "$APP" >/dev/null 2>&1; then
        echo -e "${CYAN}Found Homebrew installation, uninstalling...${NC}"
        brew uninstall "$APP"
        echo -e "${CYAN}Removed via Homebrew${NC}"
        uninstalled=true
    fi
fi

# 2. Check install script location
if [[ -d "$HOME/.sonarsweep" ]]; then
    rm -rf "$HOME/.sonarsweep"
    echo -e "${CYAN}Removed $HOME/.sonarsweep${NC}"
    uninstalled=true
else
    if [[ "$uninstalled" == "false" ]]; then
        echo -e "${CYAN}Directory $HOME/.sonarsweep not found${NC}"
    fi
fi

# 3. Clean shell config PATH entries
if [[ "$no_modify_path" == "true" ]]; then
    echo -e "${CYAN}Skipping PATH cleanup (--no-modify-path specified)${NC}"
else
    XDG_CONFIG_HOME=${XDG_CONFIG_HOME:-$HOME/.config}
    current_shell=$(basename "$SHELL" 2>/dev/null || echo "bash")

    case $current_shell in
        fish)
            config_files="$HOME/.config/fish/config.fish"
            ;;
        zsh)
            config_files="${ZDOTDIR:-$HOME}/.zshrc ${ZDOTDIR:-$HOME}/.zshenv $XDG_CONFIG_HOME/zsh/.zshrc $XDG_CONFIG_HOME/zsh/.zshenv"
            ;;
        bash)
            config_files="$HOME/.bashrc $HOME/.bash_profile $HOME/.profile $XDG_CONFIG_HOME/bash/.bashrc $XDG_CONFIG_HOME/bash/.bash_profile"
            ;;
        *)
            config_files="$HOME/.bashrc $HOME/.bash_profile $HOME/.profile"
            ;;
    esac

    for config in $config_files; do
        if [[ -f "$config" ]]; then
            if grep -qE "(# $APP|fish_add_path.*$APP|$INSTALL_DIR)" "$config" 2>/dev/null; then
                if [[ "$(uname)" == "Darwin" ]]; then
                    sed -i '' -e "\@$INSTALL_DIR@d" -e "/# $APP/d" "$config"
                else
                    sed -i -e "\@$INSTALL_DIR@d" -e "/# $APP/d" "$config"
                fi
                echo -e "${CYAN}Removed PATH entry from $config${NC}"
            fi
        fi
    done
fi

# 4. Source build warning
if ! command -v "$APP" >/dev/null 2>&1; then
    if [[ -f "$HOME/go/bin/$APP" ]] || [[ -f "/usr/local/bin/$APP" ]]; then
        echo ""
        echo -e "${RED}Note: If you built from source, manually remove the binary:${NC}"
        echo -e "${CYAN}  rm ~/go/bin/$APP${NC}"
        echo -e "${CYAN}  rm /usr/local/bin/$APP${NC}"
    fi
fi

echo ""
echo -e "${CYAN}Done. Restart terminal or source shell config.${NC}"