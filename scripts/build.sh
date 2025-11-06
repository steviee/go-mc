#!/bin/bash
set -e

# go-mc build script
# Builds binaries for multiple architectures

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="go-mc"
BUILD_DIR="build"
CMD_DIR="cmd/go-mc"
VERSION=${VERSION:-"dev"}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-s -w -X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME"

# Parse command line arguments
ARCHITECTURES=()
CLEAN=false
VERBOSE=false

while [ $# -gt 0 ]; do
    case "$1" in
        --amd64)
            ARCHITECTURES+=("amd64")
            shift
            ;;
        --arm64)
            ARCHITECTURES+=("arm64")
            shift
            ;;
        --all)
            ARCHITECTURES=("amd64" "arm64")
            shift
            ;;
        --clean)
            CLEAN=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --amd64        Build for amd64 architecture"
            echo "  --arm64        Build for arm64 architecture"
            echo "  --all          Build for all architectures (default)"
            echo "  --clean        Clean build directory before building"
            echo "  --verbose, -v  Verbose output"
            echo "  --help, -h     Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  VERSION        Version string (default: dev)"
            echo "  COMMIT         Git commit hash (default: auto-detected)"
            echo ""
            echo "Examples:"
            echo "  $0                        # Build for all architectures"
            echo "  $0 --amd64                # Build for amd64 only"
            echo "  $0 --all --clean          # Clean and build all"
            echo "  VERSION=v1.0.0 $0 --all   # Build with version v1.0.0"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Run '$0 --help' for usage information"
            exit 1
            ;;
    esac
done

# Default to all architectures if none specified
if [ ${#ARCHITECTURES[@]} -eq 0 ]; then
    ARCHITECTURES=("amd64" "arm64")
fi

# Functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

build_arch() {
    local arch=$1
    local output="$BUILD_DIR/${BINARY_NAME}-linux-${arch}"

    log_info "Building for linux/$arch..."

    local build_cmd="GOOS=linux GOARCH=$arch go build"
    if [ "$VERBOSE" = true ]; then
        build_cmd="$build_cmd -v"
    fi
    build_cmd="$build_cmd -ldflags \"$LDFLAGS\" -o \"$output\" ./$CMD_DIR"

    if eval "$build_cmd"; then
        local size
        size=$(du -h "$output" | cut -f1)
        log_success "Built: $output ($size)"

        # Generate checksum
        (cd "$BUILD_DIR" && sha256sum "$(basename "$output")" > "$(basename "$output").sha256")
        log_success "Generated checksum: $output.sha256"

        return 0
    else
        log_error "Build failed for linux/$arch"
        return 1
    fi
}

main() {
    echo ""
    echo "═══════════════════════════════════════════════"
    echo "  go-mc Build Script"
    echo "═══════════════════════════════════════════════"
    echo ""

    # Show build configuration
    log_info "Build configuration:"
    echo "  Version: $VERSION"
    echo "  Commit: $COMMIT"
    echo "  Build time: $BUILD_TIME"
    echo "  Architectures: ${ARCHITECTURES[*]}"
    echo ""

    # Clean if requested
    if [ "$CLEAN" = true ]; then
        log_info "Cleaning build directory..."
        rm -rf "$BUILD_DIR"
        log_success "Cleaned"
    fi

    # Create build directory
    mkdir -p "$BUILD_DIR"

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        log_error "Please install Go 1.21+ from https://go.dev/dl/"
        exit 1
    fi

    local go_version
    go_version=$(go version | awk '{print $3}')
    log_info "Using Go: $go_version"
    echo ""

    # Build for each architecture
    local failed=0
    for arch in "${ARCHITECTURES[@]}"; do
        if ! build_arch "$arch"; then
            failed=$((failed + 1))
        fi
        echo ""
    done

    # Summary
    echo "═══════════════════════════════════════════════"
    if [ $failed -eq 0 ]; then
        log_success "All builds completed successfully"
    else
        log_error "$failed build(s) failed"
        exit 1
    fi
    echo "═══════════════════════════════════════════════"
    echo ""

    # List built binaries
    log_info "Built binaries:"
    ls -lh "$BUILD_DIR"/${BINARY_NAME}-* 2>/dev/null | awk '{print "  " $9 " (" $5 ")"}'
    echo ""
}

# Run main function
main "$@"
