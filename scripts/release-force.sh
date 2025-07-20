#!/bin/bash

# Force release script that automatically accepts linting warnings
# Use this when you need to release despite linting issues

set -e

# Get the version argument
VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.1.3"
    exit 1
fi

# Run the release script and automatically respond 'y' to the linting prompt
echo "y" | ./scripts/release.sh "$VERSION"