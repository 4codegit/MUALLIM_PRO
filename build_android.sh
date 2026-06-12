#!/bin/bash
# ════════════════════════════════════════════════════════════════════
# eDonish Auto — Android Build Script
# Requires Flet for Android
# ════════════════════════════════════════════════════════════════════

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# Read version from config.py
VERSION="$(python3 -c "from config import APP_VERSION; print(APP_VERSION)")"
echo "Building eDonish Auto for Android v${VERSION}..."

# Install dependencies
echo "Installing dependencies..."
pip install -r requirements.txt

# Build for Android using Flet
echo "Building Android APK..."
flet package android

echo ""
echo "Android build complete!"
echo "Check the Android Studio project in build/android/"
echo ""
echo "To build APK:"
echo "  1. Open build/android/ in Android Studio"
echo "  2. Build > Build Bundle(s) / APK(s) > Build APK(s)"
echo ""
echo "Output: build/android/app/build/outputs/apk/"
