#!/bin/bash
# ════════════════════════════════════════════════════════════════════
# eDonish Auto — macOS Build Script
# Run on macOS
# ════════════════════════════════════════════════════════════════════

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# Read version from config.py
VERSION="$(python3 -c "from config import APP_VERSION; print(APP_VERSION)")"
echo "Building eDonish Auto for macOS v${VERSION}..."

# Install dependencies
echo "Installing dependencies..."
pip3 install -r requirements.txt

# Build with PyInstaller
echo "Building GUI..."
pyinstaller edonish-auto.spec --clean --noconfirm

echo "Building CLI..."
pyinstaller edonish-auto-cli.spec --clean --noconfirm

echo ""
echo "Creating DMG..."
mkdir -p dist/dmg

# Create DMG
APP_PATH="dist/macos/eDonish Auto.app"
DMG_PATH="dist/dmg/edonish-auto-${VERSION}-macos.dmg"

hdiutil create -volname "eDonish Auto" \
    -srcfolder "$APP_PATH" \
    -ov -format UDZO \
    "$DMG_PATH"

echo ""
echo "✅ macOS build complete!"
echo "DMG: $DMG_PATH"
