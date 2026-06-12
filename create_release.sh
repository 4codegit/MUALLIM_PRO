#!/bin/bash
# ════════════════════════════════════════════════════════════════════
# eDonish Auto — GitHub Release Creator (Go/Fyne version)
# Reads version dynamically from config.go
# ════════════════════════════════════════════════════════════════════
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# ── Read version dynamically from config.go ────────────────────────
VERSION="$(grep 'AppVersion' "$SCRIPT_DIR/internal/config/config.go" | grep -oP '"\K[^"]+' | head -1)"
if [ -z "$VERSION" ]; then
    echo "Error: Could not read version from config.go"
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
BODY="## Что нового в ${TAG}

### Исправления:
- Исправлен баг фильтрации четвертей в журнале — выбор конкретной четверти теперь корректно отображает данные
- Список четвертей обновляется при выборе класса (аналогично предметам)
- Добавлено логирование выбора четвертей для отладки

### Новые функции (с предыдущих версий):
- Навигация по ячейкам клавиатурой: Tab (вправо), Delete (удалить), стрелки
- Быстрый ввод оценок: цифры 1-9 и 0 (=10) на клавиатуре
- Рандомная пересдача оценок с настраиваемым диапазоном min-max
- Персональные пределы оценок для каждого ученика
- Умная система оценок на основе среднего балла ученика
- Поддержка дневных, четвертных, семестровых и годовых оценок

### Полная история:
- ${TAG}: Исправление фильтрации четвертей, обновление UI журнала
- v0.4.0: Умные оценки, навигация клавиатурой, рандом с min-max
- v0.3.0: Начальная версия Go/Fyne"

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
