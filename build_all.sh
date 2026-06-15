#!/bin/bash

set -e

VERSION=${1:-$(git describe --tags 2>/dev/null || echo "v0.1.0")}

echo "=========================================="
echo "  Edonish App - Multi-Platform Build"
echo "  Version: $VERSION"
echo "=========================================="

PLATFORM=${2:-all}

case $PLATFORM in
    linux)
        echo "🐧 Building for Linux..."
        bash build_linux.sh "$VERSION"
        ;;
    windows)
        echo "🪟 Building for Windows..."
        bash build_windows.sh "$VERSION"
        ;;
    android)
        echo "🤖 Building for Android..."
        bash build_android_go.sh "$VERSION"
        ;;
    all)
        echo "🔨 Building for all platforms..."
        
        echo ""
        echo "🐧 Linux build..."
        bash build_linux.sh "$VERSION" || echo "❌ Linux build failed"
        
        echo ""
        echo "🪟 Windows build..."
        bash build_windows.sh "$VERSION" || echo "❌ Windows build failed"
        
        echo ""
        echo "🤖 Android build..."
        bash build_android_go.sh "$VERSION" || echo "❌ Android build failed"
        ;;
    *)
        echo "❌ Неизвестная платформа: $PLATFORM"
        echo "Используйте: linux, windows, android, или all"
        exit 1
        ;;
esac

echo ""
echo "=========================================="
echo "  Build Summary"
echo "=========================================="
echo "📦 Все файлы находятся в release/ директории:"
ls -lh release/*/ 2>/dev/null || echo "❌ Нет собранных файлов"
