#!/bin/bash
# Pre-commit hook: update version file with current timestamp
REPO_ROOT=$(git rev-parse --show-toplevel)
VERSION_FILE="$REPO_ROOT/version"

NEW_VERSION=$(date +"%Y%m%d_%H%M%S")
echo "$NEW_VERSION" > "$VERSION_FILE"
git add "$VERSION_FILE"

echo "[pre-commit] version updated: $NEW_VERSION"
