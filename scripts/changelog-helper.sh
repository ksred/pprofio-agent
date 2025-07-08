#!/bin/bash

# Changelog helper script for pprofio
# Helps generate changelog entries from git commits

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Get commits since last tag
get_commits_since_tag() {
    local latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    if [ -z "$latest_tag" ]; then
        git log --oneline
    else
        git log "${latest_tag}..HEAD" --oneline
    fi
}

# Categorize commits
categorize_commits() {
    local commits="$1"
    
    echo -e "${BLUE}## Changelog Entry Suggestions${NC}"
    echo
    
    # Breaking changes
    local breaking=$(echo "$commits" | grep -iE "breaking change|breaking:" || true)
    if [ -n "$breaking" ]; then
        echo -e "${YELLOW}### Breaking Changes${NC}"
        echo "$breaking" | while IFS= read -r line; do
            echo "- ${line#* }"
        done
        echo
    fi
    
    # Features
    local features=$(echo "$commits" | grep -iE "feat:|feature:" || true)
    if [ -n "$features" ]; then
        echo -e "${GREEN}### Added${NC}"
        echo "$features" | while IFS= read -r line; do
            echo "- ${line#* }"
        done
        echo
    fi
    
    # Fixes
    local fixes=$(echo "$commits" | grep -iE "fix:|bugfix:" || true)
    if [ -n "$fixes" ]; then
        echo -e "${GREEN}### Fixed${NC}"
        echo "$fixes" | while IFS= read -r line; do
            echo "- ${line#* }"
        done
        echo
    fi
    
    # Other changes
    local others=$(echo "$commits" | grep -viE "breaking change|breaking:|feat:|feature:|fix:|bugfix:" || true)
    if [ -n "$others" ]; then
        echo -e "${GREEN}### Changed${NC}"
        echo "$others" | while IFS= read -r line; do
            echo "- ${line#* }"
        done
        echo
    fi
}

# Main
main() {
    local commits=$(get_commits_since_tag)
    
    if [ -z "$commits" ]; then
        echo "No commits since last tag"
        exit 0
    fi
    
    categorize_commits "$commits"
    
    echo -e "${BLUE}Note:${NC} Review and edit these suggestions before adding to CHANGELOG.md"
    echo "Format follows: https://keepachangelog.com"
}

main "$@"