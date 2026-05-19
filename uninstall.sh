#!/usr/bin/env bash
set -euo pipefail
APP=sonarsweep
INSTALL_DIR="$HOME/.sonarsweep/bin"

echo "Removing $APP..."

rm -rf "$HOME/.sonarsweep"

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
            echo "Removed PATH entry from $config"
        fi
    fi
done

echo "Done. Restart terminal or source shell config."