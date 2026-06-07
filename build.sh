#!/usr/bin/env bash
# ════════════════════════════════════════════════════════════════════
# eDonish Auto — Cross-Platform Build Script
# Builds installers for: Linux (.rpm), Windows (.exe), macOS (.dmg)
# ════════════════════════════════════════════════════════════════════
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

APP_NAME="edonish-auto"
APP_TITLE="eDonish Auto"
DIST_DIR="$SCRIPT_DIR/dist"
BUILD_DIR="$SCRIPT_DIR/build"

# ── Read version dynamically from config.py ────────────────────────
VERSION="$(python3 -c "import sys; sys.path.insert(0, '$SCRIPT_DIR'); from config import APP_VERSION; print(APP_VERSION)" 2>/dev/null || echo "3.20.0")"
log "Version from config.py: $VERSION"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${CYAN}[BUILD]${NC} $*"; }
ok()   { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()  { echo -e "${RED}[ERROR]${NC} $*"; }

# ── Clean ──────────────────────────────────────────────────────────
clean() {
    log "Cleaning build artifacts..."
    rm -rf "$BUILD_DIR" "$DIST_DIR"
    ok "Clean complete"
}

# ── Linux Build ────────────────────────────────────────────────────
build_linux() {
    log "Building Linux binaries..."

    # GUI binary
    log "  Compiling GUI (edonish-auto)..."
    pyinstaller edonish-auto.spec \
        --workpath "$BUILD_DIR/gui" \
        --distpath "$DIST_DIR/linux" \
        --clean \
        --noconfirm 2>&1 | tail -5

    # CLI binary
    log "  Compiling CLI (edonish-auto-cli)..."
    pyinstaller edonish-auto-cli.spec \
        --workpath "$BUILD_DIR/cli" \
        --distpath "$DIST_DIR/linux" \
        --clean \
        --noconfirm 2>&1 | tail -5

    ok "Linux binaries built:"
    ls -lh "$DIST_DIR/linux/" 2>/dev/null || warn "No binaries found"
}

# ── RPM Package ────────────────────────────────────────────────────
build_rpm() {
    log "Building .rpm package..."

    if ! command -v fpm &>/dev/null; then
        warn "fpm not found. Installing..."
        gem install fpm 2>/dev/null || {
            err "Cannot install fpm. Install with: gem install fpm"
            err "Then run: $0 rpm"
            return 1
        }
    fi

    local BINARY="$DIST_DIR/linux/$APP_NAME"
    local CLI_BINARY="$DIST_DIR/linux/$APP_NAME-cli"

    if [[ ! -f "$BINARY" ]]; then
        err "GUI binary not found. Run '$0 linux' first."
        return 1
    fi

    local STAGING="$BUILD_DIR/rpm-staging"
    rm -rf "$STAGING"
    mkdir -p "$STAGING/usr/bin"
    mkdir -p "$STAGING/usr/share/$APP_NAME"
    mkdir -p "$STAGING/usr/share/applications"
    mkdir -p "$STAGING/usr/share/icons/hicolor/256x256/apps"

    # Copy binaries
    cp "$BINARY" "$STAGING/usr/bin/$APP_NAME"
    chmod +x "$STAGING/usr/bin/$APP_NAME"

    if [[ -f "$CLI_BINARY" ]]; then
        cp "$CLI_BINARY" "$STAGING/usr/bin/$APP_NAME-cli"
        chmod +x "$STAGING/usr/bin/$APP_NAME-cli"
    fi

    # Copy supporting files
    cp config.py "$STAGING/usr/share/$APP_NAME/"
    cp api_client.py "$STAGING/usr/share/$APP_NAME/"
    cp grade_engine.py "$STAGING/usr/share/$APP_NAME/"

    # Desktop entry
    cat > "$STAGING/usr/share/applications/$APP_NAME.desktop" << EOF
[Desktop Entry]
Name=eDonish Auto
Comment=Automated grade management for edonish.tj
Exec=$APP_NAME
Icon=$APP_NAME
Terminal=false
Type=Application
Categories=Education;Utility;
StartupNotify=true
EOF

    # Build RPM with fpm
    fpm \
        --input-type dir \
        --output-type rpm \
        --name "$APP_NAME" \
        --version "$VERSION" \
        --architecture x86_64 \
        --maintainer "Kamar Narziev <kamar@edonish.tj>" \
        --description "Automated grade management for edonish.tj electronic journal" \
        --url "https://edonish.tj" \
        --license "MIT" \
        --chdir "$STAGING" \
        --package "$DIST_DIR/rpm/" \
        .

    ok "RPM package built:"
    ls -lh "$DIST_DIR/rpm/"*.rpm 2>/dev/null
}

# ── Windows EXE (cross-compile via PyInstaller spec) ──────────────
build_windows() {
    log "Building Windows .exe..."

    # Check if we're on Windows or have mingw
    if [[ "$(uname -s)" == "MINGW"* ]] || [[ "$(uname -s)" == "MSYS"* ]] || [[ "$(uname -s)" == "CYGWIN"* ]]; then
        log "  Native Windows build..."
        pyinstaller edonish-auto.spec \
            --workpath "$BUILD_DIR/win-gui" \
            --distpath "$DIST_DIR/windows" \
            --clean --noconfirm

        pyinstaller edonish-auto-cli.spec \
            --workpath "$BUILD_DIR/win-cli" \
            --distpath "$DIST_DIR/windows" \
            --clean --noconfirm

        ok "Windows binaries built:"
        ls -lh "$DIST_DIR/windows/"*.exe 2>/dev/null
    else
        warn "Not on Windows. Creating NSIS installer script for Windows build..."
        _create_windows_installer_script
        ok "Windows installer script created at $DIST_DIR/windows/"
        log "Build on Windows: pyinstaller edonish-auto.spec && makensis installer.nsi"
    fi
}

_create_windows_installer_script() {
    mkdir -p "$DIST_DIR/windows"
    cat > "$DIST_DIR/windows/installer.nsi" << NSIS_EOF
!define APPNAME "eDonish Auto"
!define APPVERSION "${VERSION}"
!define APPEXE "edonish-auto.exe"

Name "${APPNAME} ${APPVERSION}"
OutFile "edonish-auto-${APPVERSION}-setup.exe"
InstallDir "$PROGRAMFILES\${APPNAME}"
RequestExecutionLevel admin

Section "Install"
    SetOutPath $INSTDIR
    File "dist\windows\edonish-auto.exe"
    File "dist\windows\edonish-auto-cli.exe"
    
    CreateShortCut "$DESKTOP\${APPNAME}.lnk" "$INSTDIR\${APPEXE}"
    CreateDirectory "$SMPROGRAMS\${APPNAME}"
    CreateShortCut "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk" "$INSTDIR\${APPEXE}"
    CreateShortCut "$SMPROGRAMS\${APPNAME}\Uninstall.lnk" "$INSTDIR\uninstall.exe"
    
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" \
        "DisplayName" "${APPNAME}"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}" \
        "UninstallString" "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
    Delete "$INSTDIR\*.*"
    RMDir "$INSTDIR"
    Delete "$DESKTOP\${APPNAME}.lnk"
    Delete "$SMPROGRAMS\${APPNAME}\*.*"
    RMDir "$SMPROGRAMS\${APPNAME}"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APPNAME}"
SectionEnd
NSIS_EOF
}

# ── macOS DMG ─────────────────────────────────────────────────────
build_macos() {
    log "Building macOS .dmg..."

    if [[ "$(uname -s)" == "Darwin" ]]; then
        log "  Native macOS build..."
        pyinstaller edonish-auto.spec \
            --workpath "$BUILD_DIR/macos" \
            --distpath "$DIST_DIR/macos" \
            --clean --noconfirm

        # Create DMG
        local APP_PATH="$DIST_DIR/macos/eDonish Auto.app"
        local DMG_NAME="$APP_NAME-$VERSION.dmg"
        local DMG_PATH="$DIST_DIR/dmg/$DMG_NAME"

        mkdir -p "$DIST_DIR/dmg"
        hdiutil create -volname "$APP_TITLE" \
            -srcfolder "$APP_PATH" \
            -ov -format UDZO \
            "$DMG_PATH"

        ok "macOS DMG built: $DMG_PATH"
    else
        warn "Not on macOS. Creating DMG build script..."
        _create_macos_dmg_script
        ok "macOS build script created at $DIST_DIR/macos/"
        log "Build on macOS: pyinstaller edonish-auto.spec && ./build_dmg.sh"
    fi
}

_create_macos_dmg_script() {
    mkdir -p "$DIST_DIR/macos"
    cat > "$DIST_DIR/macos/build_dmg.sh" << DMGEOF
#!/bin/bash
set -e
SCRIPT_DIR="\$(cd "\$(dirname "\$0")" && pwd)"
# Read version from config.py dynamically
VERSION="\$(python3 -c "import sys; sys.path.insert(0, '\$SCRIPT_DIR/../..'); from config import APP_VERSION; print(APP_VERSION)" 2>/dev/null || echo '${VERSION}')"
APP_NAME="edonish-auto"
APP_TITLE="eDonish Auto"

echo "Building macOS app..."
pyinstaller edonish-auto.spec --clean --noconfirm

echo "Creating DMG..."
mkdir -p dist/dmg
hdiutil create -volname "$APP_TITLE" \
    -srcfolder "dist/macos/eDonish Auto.app" \
    -ov -format UDZO \
    "dist/dmg/${APP_NAME}-${VERSION}.dmg"

echo "Done: dist/dmg/${APP_NAME}-${VERSION}.dmg"
DMGEOF
    chmod +x "$DIST_DIR/macos/build_dmg.sh"
}

# ── Build All ──────────────────────────────────────────────────────
build_all() {
    log "=========================================="
    log "  eDonish Auto v${VERSION} — Full Build"
    log "=========================================="
    log ""

    build_linux
    echo ""
    build_rpm
    echo ""
    build_windows
    echo ""
    build_macos

    echo ""
    log "=========================================="
    log "  Build Complete!"
    log "=========================================="
    log "  Output: $DIST_DIR/"
    log "  Linux:  $DIST_DIR/linux/"
    log "  RPM:    $DIST_DIR/rpm/"
    log "  Win:    $DIST_DIR/windows/"
    log "  macOS:  $DIST_DIR/macos/"
    log "=========================================="
}

# ── Main ───────────────────────────────────────────────────────────
case "${1:-all}" in
    clean)    clean ;;
    linux)    build_linux ;;
    rpm)      build_rpm ;;
    windows|win|exe)  build_windows ;;
    macos|mac|dmg)    build_macos ;;
    all)      build_all ;;
    *)
        echo "Usage: $0 {clean|linux|rpm|windows|macos|all}"
        echo ""
        echo "  clean    — Remove build artifacts"
        echo "  linux    — Build Linux binaries (GUI + CLI)"
        echo "  rpm      — Build .rpm package"
        echo "  windows  — Build Windows .exe installer"
        echo "  macos    — Build macOS .dmg"
        echo "  all      — Build everything for current platform"
        exit 1
        ;;
esac
