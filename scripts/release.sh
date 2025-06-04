#!/bin/bash

# Release script for pprofio Go package
# This script validates the repository state and creates a release tag
# GitHub Actions will handle the actual release creation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Function to validate semantic version format
validate_version() {
    if [[ ! $1 =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        print_error "Version must be in format vX.Y.Z (e.g., v1.2.3)"
    fi
}

# Function to check if version already exists
check_version_exists() {
    if git tag -l | grep -q "^$1$"; then
        print_error "Version $1 already exists"
    fi
}

print_status "Starting release validation for pprofio Go package..."

# Check if we're in the correct directory
if [[ ! -f "go.mod" ]] || [[ ! -f "pprofio.go" ]]; then
    print_error "This script must be run from the pprofio package root directory"
fi

# Check if we're on the main branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    print_error "Releases must be created from the main branch. Current branch: $CURRENT_BRANCH"
fi

# Check for uncommitted changes
if [[ -n "$(git status --porcelain)" ]]; then
    print_error "Working directory must be clean. Please commit or stash your changes."
fi

# Fetch latest changes
print_status "Fetching latest changes from origin..."
git fetch origin

# Check if local main is up to date with remote
LOCAL_HASH=$(git rev-parse HEAD)
REMOTE_HASH=$(git rev-parse origin/main)
if [[ "$LOCAL_HASH" != "$REMOTE_HASH" ]]; then
    print_error "Local main branch is not up to date with origin/main. Please pull latest changes."
fi

# Show current tags
print_status "Current releases:"
git tag -l | sort -V | tail -5 || echo "No previous releases found"

# Get version from user if not provided
if [[ -z "$1" ]]; then
    echo
    echo -e "${YELLOW}Enter new version (format: vX.Y.Z):${NC}"
    read -r VERSION
else
    VERSION="$1"
fi

# Validate version format
validate_version "$VERSION"

# Check if version already exists
check_version_exists "$VERSION"

print_status "Preparing release $VERSION..."

# Run tests
print_status "Running comprehensive test suite..."
if ! go test -v -race ./...; then
    print_error "Tests failed. Please fix issues before releasing."
fi
print_success "All tests passed"

# Run linting
print_status "Running code quality checks..."
if command -v golangci-lint &> /dev/null; then
    if ! golangci-lint run --timeout=10m; then
        print_warning "Linting issues detected. Consider fixing before release."
        echo
        echo -e "${YELLOW}Continue with release despite linting issues? (y/N):${NC}"
        read -r CONTINUE
        if [[ "$CONTINUE" != "y" && "$CONTINUE" != "Y" ]]; then
            print_error "Release cancelled due to linting issues"
        fi
    else
        print_success "Code quality checks passed"
    fi
else
    print_warning "golangci-lint not found. Skipping linting checks."
fi

# Check if go.mod is tidy
print_status "Verifying go.mod is tidy..."
go mod tidy
if [[ -n "$(git status --porcelain go.mod go.sum)" ]]; then
    print_error "go.mod/go.sum not tidy. Please run 'go mod tidy' and commit changes."
fi
print_success "go.mod is tidy"

# Build to ensure everything compiles
print_status "Verifying package builds correctly..."
if ! go build ./...; then
    print_error "Package failed to build"
fi
print_success "Package builds successfully"

# Check if CHANGELOG.md has been updated
if [[ -f "CHANGELOG.md" ]]; then
    if grep -q "## \[Unreleased\]" CHANGELOG.md; then
        print_warning "CHANGELOG.md still contains [Unreleased] section"
        print_warning "Consider updating CHANGELOG.md before release"
        echo
        echo -e "${YELLOW}Continue without updating CHANGELOG.md? (y/N):${NC}"
        read -r CONTINUE_CHANGELOG
        if [[ "$CONTINUE_CHANGELOG" != "y" && "$CONTINUE_CHANGELOG" != "Y" ]]; then
            print_error "Please update CHANGELOG.md before releasing"
        fi
    fi
else
    print_warning "CHANGELOG.md not found"
fi

# Final confirmation
echo
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Release Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Version: ${GREEN}$VERSION${NC}"
echo -e "Branch: ${GREEN}$CURRENT_BRANCH${NC}"
echo -e "Commit: ${GREEN}$(git rev-parse --short HEAD)${NC}"
echo -e "Module: ${GREEN}$(grep '^module' go.mod | cut -d' ' -f2)${NC}"
echo -e "${BLUE}========================================${NC}"
echo

echo -e "${YELLOW}Create release tag $VERSION? This will trigger GitHub Actions to build and publish the release. (y/N):${NC}"
read -r CONFIRM

if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
    print_error "Release cancelled by user"
fi

# Create annotated tag
print_status "Creating release tag $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION

$(date): Released version $VERSION of pprofio Go profiling client

This release includes:
- Go profiling client library
- Support for CPU, Memory, Goroutine, Mutex, and Block profiling
- HTTP and File storage backends
- Comprehensive test suite

For detailed changes, see CHANGELOG.md"

print_success "Tag $VERSION created locally"

# Push tag
print_status "Pushing tag to origin..."
git push origin "$VERSION"

print_success "Tag $VERSION pushed to origin"
print_success "Release $VERSION initiated successfully!"

echo
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Release $VERSION Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "✅ Tag created and pushed"
echo -e "✅ GitHub Actions will handle release creation"
echo -e "✅ Go module proxy will be updated automatically"
echo
echo -e "${BLUE}Monitor the release progress at:${NC}"
echo -e "https://github.com/pprofio/pprofio/actions"
echo -e "https://github.com/pprofio/pprofio/releases"
echo
echo -e "${BLUE}Package will be available via:${NC}"
echo -e "go get github.com/pprofio/pprofio@$VERSION"
echo 