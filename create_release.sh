#!/bin/bash
# ════════════════════════════════════════════════════════════════════
# eDonish Auto — GitHub Release Creator
# Reads version dynamically from config.py
# ════════════════════════════════════════════════════════════════════
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# ── Read version dynamically from config.py ────────────────────────
VERSION="$(python3 -c "import sys; sys.path.insert(0, '$SCRIPT_DIR'); from config import APP_VERSION; print(APP_VERSION)" 2>/dev/null)"
if [ -z "$VERSION" ]; then
    echo "Error: Could not read version from config.py"
    exit 1
fi
TAG="v${VERSION}"

GITHUB_TOKEN="${1:-$GITHUB_TOKEN}"
REPO="4codegit/edonish-auto"

echo "Creating release: $TAG"
echo "Repository: $REPO"

if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN required"
    echo "Usage: ./create_release.sh <token>"
    exit 1
fi

# Build release body
BODY="## What's new in ${TAG}

### Bug Fixes:
- Fixed version mismatch across Linux packages (RPM/DEB now read version from config.py dynamically)
- Fixed role permissions: can_modify_grades checks ALL user roles, not just selected one
- Fixed mobile horizontal scroll in journal table
- Removed fractional grade display (1/2, 0/2) — now shows numerator only
- Fixed desktop AppBar missing avatar with owner name, role, and role switching

### Improvements:
- Random grade button per student row in journal
- Desktop AppBar now shows avatar with name, role, and role switching
- Version numbers synced across all platforms (Linux RPM/DEB, Windows NSIS, macOS bundle)
- Build scripts (build.sh, package.sh) now read version dynamically from config.py
- Packaging scripts auto-update RPM spec version before build
- Quarter mark cell shows average and ceil(grade) on click

### Full Changelog:
- ${TAG}: Version sync across all platforms, dynamic versioning for Linux packages
- v3.19.1: Desktop avatar, mobile scroll, role permissions fix
- v3.18.7: Fixed grade saving, improved user info display
- v3.18.6: Add detailed logging for grade operations
- v3.18.5: Auto-save grades on input
- v3.18.4: Show only numeric grades (no fractions)
- v3.18.3: Fixed UI freeze
- v3.18.2: Fixed NameError
- v3.18.1: UI fixes"

curl -s -H "Authorization: token $GITHUB_TOKEN" \
     -H "Accept: application/vnd.github.v3+json" \
     -X POST "https://api.github.com/repos/$REPO/releases" \
     -d '{
       "tag_name": "'$TAG'",
       "name": "Release '$TAG'",
       "body": "'"$BODY"'",
       "prerelease": false,
       "draft": false
     }' | jq .

echo ""
echo "Release created at: https://github.com/$REPO/releases/tag/$TAG"
