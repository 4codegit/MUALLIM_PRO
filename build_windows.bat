#!/bin/bash
# ════════════════════════════════════════════════════════════════════
# eDonish Auto — Windows Build Script
# Run on Windows (PowerShell or CMD)
# ════════════════════════════════════════════════════════════════════

echo "Building eDonish Auto for Windows..."

# Read version from config.py
for /f "tokens=*" %%i in ('python -c "from config import APP_VERSION; print(APP_VERSION)"') do set VERSION=%%i

echo "Version: %VERSION%"

# Install dependencies
echo "Installing dependencies..."
pip install -r requirements.txt

# Build with PyInstaller
echo "Building GUI..."
pyinstaller edonish-auto.spec --clean --noconfirm

echo "Building CLI..."
pyinstaller edonish-auto-cli.spec --clean --noconfirm

echo ""
echo "Windows binaries built in dist\windows\"
echo ""

# Create ZIP for distribution
echo "Creating distribution package..."
powershell -Command "Compress-Archive -Path dist\windows\*.exe -DestinationPath dist\edonish-auto-%VERSION%-windows.zip -Force"

echo "Done! Check dist\edonish-auto-%VERSION%-windows.zip"
