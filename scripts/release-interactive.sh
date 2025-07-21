#!/bin/bash

# Enhanced interactive release script for pprofio
# Provides version suggestions and streamlined release process

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    echo -e "${BLUE}ℹ ${NC}$1"
}

print_success() {
    echo -e "${GREEN}✓ ${NC}$1"
}

print_warning() {
    echo -e "${YELLOW}⚠ ${NC}$1"
}

print_error() {
    echo -e "${RED}✗ ${NC}$1"
}

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -f "pprofio.go" ]; then
    print_error "This script must be run from the pprofio project root"
    exit 1
fi

# Get the latest tag
get_latest_tag() {
    git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

# Parse semantic version
parse_version() {
    local version=$1
    # Remove 'v' prefix if present
    version=${version#v}
    
    # Extract major, minor, patch
    IFS='.' read -r major minor patch <<< "$version"
    
    # Handle pre-release versions (remove suffix)
    patch=${patch%%-*}
    
    echo "$major $minor $patch"
}

# Increment version
increment_version() {
    local current_version=$1
    local increment_type=$2
    
    read -r major minor patch <<< "$(parse_version "$current_version")"
    
    case $increment_type in
        major)
            ((major++))
            minor=0
            patch=0
            ;;
        minor)
            ((minor++))
            patch=0
            ;;
        patch)
            ((patch++))
            ;;
    esac
    
    echo "v${major}.${minor}.${patch}"
}

# Analyze commits to suggest version bump
suggest_version_bump() {
    local latest_tag=$1
    local commits_since_tag=$(git log "${latest_tag}..HEAD" --oneline 2>/dev/null | wc -l | tr -d ' ')
    
    if [ "$commits_since_tag" -eq 0 ]; then
        echo "none"
        return
    fi
    
    # Check commit messages for keywords
    local breaking_changes=$(git log "${latest_tag}..HEAD" --grep="BREAKING CHANGE" --grep="breaking:" -i --oneline | wc -l | tr -d ' ')
    local features=$(git log "${latest_tag}..HEAD" --grep="feat:" --grep="feature:" -i --oneline | wc -l | tr -d ' ')
    local fixes=$(git log "${latest_tag}..HEAD" --grep="fix:" --grep="bugfix:" -i --oneline | wc -l | tr -d ' ')
    
    if [ "$breaking_changes" -gt 0 ]; then
        echo "major"
    elif [ "$features" -gt 0 ]; then
        echo "minor"
    elif [ "$fixes" -gt 0 ]; then
        echo "patch"
    else
        echo "patch"
    fi
}

# Show commit summary
show_commit_summary() {
    local latest_tag=$1
    local commits=$(git log "${latest_tag}..HEAD" --oneline 2>/dev/null)
    
    if [ -z "$commits" ]; then
        print_info "No commits since last tag"
        return
    fi
    
    print_info "Commits since ${latest_tag}:"
    echo "$commits" | while IFS= read -r line; do
        echo "  $line"
    done | head -10
    
    local total_commits=$(echo "$commits" | wc -l | tr -d ' ')
    if [ "$total_commits" -gt 10 ]; then
        echo "  ... and $((total_commits - 10)) more"
    fi
}

# Main release process
main() {
    echo -e "${CYAN}🚀 Interactive Release Tool for pprofio${NC}"
    echo
    
    # Check git status
    if ! git diff-index --quiet HEAD --; then
        print_error "Working directory has uncommitted changes"
        print_info "Please commit or stash your changes before releasing"
        exit 1
    fi
    
    # Check current branch
    local current_branch=$(git branch --show-current)
    if [ "$current_branch" != "main" ]; then
        print_warning "You're on branch '$current_branch', not 'main'"
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Release cancelled"
            exit 0
        fi
    fi
    
    # Fetch latest tags
    print_info "Fetching latest tags from remote..."
    git fetch --tags >/dev/null 2>&1
    
    # Get current version
    local latest_tag=$(get_latest_tag)
    print_info "Latest tag: ${GREEN}${latest_tag}${NC}"
    echo
    
    # Show commit summary
    show_commit_summary "$latest_tag"
    echo
    
    # Suggest version bump
    local suggested_bump=$(suggest_version_bump "$latest_tag")
    local suggested_version=""
    
    if [ "$suggested_bump" == "none" ]; then
        print_warning "No commits since last release"
        exit 0
    fi
    
    # Calculate suggested versions
    local patch_version=$(increment_version "$latest_tag" "patch")
    local minor_version=$(increment_version "$latest_tag" "minor")
    local major_version=$(increment_version "$latest_tag" "major")
    
    # Set default based on commit analysis
    case $suggested_bump in
        major) suggested_version=$major_version ;;
        minor) suggested_version=$minor_version ;;
        patch) suggested_version=$patch_version ;;
    esac
    
    # Interactive version selection
    print_info "Version bump suggestions based on commits:"
    echo
    echo -e "  1) Patch (${patch_version}) - Bug fixes and minor changes"
    echo -e "  2) Minor (${minor_version}) - New features, backwards compatible"
    echo -e "  3) Major (${major_version}) - Breaking changes"
    echo -e "  4) Custom version"
    echo -e "  5) Cancel release"
    echo
    
    # Highlight suggested option
    if [ "$suggested_bump" != "none" ]; then
        print_info "Suggested: ${suggested_bump} (based on commit messages)"
    fi
    
    read -p "Select version option [1-5] (default: $suggested_bump): " choice
    
    # Default to suggested if no input
    if [ -z "$choice" ]; then
        case $suggested_bump in
            major) choice=3 ;;
            minor) choice=2 ;;
            patch) choice=1 ;;
        esac
    fi
    
    local new_version=""
    case $choice in
        1) new_version=$patch_version ;;
        2) new_version=$minor_version ;;
        3) new_version=$major_version ;;
        4)
            read -p "Enter custom version (e.g., v1.2.3): " new_version
            if [[ ! $new_version =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
                print_error "Invalid version format"
                exit 1
            fi
            # Ensure version starts with 'v'
            [[ $new_version != v* ]] && new_version="v$new_version"
            ;;
        5)
            print_info "Release cancelled"
            exit 0
            ;;
        *)
            print_error "Invalid choice"
            exit 1
            ;;
    esac
    
    echo
    print_info "Preparing to release: ${GREEN}${new_version}${NC}"
    echo
    
    # Run pre-release checks
    print_info "Running pre-release checks..."
    
    # Run tests
    print_info "Running tests..."
    if make test >/dev/null 2>&1; then
        print_success "Tests passed"
    else
        print_error "Tests failed"
        exit 1
    fi
    
    # Run linter
    print_info "Running linter..."
    if make lint >/dev/null 2>&1; then
        print_success "Linting passed"
    else
        print_error "Linting failed"
        exit 1
    fi
    
    # Check CHANGELOG
    if grep -q "\[Unreleased\]" CHANGELOG.md; then
        print_success "CHANGELOG.md has unreleased changes"
    else
        print_warning "No unreleased changes in CHANGELOG.md"
        read -p "Continue without changelog updates? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Please update CHANGELOG.md before releasing"
            exit 1
        fi
    fi
    
    # Update version in code
    print_info "Updating version in pprofio.go..."
    local version_without_v=${new_version#v}
    sed -i.bak "s/const Version = \".*\"/const Version = \"${version_without_v}\"/" pprofio.go && rm pprofio.go.bak
    
    # Commit version change
    git add pprofio.go
    git commit -m "chore: bump version to ${new_version}" >/dev/null 2>&1
    print_success "Version updated in code"
    
    # Final confirmation
    echo
    print_info "Ready to create release ${GREEN}${new_version}${NC}"
    echo
    echo "This will:"
    echo "  - Create an annotated git tag"
    echo "  - Push the tag to trigger GitHub Actions"
    echo "  - Create a GitHub release with changelog"
    echo
    read -p "Proceed with release? (y/N) " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_warning "Release cancelled. Rolling back version change..."
        git reset --hard HEAD~1 >/dev/null 2>&1
        exit 0
    fi
    
    # Create release
    print_info "Creating release..."
    
    # Push the version commit
    git push origin "$current_branch" >/dev/null 2>&1
    
    # Run the existing release script
    if ./scripts/release.sh "$new_version"; then
        echo
        print_success "Release ${GREEN}${new_version}${NC} created successfully! 🎉"
        echo
        print_info "Next steps:"
        echo "  - Monitor the GitHub Actions workflow"
        echo "  - Update CHANGELOG.md for next release"
        echo "  - Announce the release to users"
    else
        print_error "Release failed"
        exit 1
    fi
}

# Run main function
main "$@"