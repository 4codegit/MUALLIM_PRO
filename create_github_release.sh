#!/bin/bash
# ════════════════════════════════════════════════════════════════════
# eDonish Auto — GitHub Release Uploader
# Uploads built binaries to GitHub Release
# ════════════════════════════════════════════════════════════════════
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# ── Read version dynamically from config.go ────────────────────────
VERSION="$(grep 'AppVersion' "$SCRIPT_DIR/internal/config/config.go" | grep -oP '"\K[^"]+' | head -1)"
if [ -z "$VERSION" ]; then
    echo "Error: Could not read version from config.go"
    exit 1
fi

TAG="v${VERSION}"
REPO="4codegit/edonish-auto"
GITHUB_TOKEN="${1:-$GITHUB_TOKEN}"

echo "=========================================="
echo "  eDonish Auto Release Uploader"
echo "=========================================="
echo "  Version: $VERSION"
echo "  Tag: $TAG"
echo "  Repo: $REPO"
echo ""

if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN required"
    echo "Usage: ./create_github_release.sh <token>"
    echo ""
    echo "Get a token from: https://github.com/settings/tokens"
    exit 1
fi

# Create release if it doesn't exist
echo "Checking for existing release..."
EXISTING=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
    -H "Accept: application/vnd.github.v3+json" \
    "https://api.github.com/repos/$REPO/releases/tags/$TAG" 2>/dev/null || echo "")

if [ -z "$EXISTING" ] || echo "$EXISTING" | grep -q "Not Found"; then
    echo "Creating new release: $TAG..."
    
    BODY="## Что нового в версии ${TAG}

### Новые функции:
- 🎯 **Выбор конкретного ученика**: теперь можно генерировать оценки только для выбранного ученика
- 📊 **Фильтр по ученику**: после анализа появляется выпадающий список со всеми учениками
- ⚙️ **Настройка пределов оценок**: min/max значения настраиваются в настройках

### Исправления:
- Исправлен порядок вызова функций в build.sh
- Улучшена работа с фильтрацией студентов

### Как использовать:
1. Выберите класс, предмет и четверть
2. Укажите минимальную и максимальную оценку
3. Нажмите 'Анализировать' (F5)
4. После анализа появится список учеников
5. Включите 'Только выбранного ученика' и выберите ученика
6. Нажмите 'Запустить'

📦 **Платформы:**
- Linux: бинарники для x86_64
- Windows: в разработке
- macOS: в разработке
- Android: в разработке

🔗 **Полный changelog:** https://github.com/$REPO/commits/main
"

    RESPONSE=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
         -H "Accept: application/vnd.github.v3+json" \
         -X POST "https://api.github.com/repos/$REPO/releases" \
         -d "{
           \"tag_name\": \"$TAG\",
           \"name\": \"Release $TAG\",
           \"body\": \"$BODY\",
           \"prerelease\": false,
           \"draft\": false
         }")
    
    RELEASE_ID=$(echo "$RESPONSE" | jq -r '.id')
    echo "Release created: https://github.com/$REPO/releases/tag/$TAG"
else
    echo "Release $TAG already exists."
    RELEASE_ID=$(echo "$EXISTING" | jq -r '.id')
fi

echo ""
echo "Uploading assets..."

# Upload Linux GUI binary
if [ -f "$SCRIPT_DIR/dist/linux/edonish-auto" ]; then
    echo "  Uploading edonish-auto (Linux GUI)..."
    curl -s -H "Authorization: token $GITHUB_TOKEN" \
         -H "Accept: application/vnd.github.v3+json" \
         -H "Content-Type: application/octet-stream" \
         --data-binary @"$SCRIPT_DIR/dist/linux/edonish-auto" \
         "https://uploads.github.com/repos/$REPO/releases/$RELEASE_ID/assets?name=edonish-auto-${VERSION}-linux-x86_64"
    echo "  ✅ Done"
else
    echo "  ⚠️  edonish-auto not found. Run: ./build.sh linux"
fi

# Upload Linux CLI binary
if [ -f "$SCRIPT_DIR/dist/linux/edonish-auto-cli" ]; then
    echo "  Uploading edonish-auto-cli (Linux CLI)..."
    curl -s -H "Authorization: token $GITHUB_TOKEN" \
         -H "Accept: application/vnd.github.v3+json" \
         -H "Content-Type: application/octet-stream" \
         --data-binary @"$SCRIPT_DIR/dist/linux/edonish-auto-cli" \
         "https://uploads.github.com/repos/$REPO/releases/$RELEASE_ID/assets?name=edonish-auto-cli-${VERSION}-linux-x86_64"
    echo "  ✅ Done"
else
    echo "  ⚠️  edonish-auto-cli not found. Run: ./build.sh linux"
fi

echo ""
echo "=========================================="
echo "  Release complete!"
echo "=========================================="
echo "  https://github.com/$REPO/releases/tag/$TAG"
echo ""
