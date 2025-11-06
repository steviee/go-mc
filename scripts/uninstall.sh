#!/bin/bash
set -e

# go-mc uninstallation script
# Removes go-mc binary and optionally removes configuration

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="go-mc"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.config/go-mc"
DATA_DIR="$HOME/.local/share/go-mc"

# Parse command line arguments
REMOVE_CONFIG=false
REMOVE_DATA=false
FORCE=false

while [ $# -gt 0 ]; do
    case "$1" in
        --config)
            REMOVE_CONFIG=true
            shift
            ;;
        --data)
            REMOVE_DATA=true
            shift
            ;;
        --all)
            REMOVE_CONFIG=true
            REMOVE_DATA=true
            shift
            ;;
        --force|-f)
            FORCE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --config       Remove configuration files (~/.config/go-mc/)"
            echo "  --data         Remove data directory (~/.local/share/go-mc/)"
            echo "  --all          Remove binary, config, and data"
            echo "  --force, -f    Skip confirmation prompts"
            echo "  --help, -h     Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                    # Remove binary only"
            echo "  $0 --config           # Remove binary and config"
            echo "  $0 --all              # Remove everything"
            echo "  $0 --all --force      # Remove everything without confirmation"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Run '$0 --help' for usage information"
            exit 1
            ;;
    esac
done

# Functions
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

confirm() {
    local message=$1

    if [ "$FORCE" = true ]; then
        return 0
    fi

    echo -en "${YELLOW}?${NC} $message [y/N] " >&2
    read -r response
    case "$response" in
        [yY][eE][sS]|[yY])
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

remove_binary() {
    local binary_path="$INSTALL_DIR/$BINARY_NAME"

    if [ ! -f "$binary_path" ]; then
        log_warning "Binary not found at $binary_path"
        return 0
    fi

    log_info "Removing binary: $binary_path"

    # Check if we need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        if [ "$EUID" -ne 0 ]; then
            if ! command -v sudo &> /dev/null; then
                log_error "sudo not found and not running as root"
                return 1
            fi
            SUDO="sudo"
        else
            SUDO=""
        fi
    else
        SUDO=""
    fi

    if $SUDO rm -f "$binary_path"; then
        log_success "Removed binary"
    else
        log_error "Failed to remove binary"
        return 1
    fi
}

remove_config() {
    if [ ! -d "$CONFIG_DIR" ]; then
        log_warning "Config directory not found: $CONFIG_DIR"
        return 0
    fi

    # Check if there are active servers
    if [ -d "$CONFIG_DIR/servers" ]; then
        local server_count
        server_count=$(find "$CONFIG_DIR/servers" -name "*.yaml" 2>/dev/null | wc -l)
        if [ "$server_count" -gt 0 ]; then
            log_warning "Found $server_count server configuration(s)"
            if ! confirm "Remove all server configurations?"; then
                log_info "Keeping config directory"
                return 0
            fi
        fi
    fi

    log_info "Removing config directory: $CONFIG_DIR"
    if rm -rf "$CONFIG_DIR"; then
        log_success "Removed config directory"
    else
        log_error "Failed to remove config directory"
        return 1
    fi
}

remove_data() {
    if [ ! -d "$DATA_DIR" ]; then
        log_warning "Data directory not found: $DATA_DIR"
        return 0
    fi

    # Calculate size
    local size
    size=$(du -sh "$DATA_DIR" 2>/dev/null | cut -f1 || echo "unknown")

    log_warning "Data directory size: $size"
    log_warning "This will remove all backups and cached data"

    if ! confirm "Permanently delete data directory?"; then
        log_info "Keeping data directory"
        return 0
    fi

    log_info "Removing data directory: $DATA_DIR"
    if rm -rf "$DATA_DIR"; then
        log_success "Removed data directory"
    else
        log_error "Failed to remove data directory"
        return 1
    fi
}

check_running_servers() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        log_info "Checking for running servers..."

        # Try to list running servers (this might fail if binary is already removed)
        local running_count
        running_count=$($BINARY_NAME servers list --format json 2>/dev/null | grep -c '"status": "running"' || echo "0")

        if [ "$running_count" -gt 0 ]; then
            log_warning "Found $running_count running server(s)"
            log_warning "Consider stopping servers before uninstalling:"
            log_warning "  $ go-mc servers stop --all"

            if ! confirm "Continue anyway?"; then
                exit 0
            fi
        fi
    fi
}

main() {
    echo ""
    echo "═══════════════════════════════════════════════"
    echo "  go-mc Uninstallation Script"
    echo "═══════════════════════════════════════════════"
    echo ""

    # Show what will be removed
    log_info "Uninstallation plan:"
    echo "  - Binary: $INSTALL_DIR/$BINARY_NAME" >&2
    if [ "$REMOVE_CONFIG" = true ]; then
        echo "  - Config: $CONFIG_DIR" >&2
    fi
    if [ "$REMOVE_DATA" = true ]; then
        echo "  - Data: $DATA_DIR" >&2
    fi
    echo "" >&2

    # Check for running servers
    check_running_servers

    # Confirm uninstallation
    if ! confirm "Proceed with uninstallation?"; then
        log_info "Uninstallation cancelled"
        exit 0
    fi

    echo "" >&2

    # Remove binary
    if ! remove_binary; then
        log_error "Failed to remove binary"
        exit 1
    fi

    # Remove config if requested
    if [ "$REMOVE_CONFIG" = true ]; then
        if ! remove_config; then
            log_warning "Failed to remove config, continuing..."
        fi
    fi

    # Remove data if requested
    if [ "$REMOVE_DATA" = true ]; then
        if ! remove_data; then
            log_warning "Failed to remove data, continuing..."
        fi
    fi

    # Summary
    echo "" >&2
    echo "═══════════════════════════════════════════════" >&2
    log_success "Uninstallation complete!"
    echo "═══════════════════════════════════════════════" >&2
    echo "" >&2

    if [ "$REMOVE_CONFIG" = false ] || [ "$REMOVE_DATA" = false ]; then
        log_info "To remove remaining files:"
        if [ "$REMOVE_CONFIG" = false ]; then
            echo "  Config: rm -rf $CONFIG_DIR" >&2
        fi
        if [ "$REMOVE_DATA" = false ]; then
            echo "  Data: rm -rf $DATA_DIR" >&2
        fi
        echo "" >&2
    fi

    log_info "Thank you for using go-mc!"
}

# Run main function
main "$@"
