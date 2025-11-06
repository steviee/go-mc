#!/bin/bash
set -e

# go-mc installation script
# Downloads and installs the latest release from GitHub

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="steviee/go-mc"
BINARY_NAME="go-mc"
INSTALL_DIR="/usr/local/bin"
VERSION=""
SKIP_CHECKSUM=false
FORCE=false

# Functions - all output to stderr to avoid contaminating function return values
log_info() {
    echo -e "${BLUE}ℹ${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}✓${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1" >&2
}

log_error() {
    echo -e "${RED}✗${NC} $1" >&2
}

check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "Required command '$1' not found"
        return 1
    fi
    return 0
}

detect_os() {
    local os_type
    os_type=$(uname -s)

    if [ "$os_type" != "Linux" ]; then
        log_error "Unsupported OS: $os_type"
        log_error "go-mc only supports Linux (Debian 12/13)"
        exit 1
    fi

    # Check if running on Debian (optional warning)
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        if [ "$ID" != "debian" ]; then
            log_warning "go-mc is designed for Debian 12/13"
            log_warning "Detected: $PRETTY_NAME"
            log_warning "Installation will continue, but compatibility is not guaranteed"
        fi
    fi
}

detect_arch() {
    local arch
    arch=$(uname -m)

    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            log_error "Supported architectures: amd64, arm64"
            exit 1
            ;;
    esac
}

get_version_from_git_commit() {
    local current_commit

    # Try to get current git commit
    if ! current_commit=$(git rev-parse HEAD 2>/dev/null); then
        log_warning "Not in a git repository, cannot auto-detect version from commit"
        return 1
    fi

    log_info "Detected git commit: ${current_commit:0:7}"
    log_info "Searching for matching release..."

    # Fetch all releases and find one matching this commit
    local releases
    releases=$(curl -fsSL "https://api.github.com/repos/$REPO/releases?per_page=50" 2>/dev/null)

    if [ -z "$releases" ]; then
        log_warning "Failed to fetch releases from GitHub"
        return 1
    fi

    # Parse releases and find matching commit
    local version
    version=$(echo "$releases" | grep -B 5 "\"target_commitish\": \"$current_commit\"" | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -n "$version" ]; then
        log_success "Found matching release: $version (commit: ${current_commit:0:7})"
        echo "$version"
        return 0
    fi

    # If exact match not found, check if there's a tag on this commit
    local tag
    tag=$(git describe --exact-match --tags HEAD 2>/dev/null)

    if [ -n "$tag" ]; then
        log_success "Found git tag on current commit: $tag"
        echo "$tag"
        return 0
    fi

    log_warning "No release found for commit ${current_commit:0:7}"
    return 1
}

get_latest_version() {
    log_info "Fetching latest release version..."

    local version
    version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$version" ]; then
        log_error "Failed to fetch latest version from GitHub"
        log_error "This could be due to:"
        log_error "  - Network connectivity issues"
        log_error "  - GitHub API rate limiting"
        log_error "  - Invalid repository name"
        log_info "Try again later or specify a version with --version"
        exit 1
    fi

    echo "$version"
}

download_file() {
    local url=$1
    local output=$2

    if ! curl -fsSL -o "$output" "$url"; then
        log_error "Failed to download: $url"
        log_error "Possible causes:"
        log_error "  - Network connectivity issues"
        log_error "  - File not found (404)"
        log_error "  - Invalid URL"
        log_info "Check your internet connection and try again"
        return 1
    fi
    return 0
}

verify_checksum() {
    local file=$1
    local checksum_file=$2

    log_info "Verifying checksum..."

    if ! sha256sum -c "$checksum_file" &> /dev/null; then
        log_error "Checksum verification failed!"
        log_error "The downloaded file does not match the expected checksum."
        log_error "This could indicate:"
        log_error "  - Corrupted download"
        log_error "  - Network tampering"
        log_error "  - Man-in-the-middle attack"
        log_warning "DO NOT install untrusted binaries!"
        log_info "Try downloading again or use --skip-checksum (not recommended)"
        return 1
    fi

    log_success "Checksum verified"
    return 0
}

install_binary() {
    local binary=$1
    local install_path="$INSTALL_DIR/$BINARY_NAME"

    log_info "Installing $BINARY_NAME to $INSTALL_DIR..."

    # Check if we need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        if [ "$EUID" -ne 0 ]; then
            log_info "Requires root privileges for installation"
            if ! command -v sudo &> /dev/null; then
                log_error "sudo not found and not running as root"
                log_error "Please run as root or install sudo"
                exit 1
            fi
            SUDO="sudo"
        else
            SUDO=""
        fi
    else
        SUDO=""
    fi

    # Remove old version if exists
    if [ -f "$install_path" ]; then
        log_warning "Existing installation found at $install_path"
        local old_version
        old_version=$($install_path version 2>/dev/null | head -n1 || echo "unknown")
        log_info "Current version: $old_version"
    fi

    # Install new binary
    if ! $SUDO install -m 755 "$binary" "$install_path"; then
        log_error "Failed to install binary"
        exit 1
    fi

    log_success "Installed: $install_path"
}

show_help() {
    cat << EOF
go-mc Installation Script

Usage: $0 [OPTIONS]

Options:
  --version <version>    Install specific go-mc release version (e.g., v0.0.8)
                         NOT the Debian version! Use go-mc release tags.
                         Default: Auto-detect from git commit, or latest if not in repo
                         See releases: https://github.com/$REPO/releases

  --install-dir <dir>    Custom installation directory
                         Default: /usr/local/bin

  --skip-checksum        Skip SHA256 checksum verification (not recommended)

  --force                Force reinstallation even if already installed

  --help, -h             Show this help message

Environment Variables:
  INSTALL_DIR            Installation directory (overridden by --install-dir)

Version Detection:
  When run without --version flag:
  1. If in a git repository: finds release matching current commit/tag
  2. Otherwise: installs latest release

Examples:
  # Auto-detect version from git commit (if in repo), or install latest
  $0

  # Install specific go-mc version (use release tag, not Debian version!)
  $0 --version v0.0.8

  # Install to custom directory
  $0 --install-dir /usr/bin

  # Install with pip-style pattern (auto-detects latest)
  curl -fsSL https://raw.githubusercontent.com/steviee/go-mc/main/scripts/install.sh | sudo bash

Repository: https://github.com/$REPO
Available versions: https://github.com/$REPO/releases

EOF
}

validate_version() {
    local version=$1

    # Check if version contains spaces
    if [[ "$version" =~ [[:space:]] ]]; then
        log_error "Invalid version format: '$version'"
        log_error "Version must be a go-mc release tag (e.g., v0.0.8), not a Debian version"
        log_info "To see available versions, visit: https://github.com/$REPO/releases"
        log_info "Example: --version v0.0.8"
        return 1
    fi

    # Check if version starts with 'v'
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        log_warning "Version '$version' doesn't follow semver format (vX.Y.Z)"
        log_warning "This may fail if the release doesn't exist"
    fi

    return 0
}

parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --version)
                if [ -z "$2" ] || [ "${2:0:1}" = "-" ]; then
                    log_error "Option --version requires a value"
                    log_info "Example: --version v0.0.7"
                    exit 1
                fi
                VERSION="$2"
                if ! validate_version "$VERSION"; then
                    exit 1
                fi
                shift 2
                ;;
            --install-dir)
                if [ -z "$2" ] || [ "${2:0:1}" = "-" ]; then
                    log_error "Option --install-dir requires a value"
                    log_info "Example: --install-dir /usr/local/bin"
                    exit 1
                fi
                INSTALL_DIR="$2"
                shift 2
                ;;
            --skip-checksum)
                SKIP_CHECKSUM=true
                log_warning "Checksum verification will be skipped"
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                log_info "Run '$0 --help' for usage information"
                exit 1
                ;;
        esac
    done
}

main() {
    echo ""
    echo "═══════════════════════════════════════════════"
    echo "  go-mc Installation Script"
    echo "═══════════════════════════════════════════════"
    echo ""

    # Check prerequisites
    log_info "Checking prerequisites..."
    check_command curl || exit 1
    check_command sha256sum || exit 1
    log_success "All prerequisites found"

    # Detect system
    log_info "Detecting system configuration..."
    detect_os
    local arch
    arch=$(detect_arch)
    log_success "Detected architecture: $arch"

    # Get version
    local version
    if [ -n "$VERSION" ]; then
        version="$VERSION"
        log_info "Using specified version: $version"
    else
        # Try to auto-detect version from git commit first
        if version=$(get_version_from_git_commit); then
            log_info "Auto-detected version from git commit: $version"
        else
            # Fall back to latest version
            version=$(get_latest_version)
            log_success "Latest version: $version"
        fi
    fi

    # Prepare download
    local binary_name="${BINARY_NAME}-linux-${arch}"
    local download_url="https://github.com/$REPO/releases/download/$version/$binary_name"
    local checksum_url="${download_url}.sha256"
    local temp_dir
    temp_dir=$(mktemp -d)
    trap 'rm -rf "$temp_dir"' EXIT

    local binary_path="$temp_dir/$binary_name"
    local checksum_path="$temp_dir/${binary_name}.sha256"

    # Download binary
    log_info "Downloading $binary_name..."
    if ! download_file "$download_url" "$binary_path"; then
        exit 1
    fi
    log_success "Downloaded binary"

    # Download and verify checksum
    if [ "$SKIP_CHECKSUM" = true ]; then
        log_warning "Skipping checksum verification (--skip-checksum flag set)"
    else
        log_info "Downloading checksum..."
        if ! download_file "$checksum_url" "$checksum_path"; then
            log_warning "Checksum file not found, skipping verification"
        else
            # Verify checksum
            if ! (cd "$temp_dir" && verify_checksum "$binary_name" "${binary_name}.sha256"); then
                exit 1
            fi
        fi
    fi

    # Install
    install_binary "$binary_path"

    # Verify installation
    log_info "Verifying installation..."
    if command -v "$BINARY_NAME" &> /dev/null; then
        local installed_version
        installed_version=$($BINARY_NAME version 2>/dev/null | head -n1 || echo "unknown")
        log_success "Installation verified: $installed_version"
    else
        log_warning "Binary installed but not found in PATH"
        log_warning "You may need to add $INSTALL_DIR to your PATH"
    fi

    # Next steps
    echo ""
    echo "═══════════════════════════════════════════════"
    echo -e "${GREEN}✓ Installation complete!${NC}"
    echo "═══════════════════════════════════════════════"
    echo ""
    echo "Next steps:"
    echo "  1. Run initial setup:"
    echo "     $ go-mc system setup"
    echo ""
    echo "  2. Create your first server:"
    echo "     $ go-mc servers create survival"
    echo ""
    echo "  3. Start the server:"
    echo "     $ go-mc servers start survival"
    echo ""
    echo "  4. Monitor with TUI dashboard:"
    echo "     $ go-mc servers top"
    echo ""
    echo "For more information:"
    echo "  Documentation: https://github.com/$REPO"
    echo "  Issues: https://github.com/$REPO/issues"
    echo ""
}

# Run main function only if script is executed directly (not sourced)
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    parse_args "$@"
    main
fi
