#!/bin/bash
#
# Release script for Kairo
# Usage: ./scripts/release.sh <version>
# Example: ./scripts/release.sh 2.3.0
#

set -e

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Error: Version argument required"
    echo "Usage: ./scripts/release.sh <version>"
    echo "Example: ./scripts/release.sh 2.3.0"
    exit 1
fi

# Validate semantic version format (x.y.z)
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "Error: Version must follow semantic versioning (e.g., 2.3.0)"
    exit 1
fi

DATE=$(date +%Y-%m-%d)
CHANGELOG="CHANGELOG.md"
SCRIPTS_DIR="scripts"
CHECKSUMS_FILE="$SCRIPTS_DIR/checksums.txt"

if [ ! -f "$CHANGELOG" ]; then
    echo "Error: $CHANGELOG not found in current directory"
    exit 1
fi

# Check if [Unreleased] section exists
if ! grep -q "^## \[Unreleased\]" "$CHANGELOG"; then
    echo "Error: [Unreleased] section not found in $CHANGELOG"
    echo "Please ensure your changelog has a '## [Unreleased]' section"
    exit 1
fi

# Check if Unreleased section has content
# Get the line after [Unreleased] (skip blank line) and check if it's another version header
next_section=$(grep -A 2 "^## \[Unreleased\]" "$CHANGELOG" | tail -n 1)
if echo "$next_section" | grep -q "^## \["; then
    echo "Warning: [Unreleased] section appears to be empty (no changes documented)"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 0
    fi
fi

# Create backup
cp "$CHANGELOG" "$CHANGELOG.bak"

# Update changelog: add version header after [Unreleased]
# This preserves the [Unreleased] section and adds the new version
sed -i "s/^## \[Unreleased\]/## [Unreleased]\n\n## [$VERSION] - $DATE/" "$CHANGELOG"

echo "Updated $CHANGELOG:"
echo "  - Added version [$VERSION] - $DATE"
echo "  - Preserved [Unreleased] section for future changes"
echo ""

# Generate checksums for install scripts
echo "Generating checksums for install scripts..."

# Create checksums file header
{
    echo "# Kairo release checksums"
    echo "# Generated for version $VERSION"
    echo "# DO NOT EDIT - This file is auto-generated during release"
    echo ""
} > "$CHECKSUMS_FILE"

# Add checksums for each install script
for script in "$SCRIPTS_DIR"/install.{sh,ps1}; do
    if [ -f "$script" ]; then
        sha256sum "$script" >> "$CHECKSUMS_FILE"
    fi
done

echo "  - Generated $CHECKSUMS_FILE"

echo ""
echo "Next steps:"
echo "  1. Review changes in $CHANGELOG"
echo "  2. Review $CHECKSUMS_FILE"
echo "  3. git add $CHANGELOG $CHECKSUMS_FILE"
echo "  4. git commit -m \"chore: release v$VERSION\""
echo "  5. git tag v$VERSION"
echo "  6. git push origin main --tags"
