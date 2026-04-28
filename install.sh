#!/usr/bin/env bash
set -euo pipefail
APP=sonarsweep
REPO="ariffrahimin/sonarsweep"

MUTED='\033[0;2m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

usage() {
    cat <<EOF
SonarSweep Installer

Usage: install.sh [options]

Options:
    -h, --help              Display this help message
    -v, --version <version> Install a specific version (e.g., 1.0.0)
    -b, --binary <path>     Install from a local binary instead of downloading
        --no-modify-path    Don't modify shell config files (.zshrc, .bashrc, etc.)

Examples:
    curl -sSL https://raw.githubusercontent.com/ariffrahimin/sonarsweep/main/install.sh | bash
    curl -sSL https://raw.githubusercontent.com/ariffrahimin/sonarsweep/main/install.sh | bash -s -- --version 1.0.0
    ./install.sh --binary /path/to/sonarsweep
EOF
}

requested_version=${VERSION:-}
no_modify_path=false
binary_path=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            usage
            exit 0
            ;;
        -v|--version)
            if [[ -n "${2:-}" ]]; then
                requested_version="$2"
                shift 2
            else
                echo -e "${RED}Error: --version requires a version argument${NC}"
                exit 1
            fi
            ;;
        -b|--binary)
            if [[ -n "${2:-}" ]]; then
                binary_path="$2"
                shift 2
            else
                echo -e "${RED}Error: --binary requires a path argument${NC}"
                exit 1
            fi
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

INSTALL_DIR=$HOME/.sonarsweep/bin
mkdir -p "$INSTALL_DIR"

# If --binary is provided, skip all download/detection logic
if [ -n "$binary_path" ]; then
    if [ ! -f "$binary_path" ]; then
        echo -e "${RED}Error: Binary not found at ${binary_path}${NC}"
        exit 1
    fi
    specific_version="local"
else
    raw_os=$(uname -s)
    os=$(echo "$raw_os" | tr '[:upper:]' '[:lower:]')
    case "$raw_os" in
      Darwin*) os="darwin" ;;
      Linux*) os="linux" ;;
      MINGW*|MSYS*|CYGWIN*) os="windows" ;;
    esac

    arch=$(uname -m)
    if [[ "$arch" == "aarch64" ]]; then
      arch="arm64"
    fi
    if [[ "$arch" == "x86_64" ]] || [[ "$arch" == "x64" ]]; then
      arch="amd64"
    fi

    if [ "$os" = "darwin" ] && [ "$arch" = "amd64" ]; then
      rosetta_flag=$(sysctl -n sysctl.proc_translated 2>/dev/null || echo 0)
      if [ "$rosetta_flag" = "1" ]; then
        arch="arm64"
      fi
    fi

    combo="$os-$arch"
    case "$combo" in
      linux-amd64|linux-arm64|darwin-amd64|darwin-arm64|windows-amd64)
        ;;
      *)
        echo -e "${RED}Unsupported OS/Arch: $os/$arch${NC}"
        exit 1
        ;;
    esac

    archive_ext=".tar.gz"
    if [ "$os" = "windows" ]; then
      archive_ext=".zip"
    fi

    filename="$APP-$os-$arch$archive_ext"

    if [ "$os" = "windows" ]; then
        if ! command -v unzip >/dev/null 2>&1; then
            echo -e "${RED}Error: 'unzip' is required but not installed.${NC}"
            exit 1
        fi
    else
        if ! command -v tar >/dev/null 2>&1; then
             echo -e "${RED}Error: 'tar' is required but not installed.${NC}"
             exit 1
        fi
    fi

    if [ -z "$requested_version" ]; then
        url="https://github.com/$REPO/releases/latest/download/$filename"
        specific_version=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4 | sed 's/^v//')

        if [[ -z "$specific_version" ]]; then
            echo -e "${RED}Failed to fetch version information${NC}"
            exit 1
        fi
    else
        # Strip leading 'v' if present
        requested_version="${requested_version#v}"
        url="https://github.com/$REPO/releases/download/v${requested_version}/$filename"
        specific_version=$requested_version

        # Verify the release exists before downloading
        http_status=$(curl -sI -o /dev/null -w "%{http_code}" "https://github.com/$REPO/releases/tag/v${requested_version}")
        if [ "$http_status" = "404" ]; then
            # Try without 'v' just in case
            url="https://github.com/$REPO/releases/download/${requested_version}/$filename"
            http_status_no_v=$(curl -sI -o /dev/null -w "%{http_code}" "https://github.com/$REPO/releases/tag/${requested_version}")
            if [ "$http_status_no_v" = "404" ]; then
                echo -e "${RED}Error: Release v${requested_version} not found${NC}"
                echo -e "${MUTED}Available releases: https://github.com/$REPO/releases${NC}"
                exit 1
            fi
        fi
    fi
fi

print_message() {
    local level=$1
    local message=$2
    local color=""

    case $level in
        info) color="${NC}" ;;
        warning) color="${CYAN}" ;;
        error) color="${RED}" ;;
    esac

    echo -e "${color}${message}${NC}"
}

check_version() {
    if command -v $APP >/dev/null 2>&1; then
        app_path=$(which $APP)

        ## Check the installed version
        installed_version=$($APP --version 2>/dev/null | grep -o 'v[0-9]*\.[0-9]*\.[0-9]*' | sed 's/^v//' || echo "")

        if [[ "$installed_version" != "$specific_version" ]]; then
            print_message info "${MUTED}Installed version: ${NC}$installed_version."
        else
            print_message info "${MUTED}Version ${NC}$specific_version${MUTED} already installed${NC}"
            exit 0
        fi
    fi
}

unbuffered_sed() {
    if echo | sed -u -e "" >/dev/null 2>&1; then
        sed -nu "$@"
    elif echo | sed -l -e "" >/dev/null 2>&1; then
        sed -nl "$@"
    else
        local pad="$(printf "\n%512s" "")"
        sed -ne "s/$/\\${pad}/" "$@"
    fi
}

print_progress() {
    local bytes="$1"
    local length="$2"
    [ "$length" -gt 0 ] || return 0

    local width=50
    local percent=$(( bytes * 100 / length ))
    [ "$percent" -gt 100 ] && percent=100
    local on=$(( percent * width / 100 ))
    local off=$(( width - on ))

    local filled=$(printf "%*s" "$on" "")
    filled=${filled// /■}
    local empty=$(printf "%*s" "$off" "")
    empty=${empty// /･}

    printf "\r${CYAN}%s%s %3d%%${NC}" "$filled" "$empty" "$percent" >&4
}

download_with_progress() {
    local url="$1"
    local output="$2"

    if [ -t 2 ]; then
        exec 4>&2
    else
        exec 4>/dev/null
    fi

    local tmp_dir=${TMPDIR:-/tmp}
    local basename="${tmp_dir}/${APP}_install_$$"
    local tracefile="${basename}.trace"

    rm -f "$tracefile"
    mkfifo "$tracefile"

    # Hide cursor
    printf "\033[?25l" >&4

    trap "trap - RETURN; rm -f \"$tracefile\"; printf '\033[?25h' >&4; exec 4>&-" RETURN

    (
        curl --trace-ascii "$tracefile" -s -L -o "$output" "$url"
    ) &
    local curl_pid=$!

    unbuffered_sed \
        -e 'y/ACDEGHLNORTV/acdeghlnortv/' \
        -e '/^0000: content-length:/p' \
        -e '/^<= recv data/p' \
        "$tracefile" | \
    {
        local length=0
        local bytes=0

        while IFS=" " read -r -a line; do
            [ "${#line[@]}" -lt 2 ] && continue
            local tag="${line[0]} ${line[1]}"

            if [ "$tag" = "0000: content-length:" ]; then
                length="${line[2]}"
                length=$(echo "$length" | tr -d '\r')
                bytes=0
            elif [ "$tag" = "<= recv" ]; then
                local size="${line[3]}"
                bytes=$(( bytes + size ))
                if [ "$length" -gt 0 ]; then
                    print_progress "$bytes" "$length"
                fi
            fi
        done
    }

    wait $curl_pid
    local ret=$?
    echo "" >&4
    return $ret
}

download_and_install() {
    print_message info "\n${MUTED}Installing ${NC}$APP ${MUTED}version: ${NC}$specific_version"
    local tmp_dir="${TMPDIR:-/tmp}/${APP}_install_$$"
    mkdir -p "$tmp_dir"

    if [[ "$os" == "windows" ]] || ! [ -t 2 ] || ! download_with_progress "$url" "$tmp_dir/$filename"; then
        # Fallback to standard curl on Windows, non-TTY environments, or if custom progress fails
        curl -# -L -o "$tmp_dir/$filename" "$url"
    fi

    if [ "$os" = "linux" ] || [ "$os" = "darwin" ]; then
        tar -xzf "$tmp_dir/$filename" -C "$tmp_dir"
    else
        unzip -q "$tmp_dir/$filename" -d "$tmp_dir"
    fi

    # Handle windows binary name
    local bin_file="$APP"
    if [ "$os" = "windows" ]; then
        bin_file="${APP}.exe"
    fi

    if [ -f "$tmp_dir/$bin_file" ]; then
        mv "$tmp_dir/$bin_file" "${INSTALL_DIR}/"
    else
        # Sometimes tarballs contain a directory
        find "$tmp_dir" -name "$bin_file" -type f -exec mv {} "${INSTALL_DIR}/" \;
    fi

    chmod 755 "${INSTALL_DIR}/$bin_file"
    rm -rf "$tmp_dir"
}

install_from_binary() {
    print_message info "\n${MUTED}Installing ${NC}$APP ${MUTED}from: ${NC}$binary_path"
    
    local bin_file="$APP"
    if [ "$os" = "windows" ]; then
        bin_file="${APP}.exe"
    fi

    cp "$binary_path" "${INSTALL_DIR}/$bin_file"
    chmod 755 "${INSTALL_DIR}/$bin_file"
}

if [ -n "$binary_path" ]; then
    install_from_binary
else
    check_version
    download_and_install
fi

add_to_path() {
    local config_file=$1
    local command=$2

    if grep -Fxq "$command" "$config_file"; then
        print_message info "Command already exists in $config_file, skipping write."
    elif [[ -w $config_file ]]; then
        echo -e "\n# $APP" >> "$config_file"
        echo "$command" >> "$config_file"
        print_message info "${MUTED}Successfully added ${NC}$APP ${MUTED}to \$PATH in ${NC}$config_file"
    else
        print_message warning "Manually add the directory to $config_file (or similar):"
        print_message info "  $command"
    fi
}

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
    ash)
        config_files="$HOME/.ashrc $HOME/.profile /etc/profile"
    ;;
    sh)
        config_files="$HOME/.ashrc $HOME/.profile /etc/profile"
    ;;
    *)
        # Default case if none of the above matches
        config_files="$HOME/.bashrc $HOME/.bash_profile $XDG_CONFIG_HOME/bash/.bashrc $XDG_CONFIG_HOME/bash/.bash_profile"
    ;;
esac

if [[ "$no_modify_path" != "true" ]]; then
    config_file=""
    for file in $config_files; do
        if [[ -f $file ]]; then
            config_file=$file
            break
        fi
    done

    if [[ -z $config_file ]]; then
        print_message warning "No config file found for $current_shell. You may need to manually add to PATH:"
        print_message info "  export PATH=\"$INSTALL_DIR:\$PATH\""
    elif [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        case $current_shell in
            fish)
                add_to_path "$config_file" "fish_add_path \"$INSTALL_DIR\""
            ;;
            *)
                add_to_path "$config_file" "export PATH=\"$INSTALL_DIR:\$PATH\""
            ;;
        esac
    fi
fi

if [ -n "${GITHUB_ACTIONS-}" ] && [ "${GITHUB_ACTIONS}" == "true" ]; then
    echo "$INSTALL_DIR" >> $GITHUB_PATH
    print_message info "Added $INSTALL_DIR to \$GITHUB_PATH"
fi

echo -e ""
echo -e "${CYAN} ▗▄▄▖ ▗▄▖ ▗▖  ▗▖ ▗▄▖ ▗▄▄▖  ▗▄▄▖▗▖ ▗▖▗▄▄▄▖▗▄▄▄▖▗▄▄▖ ${NC}"
echo -e "${CYAN}▐▌   ▐▌ ▐▌▐▛▚▖▐▌▐▌ ▐▌▐▌ ▐▌▐▌   ▐▌ ▐▌▐▌   ▐▌   ▐▌ ▐▌${NC}"
echo -e "${CYAN} ▝▀▚▖▐▌ ▐▌▐▌ ▝▜▌▐▛▀▜▌▐▛▀▚▖ ▝▀▚▖▐▌ ▐▌▐▛▀▀▘▐▛▀▀▘▐▛▀▘ ${NC}"
echo -e "${CYAN}▗▄▄▞▘▝▚▄▞▘▐▌  ▐▌▐▌ ▐▌▐▌ ▐▌▗▄▄▞▘▐▙█▟▌▐▙▄▄▖▐▙▄▄▖▐▌   ${NC}"
echo -e ""
echo -e "${MUTED}SonarSweep is ready to use!${NC}"
echo -e ""
echo -e "${CYAN}To get started:${NC}"
echo -e "  sonarsweep"
echo -e ""
echo -e "${MUTED}Make sure to restart your terminal or source your shell config to use it.${NC}"
echo -e ""
