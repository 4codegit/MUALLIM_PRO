# eDonish Auto — Makefile (Go + Fyne)

.PHONY: help deps run linux windows android macos all clean

# Default target
help:
	@echo "eDonish Auto — команды:"
	@echo ""
	@echo "  make deps      — Установить Go зависимости"
	@echo "  make run       — Запустить приложение"
	@echo "  make linux     — Собрать для Linux (binary + deb + rpm)"
	@echo "  make windows   — Собрать для Windows (exe)"
	@echo "  make android   — Собрать для Android (apk)"
	@echo "  make macos     — Собрать для macOS (binary)"
	@echo "  make all       — Собрать для всех платформ"
	@echo "  make clean     — Очистить сборочные файлы"
	@echo "  make vet       — Проверить код go vet"

deps:
	go mod tidy
	go get fyne.io/fyne/v2@latest

run: deps
	go run .

linux:
	@bash build_linux.sh $(shell git describe --tags 2>/dev/null || echo "dev")

windows:
	@bash build_windows.sh $(shell git describe --tags 2>/dev/null || echo "dev")

android:
	@bash build_android_go.sh $(shell git describe --tags 2>/dev/null || echo "dev")

macos:
	@echo "🔨 Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o release/macos/edonish-auto .

all: linux windows android
	@echo "✅ All builds complete!"

vet:
	go vet ./...

clean:
	rm -rf release/ deb/ rpm/
	rm -f edonish-auto-linux edonish-auto-windows.exe edonish-auto.apk
	go clean
