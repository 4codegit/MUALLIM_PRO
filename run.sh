#!/bin/bash
# eDonish Auto - Installation and Launch Script

set -e

echo "📚 eDonish Auto - Установка и запуск"
echo "====================================="

# Check Python
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 не найден. Установите Python 3.8+"
    exit 1
fi

PYTHON_VERSION=$(python3 -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')")
echo "✅ Python $PYTHON_VERSION найден"

# Install dependencies
echo ""
echo "📦 Установка зависимостей..."
pip3 install --break-system-packages customtkinter requests 2>/dev/null || \
pip3 install customtkinter requests 2>/dev/null || \
pip install customtkinter requests

echo "✅ Зависимости установлены"

# Run the app
echo ""
echo "🚀 Запуск eDonish Auto..."
cd "$(dirname "$0")"
python3 main.py
