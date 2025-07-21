#!/bin/bash
# Script to install git hooks for the pprofio project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${YELLOW}→${NC} $1"
}

# Get the git directory
GIT_DIR=$(git rev-parse --git-dir 2>/dev/null)
if [ $? -ne 0 ]; then
    print_error "Not in a git repository"
    exit 1
fi

HOOKS_DIR="${GIT_DIR}/hooks"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

print_info "Installing git hooks..."

# Create pre-commit hook
cat > "${HOOKS_DIR}/pre-commit" << 'EOF'
#!/bin/sh
# Pre-commit hook that runs gofmt before committing
# This ensures all Go code is properly formatted

# Get list of modified Go files
GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$')

if [ -z "$GO_FILES" ]; then
    # No Go files to check
    exit 0
fi

# Check if gofmt is installed
if ! command -v gofmt &> /dev/null; then
    echo "Error: gofmt is not installed. Please install Go."
    exit 1
fi

# Check for unformatted files
UNFORMATTED=$(gofmt -l $GO_FILES)

if [ -n "$UNFORMATTED" ]; then
    echo "Error: The following Go files are not properly formatted:"
    echo "$UNFORMATTED"
    echo ""
    echo "Please run 'gofmt -w' on these files or 'make fmt' to format all files."
    echo "You can also run 'gofmt -w $UNFORMATTED' to format only these files."
    exit 1
fi

echo "All Go files are properly formatted."
exit 0
EOF

# Make the hook executable
chmod +x "${HOOKS_DIR}/pre-commit"

print_success "Installed pre-commit hook"
print_info "The pre-commit hook will run gofmt on all staged Go files"
print_info "To bypass the hook (not recommended), use: git commit --no-verify"

echo ""
print_success "Git hooks installation completed!"