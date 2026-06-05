#!/usr/bin/env bash
# ════════════════════════════════════════════════════════════════════
# eDonish Auto — Package Builder (RPM + DEB without fpm)
# ════════════════════════════════════════════════════════════════════
set -euo pipefail

VERSION="3.17.0"
APP_NAME="edonish-auto"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

CYAN='\033[0;36m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

log() { echo -e "${CYAN}[PKG]${NC} $*"; }
ok()  { echo -e "${GREEN}[OK]${NC} $*"; }
err() { echo -e "${RED}[ERR]${NC} $*"; }

# ── RPM Package ────────────────────────────────────────────────────
build_rpm() {
    log "Building RPM package..."
    
    if ! command -v rpmbuild &>/dev/null; then
        err "rpmbuild not found. Install: sudo apt install rpm  OR  sudo dnf install rpm-build"
        err "Alternatively, use fpm: gem install fpm"
        return 1
    fi

    local RPMBUILD="$SCRIPT_DIR/dist/rpmbuild"
    rm -rf "$RPMBUILD"
    mkdir -p "$RPMBUILD"/{SOURCES,SPECS,BUILD,RPMS,SRPMS}

    # Create source tarball — MUST contain a top-level dir named APP_NAME-VERSION
    # because RPM %setup -q expects: cd APP_NAME-VERSION/
    local STAGING="$RPMBUILD/staging/${APP_NAME}-${VERSION}"
    mkdir -p "$STAGING/dist/linux"
    cp dist/linux/edonish-auto "$STAGING/dist/linux/"
    cp config.py api_client.py grade_engine.py "$STAGING/"
    
    cd "$RPMBUILD/staging"
    tar czf "$RPMBUILD/SOURCES/${APP_NAME}-${VERSION}.tar.gz" "${APP_NAME}-${VERSION}"
    cd "$SCRIPT_DIR"

    # Copy spec file
    cp edonish-auto.spec.rpm "$RPMBUILD/SPECS/edonish-auto.spec"

    # Build
    rpmbuild -bb \
        --define "_topdir $RPMBUILD" \
        "$RPMBUILD/SPECS/edonish-auto.spec"

    # Collect result
    mkdir -p "$SCRIPT_DIR/dist/rpm"
    cp "$RPMBUILD/RPMS/x86_64/"*.rpm "$SCRIPT_DIR/dist/rpm/" 2>/dev/null || true

    ok "RPM package built: dist/rpm/"
    ls -lh "$SCRIPT_DIR/dist/rpm/"*.rpm 2>/dev/null
}

# ── DEB Package (manual, no dpkg-deb needed) ──────────────────────
build_deb() {
    log "Building DEB package..."

    local DEB_DIR="$SCRIPT_DIR/dist/deb-staging"
    local DEB_PKG="$SCRIPT_DIR/dist/deb"
    rm -rf "$DEB_DIR" "$DEB_PKG"
    mkdir -p "$DEB_PKG"

    # Create DEB directory structure
    mkdir -p "$DEB_DIR/DEBIAN"
    mkdir -p "$DEB_DIR/usr/bin"
    mkdir -p "$DEB_DIR/usr/share/$APP_NAME"
    mkdir -p "$DEB_DIR/usr/share/applications"
    mkdir -p "$DEB_DIR/usr/share/doc/$APP_NAME"

    # Copy binary
    if [[ -f "$SCRIPT_DIR/dist/linux/edonish-auto" ]]; then
        cp "$SCRIPT_DIR/dist/linux/edonish-auto" "$DEB_DIR/usr/bin/"
        chmod 755 "$DEB_DIR/usr/bin/edonish-auto"
    fi

    # Copy shared files
    cp config.py api_client.py grade_engine.py "$DEB_DIR/usr/share/$APP_NAME/"

    # Control file
    cat > "$DEB_DIR/DEBIAN/control" << EOF
Package: edonish-auto
Version: ${VERSION}
Section: education
Priority: optional
Architecture: amd64
Depends: libx11-6, libxext6, libxrender1, libxrandr2, libxinerama1, libxcursor1, libxi6, libxtst6, libgtk-3-0
Maintainer: Kamar Narziev <kamar@edonish.tj>
Description: Automated grade management for edonish.tj
 eDonish Auto is a desktop application for automated grade
 management on the edonish.tj electronic journal system.
 Features: parallel grade engine, CLI mode, Docker support.
EOF

    # Post-install script
    cat > "$DEB_DIR/DEBIAN/postinst" << 'EOF'
#!/bin/bash
update-desktop-database /usr/share/applications/ 2>/dev/null || true
EOF
    chmod 755 "$DEB_DIR/DEBIAN/postinst"

    # Desktop entry
    cat > "$DEB_DIR/usr/share/applications/edonish-auto.desktop" << EOF
[Desktop Entry]
Name=eDonish Auto
Comment=Automated grade management for edonish.tj
Exec=edonish-auto
Icon=edonish-auto
Terminal=false
Type=Application
Categories=Education;Utility;
StartupNotify=true
EOF

    # Copyright
    cat > "$DEB_DIR/usr/share/doc/$APP_NAME/copyright" << EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: edonish-auto
Source: https://github.com/edonish-auto/edonish-auto

Files: *
Copyright: 2024-2025 Kamar Narziev
License: MIT
 Permission is hereby granted, free of charge, to any person obtaining
 a copy of this software and associated documentation files, to deal
 in the Software without restriction, including without limitation
 the rights to use, copy, modify, merge, publish, distribute,
 sublicense, and/or sell copies of the Software.
EOF

    # Build .deb
    if command -v dpkg-deb &>/dev/null; then
        dpkg-deb --build "$DEB_DIR" "$DEB_PKG/edonish-auto_${VERSION}_amd64.deb"
        ok "DEB package built:"
        ls -lh "$DEB_PKG/"*.deb
    else
        err "dpkg-deb not found. Install: sudo apt install dpkg"
        err "DEB directory prepared at: $DEB_DIR"
        log "Build manually: dpkg-deb --build $DEB_DIR edonish-auto_${VERSION}_amd64.deb"
    fi
}

# ── Main ───────────────────────────────────────────────────────────
case "${1:-all}" in
    rpm)  build_rpm ;;
    deb)  build_deb ;;
    all)  build_rpm; build_deb ;;
    *)
        echo "Usage: $0 {rpm|deb|all}"
        exit 1
        ;;
esac
